// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// Note: Consul API ver was 1.12 at the time this code was written.

const (
	// consulQueryParamRecurse specifies if the lookup should be recursive and
	// the "key" treated as a prefix instead of a literal match.
	consulQueryParamRecurse = "recurse"
	// consulQueryParamDataCenter specifies the datacenter to query.
	// This will default to the datacenter of the agent being queried.
	consulQueryParamDataCenter = "dc"
	// consulQueryParamNamespace specifies the namespace to query (enterprise).
	// If not provided, the namespace will be inferred from the request's ACL token,
	// or will default to the default namespace. For recursive lookups, the namespace
	// may be specified as '*' and then results will be returned for all namespaces.
	// Added in Consul 1.7.0.
	consulQueryParamNamespace = "ns"
	// ConsulHeaderAuthToken is the header name for setting a token.
	// See also https://www.consul.io/api-docs#authentication .
	ConsulHeaderAuthToken = "X-Consul-Token"
)

const (
	// consulHTTPAddrEnvName defines an environment variable name which sets
	// the HTTP address.
	// Note: complied with official client: https://github.com/hashicorp/consul/blob/v1.12.0/api/api.go#L28
	consulHTTPAddrEnvName = "CONSUL_HTTP_ADDR"

	// consulHTTPSSLEnvName defines an environment variable name which sets
	// whether or not to use HTTPS.
	// Note: complied with official client: https://github.com/hashicorp/consul/blob/v1.12.0/api/api.go#L44
	consulHTTPSSLEnvName = "CONSUL_HTTP_SSL"
)

const consulDefaultHost = "http://127.0.0.1:8500"

// ErrConsulKeyNotFound is thrown when a Consul read key request responds with 404.
var ErrConsulKeyNotFound = errors.New("404 - Consul Key Not Found")

// newDefaultHTTPClient instantiates a new default HTTP client.
func newDefaultHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   20 * time.Second,
				KeepAlive: 20 * time.Second,
			}).DialContext,
			MaxIdleConns:          64,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			DisableKeepAlives:     false,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
	}
}

// consulKVPair holds the Key and base64 encoded blob of data.
type consulKVPair struct {
	// Key is the main key under which configs are hold.
	Key string
	// Value contains base64 encoded blob of data.
	Value string
	// ModifyIndex is the last index that modified this key
	// (here, it's used in caching).
	ModifyIndex int64
}

// ConsulLoader loads configuration from Consul Key-Value Store.
type ConsulLoader struct {
	key         string       // the key to load
	valueFormat string       // value format, one of RemoteValue* constants
	httpClient  *http.Client // the http client used for calls
	reqInfo     *requestInfo // extra request info
	cache       *consulCache // cache storage
}

// NewConsulLoader instantiates a new ConsulLoader object that loads
// configuration from Consul.
func NewConsulLoader(key string, opts ...ConsulLoaderOption) ConsulLoader {
	loader := ConsulLoader{
		key:         key,
		valueFormat: RemoteValuePlain,
		httpClient:  newDefaultHTTPClient(),
		reqInfo:     newRequestInfo(),
	}

	// apply options, if any.
	for _, opt := range opts {
		opt(&loader)
	}

	return loader
}

// Load returns a configuration key-value map from Consul KV Store, or an error
// if something bad happens along the process.
func (loader ConsulLoader) Load() (map[string]interface{}, error) {
	endpoint := loader.reqInfo.baseURL + "/v1/kv/" + loader.key

	// build the request
	req, err := buildConsulRequest(loader.reqInfo, endpoint)
	if err != nil {
		return nil, err
	}

	// do the http call
	resp, err := loader.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	// parse the response, api can respond with 200 OK, or 404 Not Found according to doc.
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrConsulKeyNotFound // isolate the 404 case with a custom error.
	}

	dec := json.NewDecoder(resp.Body)
	var kvPairs []consulKVPair
	if err := dec.Decode(&kvPairs); err != nil {
		return nil, err
	}

	return loader.kvPairsLoad(kvPairs)
}

