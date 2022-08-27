// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/actforgood/xerr"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Note: Etcd API ver was 3.5 at the time this code was written.
// API ref: https://etcd.io/docs/v3.5/learning/api/ .

const (
	etcdDefaultEndpoint = "127.0.0.1:2379"

	// etcdEndpointsEnvName defines an environment variable name which sets
	// the Etcd endpoints, comma separated.
	etcdEndpointsEnvName = "ETCD_ENDPOINTS"
)

// EtcdLoader loads configuration from etcd.
// Close it if watcher option is enabled, in order to properly release resources.
type EtcdLoader struct {
	strategyInfo *etcdStrategyInfo // common strategy info.
	strategy     Loader            // delegated strategy loader.
}

// NewEtcdLoader instantiates a new EtcdLoader object that loads
// configuration from etcd.
func NewEtcdLoader(key string, opts ...EtcdLoaderOption) EtcdLoader {
	loader := EtcdLoader{
		strategyInfo: &etcdStrategyInfo{
			key:         key,
			valueFormat: RemoteValuePlain,
			ctx:         context.Background(),
			clientCfg:   clientv3.Config{DialTimeout: 10 * time.Second},
		},
	}

	// apply options, if any.
	for _, opt := range opts {
		opt(&loader)
	}
	if loader.strategyInfo.clientCfg.Endpoints == nil {
		loader.strategyInfo.clientCfg.Endpoints = getDefaultEtcdEndpoints()
	}
	if loader.strategy == nil {
		loader.strategy = etcdSimpleLoadStrategy{info: loader.strategyInfo}
	}

	return loader
}

// Load returns a configuration key-value map from etcd, or an error
// if something bad happens along the process.
func (loader EtcdLoader) Load() (map[string]interface{}, error) {
	return loader.strategy.Load()
}

// Close needs to be called in case watch key changes were enabled.
// It releases associated resources.
func (loader EtcdLoader) Close() error {
	if closeableStrategy, ok := loader.strategy.(io.Closer); ok {
		return closeableStrategy.Close()
	}

	return nil
}

// getDefaultEtcdEndpoints tries to get etcd endpoints from ENV.
// It defaults on localhost address.
func getDefaultEtcdEndpoints() []string {
	endpoints := []string{etcdDefaultEndpoint}

	// try to get from env variables
	if eps := os.Getenv(etcdEndpointsEnvName); eps != "" {
		endpoints = strings.Split(eps, ",")
	}

	return endpoints
}

// EtcdLoaderOption defines optional function for configuring
// an Etcd Loader.
type EtcdLoaderOption func(*EtcdLoader)

// EtcdLoaderWithEndpoints sets the etcd host(s) for the client.
// By default, is set to "127.0.0.1:2379".
// Etcd hosts can also be set through ETCD_ENDPOINTS ENV
// (comma separated, if there is more than 1 ep).
func EtcdLoaderWithEndpoints(endpoints []string) EtcdLoaderOption {
	return func(loader *EtcdLoader) {
		loader.strategyInfo.clientCfg.Endpoints = endpoints
	}
}

// EtcdLoaderWithPrefix sets the WithPrefix() option on etcd client.
// The loaded key will be treated as a prefix, and thus all the keys
// having that prefix will be returned.
func EtcdLoaderWithPrefix() EtcdLoaderOption {
	return func(loader *EtcdLoader) {
		loader.strategyInfo.clientOpOpts = []clientv3.OpOption{clientv3.WithPrefix()}
	}
}

// EtcdLoaderWithContext sets request's context.
// By default, a context.Background() is used.
func EtcdLoaderWithContext(ctx context.Context) EtcdLoaderOption {
	return func(loader *EtcdLoader) {
		loader.strategyInfo.ctx = ctx
		loader.strategyInfo.clientCfg.Context = ctx
	}
}

// EtcdLoaderWithAuth sets the authentication username and password.
func EtcdLoaderWithAuth(username, pwd string) EtcdLoaderOption {
	return func(loader *EtcdLoader) {
		loader.strategyInfo.clientCfg.Username = username
		loader.strategyInfo.clientCfg.Password = pwd
	}
}

// EtcdLoaderWithValueFormat sets the value format for a key.
//
// If is set to RemoteValueJSON, the key's value will be treated as JSON and configuration will be loaded from it.
//
// If is set to RemoteValueYAML, the key's value will be treated as YAML and configuration will be loaded from it.
//
// If is set to RemoteValuePlain, the key's value will be treated as plain content and configuration will contain the key and its plain value.
//
// By default, is set to RemoteValuePlain.
func EtcdLoaderWithValueFormat(valueFormat string) EtcdLoaderOption {
	return func(loader *EtcdLoader) {
		if valueFormat == RemoteValueJSON ||
			valueFormat == RemoteValueYAML ||
			valueFormat == RemoteValuePlain {
			loader.strategyInfo.valueFormat = valueFormat
		}
	}
}

// EtcdLoaderWithWatcher enables watch for keys changes.
// Use this if you intend to load configuration intensively, multiple times.
// If you plan to load configuration only once, or rarely, don't use this feature.
// If you use this feature, call Close() method on the loader to gracefully release resources
// (at your application shutdown).
func EtcdLoaderWithWatcher() EtcdLoaderOption {
	return func(loader *EtcdLoader) {
		loader.strategy = &etcdWatcherLoadStrategy{
			info: loader.strategyInfo,
		}
	}
}