// consulKVPairsLoad loads config from a Key's Value given the format provided.
func (loader ConsulLoader) kvPairsLoad(kvPairs []consulKVPair) (map[string]interface{}, error) {
	if configMap := loader.cache.load(kvPairs); configMap != nil {
		return configMap, nil
	}

	var (
		configMap  map[string]interface{}
		versionIDs map[string]int64
	)
	for idx, kvPair := range kvPairs {
		valueData, err := base64.StdEncoding.DecodeString(kvPair.Value)
		if err != nil {
			return nil, err // Note: this scenario should never happen, Consul server should return valid base 64 encoded data.
		}

		currentKeyConfigMap, err := getRemoteKVPairConfigMap(kvPair.Key, valueData, loader.valueFormat)
		if err != nil {
			return nil, err
		}

		if idx == 0 {
			configMap = currentKeyConfigMap
		} else {
			// merge configs from different keys.
			// Note: here, if a duplicate key exists, it will get overwritten.
			for key, value := range currentKeyConfigMap {
				configMap[key] = value
			}
		}

		// gather new ModifyIndex information.
		if loader.cache != nil {
			if versionIDs == nil {
				versionIDs = make(map[string]int64, len(kvPairs))
			}
			versionIDs[kvPair.Key] = kvPair.ModifyIndex
		}
	}

	loader.cache.save(configMap, versionIDs)

	return configMap, nil
}

// buildConsulRequest returns the http request, or an error if it could not be created.
// Query parameters and headers are set on it, if any.
func buildConsulRequest(reqInfo *requestInfo, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(reqInfo.ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	// add query params, if any
	if len(reqInfo.query) > 0 {
		q := req.URL.Query()
		for qKey, qValue := range reqInfo.query {
			q.Add(qKey, qValue)
		}
		req.URL.RawQuery = q.Encode()
	}
	// add headers, if any
	for reqHeaderKey, reqHeaderValue := range reqInfo.headers {
		req.Header.Set(reqHeaderKey, reqHeaderValue)
	}

	return req, nil
}

// closeResponseBody reads resp.Body until EOF, and then closes it. The read
// is necessary to ensure that the http.Client's underlying RoundTripper is able
// to re-use the TCP connection. See godoc on net/http Client.Do.
func closeResponseBody(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// getDefaultConsulBaseURL tries to get consul host from ENV.
// It defaults on localhost address.
func getDefaultConsulBaseURL() string {
	baseURL := consulDefaultHost

	// try to get from env variables, as in official client.
	// https://github.com/hashicorp/consul/blob/v1.12.0/api/api.go#L427
	if addr := os.Getenv(consulHTTPAddrEnvName); addr != "" {
		enabledSSL := false
		if ssl := os.Getenv(consulHTTPSSLEnvName); ssl != "" {
			enabledSSL, _ = strconv.ParseBool(ssl)
		}
		if enabledSSL {
			baseURL = "https://" + addr
		} else {
			baseURL = "http://" + addr
		}
	}

	return baseURL
}

// requestInfo is an object holding request information.
type requestInfo struct {
	baseURL string            // Consul host
	query   map[string]string // GET kv query parameters
	headers map[string]string // request's headers
	ctx     context.Context   // request's context
}

// newRequestInfo instantiates new request info object with default values.
func newRequestInfo() *requestInfo {
	return &requestInfo{
		baseURL: getDefaultConsulBaseURL(),
		ctx:     context.Background(),
		headers: map[string]string{"User-Agent": "Go-ActForGood-Xconf/1.0"},
	}
}

// setQuery stores the given query parameter.
// It lazily instantiates the query member.
func (ri *requestInfo) setQuery(qKey, qValue string) {
	if ri.query == nil {
		// we can have max 3 query params set, see consulQueryParam* constants.
		ri.query = make(map[string]string, 3)
	}
	ri.query[qKey] = qValue
}

// ConsulLoaderOption defines optional function for configuring
// a Consul Loader.
type ConsulLoaderOption func(*ConsulLoader)

// ConsulLoaderWithHTTPClient sets the http client used for calls.
// A default one is provided if you don't use this option.
func ConsulLoaderWithHTTPClient(client *http.Client) ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		loader.httpClient = client
	}
}

// ConsulLoaderWithHost sets Consul's base url.
// By default, is set to "http://127.0.0.1:8500".
// Consul host can also be set through CONSUL_HTTP_ADDR and CONSUL_HTTP_SSL
// ENV as in official hashicorp's client.
// Example:
//		xconf.ConsulLoaderWithHost("http://consul.example.com:8500")
func ConsulLoaderWithHost(host string) ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		loader.reqInfo.baseURL = host
	}
}

// ConsulLoaderWithContext sets request 's context.
// By default, a context.Background() is used.
func ConsulLoaderWithContext(ctx context.Context) ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		loader.reqInfo.ctx = ctx
	}
}

// ConsulLoaderWithQueryDataCenter specifies the datacenter to query.
// This will default to the datacenter of the agent being queried.
// See also official doc https://www.consul.io/api-docs/kv#read-key .
// Example:
//		xconf.ConsulLoaderWithQueryDataCenter("my-dc")
func ConsulLoaderWithQueryDataCenter(dc string) ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		loader.reqInfo.setQuery(consulQueryParamDataCenter, dc)
	}
}

// ConsulLoaderWithQueryNamespace specifies the namespace to query (enterprise).
// If not provided, the namespace will be inferred from the request's ACL token,
// or will default to the default namespace. For recursive lookups, the namespace
// may be specified as '*' and then results will be returned for all namespaces.
// Added in Consul 1.7.0.
// See also official doc https://www.consul.io/api-docs/kv#read-key .
// Example:
//		xconf.ConsulLoaderWithQueryNamespace("my-ns")
func ConsulLoaderWithQueryNamespace(ns string) ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		loader.reqInfo.setQuery(consulQueryParamNamespace, ns)
	}
}

// ConsulLoaderWithPrefix specifies if the lookup should be recursive and
// the "key" treated as a prefix instead of a literal match.
func ConsulLoaderWithPrefix() ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		loader.reqInfo.setQuery(consulQueryParamRecurse, "")
	}
}

// ConsulLoaderWithRequestHeader adds a request header.
// You can set the auth token for example:
// 		xconf.ConsulLoaderWithRequestHeader(xconf.ConsulHeaderAuthToken, "someSecretToken")
// or some basic auth header:
// 		xconf.ConsulLoaderWithRequestHeader("Authorization", "Basic " + base64.StdEncoding.EncodeToString([]byte(usr + ":" + pwd))
//
func ConsulLoaderWithRequestHeader(hName, hValue string) ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		loader.reqInfo.headers[hName] = hValue
	}
}

// ConsulLoaderWithCache enables cache.
func ConsulLoaderWithCache() ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		loader.cache = new(consulCache)
	}
}

// ConsulLoaderWithValueFormat sets the value format for a key.
//
// If is set to RemoteValueJSON, the key's value will be treated as JSON and configuration will be loaded from it.
//
// If is set to RemoteValueYAML, the key's value will be treated as YAML and configuration will be loaded from it.
//
// If is set to RemoteValuePlain, the key's value will be treated as plain content and configuration will contain the key and its plain value.
//
// By default, is set to RemoteValuePlain.
func ConsulLoaderWithValueFormat(valueFormat string) ConsulLoaderOption {
	return func(loader *ConsulLoader) {
		if valueFormat == RemoteValueJSON ||
			valueFormat == RemoteValueYAML ||
			valueFormat == RemoteValuePlain {
			loader.valueFormat = valueFormat
		}
	}
}

// consulCache holds caching info.
type consulCache struct {
	configMap  map[string]interface{} // cached config map.
	versionIDs map[string]int64       // map of key and its version ID.
	mu         sync.RWMutex           // concurrency semaphore
}

// save stores configuration key-value map and the key-version map.
func (cache *consulCache) save(configMap map[string]interface{}, versionIDs map[string]int64) {
	if cache == nil { // cache is optional on loaders.
		return
	}
	cache.mu.Lock()
	cache.configMap = DeepCopyConfigMap(configMap)
	cache.versionIDs = versionIDs
	cache.mu.Unlock()
}

// load retrieves configuration key-value map comparing each key's version ID.
// If a single key's version ID mismatches, nil is returned, meaning configuration
// map should be loaded from original source.
func (cache *consulCache) load(kvPairs []consulKVPair) map[string]interface{} {
	if cache == nil { // cache is optional on loaders.
		return nil
	}
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	kvPairsLen := len(kvPairs)
	if kvPairsLen == 0 || kvPairsLen != len(cache.versionIDs) {
		return nil
	}

	for _, kvPair := range kvPairs {
		if kvPair.ModifyIndex != cache.versionIDs[kvPair.Key] {
			return nil
		}
	}

	// return a copy not to modify this state from outside (for example from a decorator,
	// which usually modifies directly the original returned configuration map reference
	// - for performance reasons, so we ensure from this stateful loader that we return a
	// new configuration map each time)
	return DeepCopyConfigMap(cache.configMap)
}