// etcdStrategyInfo holds common info needed for strategies.
type etcdStrategyInfo struct {
	key          string              // the key to load
	valueFormat  string              // value format, one of RemoteValue* constants
	clientCfg    clientv3.Config     // client config
	clientOpOpts []clientv3.OpOption // client operation options
	ctx          context.Context     // request context
}

// etcdSimpleLoadStrategy loads configuration
// by making a grpc call.
type etcdSimpleLoadStrategy struct {
	info *etcdStrategyInfo
}

// Load retrieves configuration by a simple client call.
func (loaderStrategy etcdSimpleLoadStrategy) Load() (map[string]interface{}, error) {
	cli, err := clientv3.New(loaderStrategy.info.clientCfg)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	resp, err := cli.KV.Get(
		loaderStrategy.info.ctx,
		loaderStrategy.info.key,
		loaderStrategy.info.clientOpOpts...,
	)
	if err != nil {
		return nil, err
	}

	return etcdKVPairsLoad(resp.Kvs, loaderStrategy.info.valueFormat)
}

// etcdKVPairsLoad loads config from a Key's Value given the format provided.
func etcdKVPairsLoad(kvPairs []*mvccpb.KeyValue, format string) (map[string]interface{}, error) {
	var configMap map[string]interface{}
	for idx, kvPair := range kvPairs {
		currentKeyConfigMap, err := getRemoteKVPairConfigMap(
			string(kvPair.Key),
			kvPair.Value,
			format,
		)
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
	}

	return configMap, nil
}

// etcdWatcherLoadStrategy loads initial configuration
// by making a grpc call, and after that listens for
// key changes asynchronously.
type etcdWatcherLoadStrategy struct {
	info      *etcdStrategyInfo
	configMap map[string]interface{} // "live" configuration map
	client    *clientv3.Client       // underlying client
	mErr      *xerr.MultiError       // error(s) occurred during watching, between 2 Loads.
	mu        sync.RWMutex           // concurrency semaphore
	wg        sync.WaitGroup         // wait group to wait for watching goroutine to finish
}

// Load returns a copy of the stored configuration map,
// or an error if something bad happens along the process.
func (loaderStrategy *etcdWatcherLoadStrategy) Load() (map[string]interface{}, error) {
	if err := loaderStrategy.init(); err != nil {
		return nil, err
	}

	loaderStrategy.mu.RLock()
	configMap := DeepCopyConfigMap(loaderStrategy.configMap)
	err := loaderStrategy.mErr.ErrOrNil()
	loaderStrategy.mErr.Reset()
	loaderStrategy.mu.RUnlock()

	return configMap, err
}

// init initializes the client, populates initial configuration map
// and starts watching for keys changes.
func (loaderStrategy *etcdWatcherLoadStrategy) init() error {
	loaderStrategy.mu.Lock()
	defer loaderStrategy.mu.Unlock()

	if loaderStrategy.client == nil {
		cli, err := clientv3.New(loaderStrategy.info.clientCfg)
		if err != nil {
			return err
		}
		loaderStrategy.client = cli

		// populate config for the first time.
		resp, err := cli.KV.Get(
			loaderStrategy.info.ctx,
			loaderStrategy.info.key,
			loaderStrategy.info.clientOpOpts...,
		)
		if err != nil {
			return err
		}
		configMap, err := etcdKVPairsLoad(resp.Kvs, loaderStrategy.info.valueFormat)
		if err != nil {
			return err
		}
		loaderStrategy.configMap = configMap

		// listen for changes.
		loaderStrategy.wg.Add(1)
		go loaderStrategy.watchKeysAsync()
	}

	return nil
}

// watchKeysAsync listens for key(s) changes.
func (loaderStrategy *etcdWatcherLoadStrategy) watchKeysAsync() {
	defer loaderStrategy.wg.Done()

	watchChan := loaderStrategy.client.Watch(
		loaderStrategy.info.ctx,
		loaderStrategy.info.key,
		loaderStrategy.info.clientOpOpts...,
	)
	for entry := range watchChan {
		if entry.Canceled {
			continue
		}
		for _, event := range entry.Events {
			kvPair := event.Kv
			if event.Type == mvccpb.DELETE { // key was deleted.
				loaderStrategy.mu.Lock()
				delete(loaderStrategy.configMap, string(kvPair.Key))
				loaderStrategy.mu.Unlock()

				continue
			}

			// key was created/modified.
			currentKeyConfigMap, err := getRemoteKVPairConfigMap(
				string(kvPair.Key),
				kvPair.Value,
				loaderStrategy.info.valueFormat,
			)
			loaderStrategy.mu.Lock()
			if err != nil {
				loaderStrategy.mErr = loaderStrategy.mErr.Add(err)
			} else {
				// merge configs from different keys.
				for key, value := range currentKeyConfigMap {
					loaderStrategy.configMap[key] = value
				}
			}
			loaderStrategy.mu.Unlock()
		}
	}
}

// Close closes the underlying client connection.
func (loaderStrategy *etcdWatcherLoadStrategy) Close() error {
	loaderStrategy.mu.RLock()
	defer loaderStrategy.mu.RUnlock()

	if loaderStrategy.client != nil {
		err := loaderStrategy.client.Close()
		loaderStrategy.wg.Wait()

		return err
	}

	return nil
}
