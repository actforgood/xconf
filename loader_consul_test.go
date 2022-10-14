// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/actforgood/xconf"
	"gopkg.in/yaml.v3"
)

var consulResponseContent = map[string]map[bool]string{
	xconf.RemoteValueJSON: {
		true: `[
			{
				"LockIndex": 0,
				"Key": "consul_json_key",
				"Flags": 0,
				"Value": "ewogICJjb25zdWxfanNvbl9mb28iOiAiYmFyIiwKICAiY29uc3VsX2pzb25feWVhciI6IDIwMjIsCiAgImNvbnN1bF9qc29uX3RlbXBlcmF0dXJlIjogMzcuNSwKICAiY29uc3VsX2pzb25fc2hvcHBpbmdfbGlzdCI6IFsiYnJlYWQiLCAibWlsayIsICJlZ2dzIl0KfQ==",
				"CreateIndex": 20,
				"ModifyIndex": 20
			},
			{
				"LockIndex": 0,
				"Key": "consul_json_key/subkey",
				"Flags": 0,
				"Value": "ewogICJjb25zdWxfanNvbl9hYmMiOiJ4eXoiCn0=",
				"CreateIndex": 26,
				"ModifyIndex": 68
			}
		]`,
		false: `[
			{
				"LockIndex": 0,
				"Key": "consul_json_key",
				"Flags": 0,
				"Value": "ewogICJjb25zdWxfanNvbl9mb28iOiAiYmFyIiwKICAiY29uc3VsX2pzb25feWVhciI6IDIwMjIsCiAgImNvbnN1bF9qc29uX3RlbXBlcmF0dXJlIjogMzcuNSwKICAiY29uc3VsX2pzb25fc2hvcHBpbmdfbGlzdCI6IFsiYnJlYWQiLCAibWlsayIsICJlZ2dzIl0KfQ==",
				"CreateIndex": 20,
				"ModifyIndex": 20
			}
		]`,
	},
	xconf.RemoteValueYAML: {
		true: `[
			{
				"LockIndex": 0,
				"Key": "consul_yaml_key",
				"Flags": 0,
				"Value": "LS0tCmNvbnN1bF95YW1sX2ZvbzogYmFyCmNvbnN1bF95YW1sX3llYXI6IDIwMjIKY29uc3VsX3lhbWxfdGVtcGVyYXR1cmU6IDM3LjUKY29uc3VsX3lhbWxfc2hvcHBpbmdfbGlzdDoKICAtIGJyZWFkCiAgLSBtaWxrCiAgLSBlZ2dz",
				"CreateIndex": 20,
				"ModifyIndex": 20
			},
			{
				"LockIndex": 0,
				"Key": "consul_yaml_key/subkey",
				"Flags": 0,
				"Value": "LS0tCmNvbnN1bF95YW1sX2FiYzogeHl6",
				"CreateIndex": 26,
				"ModifyIndex": 68
			}
		]`,
		false: `[
			{
				"LockIndex": 0,
				"Key": "consul_yaml_key",
				"Flags": 0,
				"Value": "LS0tCmNvbnN1bF95YW1sX2ZvbzogYmFyCmNvbnN1bF95YW1sX3llYXI6IDIwMjIKY29uc3VsX3lhbWxfdGVtcGVyYXR1cmU6IDM3LjUKY29uc3VsX3lhbWxfc2hvcHBpbmdfbGlzdDoKICAtIGJyZWFkCiAgLSBtaWxrCiAgLSBlZ2dz",
				"CreateIndex": 20,
				"ModifyIndex": 20
			}
		]`,
	},
	xconf.RemoteValuePlain: {
		true: `[
			{
				"LockIndex": 0,
				"Key": "consul_plain_key",
				"Flags": 0,
				"Value": "MTAwMA==",
				"CreateIndex": 20,
				"ModifyIndex": 20
			},
			{
				"LockIndex": 0,
				"Key": "consul_plain_key/subkey",
				"Flags": 0,
				"Value": "eHl6",
				"CreateIndex": 26,
				"ModifyIndex": 68
			}
		]`,
		false: `[
			{
				"LockIndex": 0,
				"Key": "consul_plain_key",
				"Flags": 0,
				"Value": "MTAwMA==",
				"CreateIndex": 20,
				"ModifyIndex": 20
			}
		]`,
	},
}

var consulKeys = map[string]string{
	xconf.RemoteValueJSON:  "consul_json_key",
	xconf.RemoteValueYAML:  "consul_yaml_key",
	xconf.RemoteValuePlain: "consul_plain_key",
}

func TestConsulLoader(t *testing.T) {
	// Note: do not run this test with t.Parallel() as it can affect others by setting ENVs.

	t.Run("success - json single key", testConsulLoaderByFormatAndPrefix(xconf.RemoteValueJSON, false))
	t.Run("success - json prefix key", testConsulLoaderByFormatAndPrefix(xconf.RemoteValueJSON, true))
	t.Run("success - yaml single key", testConsulLoaderByFormatAndPrefix(xconf.RemoteValueYAML, false))
	t.Run("success - yaml prefix key", testConsulLoaderByFormatAndPrefix(xconf.RemoteValueYAML, true))
	t.Run("success - plain single key", testConsulLoaderByFormatAndPrefix(xconf.RemoteValuePlain, false))
	t.Run("success - plain prefix key", testConsulLoaderByFormatAndPrefix(xconf.RemoteValuePlain, true))
	t.Run("error - key is not found", testConsulLoaderReturnsErrWhenKeyIsNotFound)
	t.Run("error - http call fails", testConsulLoaderReturnsErrFromHTTPCall)
	t.Run("error - cannot build request", testConsulLoaderReturnsErrFromBuildRequest)
	t.Run("error - response deserialization fails", testConsulLoaderReturnsErrFromJSONResponseDeserialization)
	t.Run("error - value base64 decoding fails", testConsulLoaderReturnsErrFromValueBase64Decoding)
	t.Run("error - json value deserialization fails", testConsulLoaderReturnsErrFromJSONValueDeserialization)
	t.Run("error - yaml value deserialization fails", testConsulLoaderReturnsErrFromYAMLValueDeserialization)
	t.Run("success - query and headers set on request", testConsulLoaderRequestQueryAndHeaders)
	t.Run("success - default consul url taken from env", testConsulLoaderWithBaseURLTakenFromEnv)
	t.Run("success - caching works", testConsulLoaderWithCache)
	t.Run("success - safe-mutable config map", testConsulLoaderReturnsSafeMutableConfigMap)
}

func testConsulLoaderByFormatAndPrefix(format string, withPrefix bool) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		key := consulKeys[format]
		content := consulResponseContent[format][withPrefix]
		svr := startConsulKVMockServer(t, key, content, withPrefix)
		defer svr.Close()
		opts := []xconf.ConsulLoaderOption{
			xconf.ConsulLoaderWithHost(svr.URL),
			xconf.ConsulLoaderWithValueFormat(format),
		}
		if withPrefix {
			opts = append(opts, xconf.ConsulLoaderWithPrefix())
		}
		subject := xconf.NewConsulLoader(key, opts...)

		// act
		config, err := subject.Load()

		// assert
		assertNil(t, err)
		assertEqual(t, getConsulExpectedConfigMapByFormatAndPrefix(format, withPrefix), config)
	}
}

func testConsulLoaderReturnsErrWhenKeyIsNotFound(t *testing.T) {
	t.Parallel()

	// arrange
	key := "consul_not_found_key"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "/v1/kv/"+key, r.URL.String())
		w.WriteHeader(http.StatusNotFound)
	}))
	defer svr.Close()
	subject := xconf.NewConsulLoader(key, xconf.ConsulLoaderWithHost(svr.URL))

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, errors.Is(err, xconf.ErrConsulKeyNotFound))
}

func testConsulLoaderReturnsErrFromHTTPCall(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.NewConsulLoader(
		"some-key",
		xconf.ConsulLoaderWithHost("http://127.0.0.1:12345"),
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var target *net.OpError
	assertTrue(t, errors.As(err, &target))
}

func testConsulLoaderReturnsErrFromBuildRequest(t *testing.T) {
	t.Parallel()

	// arrange
	var ctx context.Context
	subject := xconf.NewConsulLoader(
		"some-key",
		xconf.ConsulLoaderWithContext(ctx),
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	if assertNotNil(t, err) {
		assertTrue(t, strings.Contains(err.Error(), "nil Context"))
	}
}

func testConsulLoaderReturnsErrFromJSONResponseDeserialization(t *testing.T) {
	t.Parallel()

	// Note: this scenario should never happen in theory, server should always
	// respond with valid json content in case of a 200 OK.

	// arrange
	content := `[{ corrupted json`
	key := "some-key"
	svr := startConsulKVMockServer(t, key, content, false)
	defer svr.Close()
	subject := xconf.NewConsulLoader(
		key,
		xconf.ConsulLoaderWithHost(svr.URL),
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var jsonErr *json.SyntaxError
	assertTrue(t, errors.As(err, &jsonErr))
}

func testConsulLoaderReturnsErrFromValueBase64Decoding(t *testing.T) {
	t.Parallel()

	// Note: this scenario should never happen in theory, server should always
	// respond with valid content in case of a 200 OK.

	// arrange
	content := `[
		{
			"LockIndex": 0,
			"Key": "some-key",
			"Flags": 0,
			"Value": "invalid-base64-data",
			"CreateIndex": 20,
			"ModifyIndex": 20
		}
	]`
	key := "some-key"
	svr := startConsulKVMockServer(t, key, content, false)
	defer svr.Close()
	subject := xconf.NewConsulLoader(
		key,
		xconf.ConsulLoaderWithHost(svr.URL),
		xconf.ConsulLoaderWithHTTPClient(http.DefaultClient),
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	if assertNotNil(t, err) {
		assertTrue(t, strings.Contains(err.Error(), "base64 data"))
	}
}

func testConsulLoaderReturnsErrFromJSONValueDeserialization(t *testing.T) {
	t.Parallel()

	// arrange
	format := xconf.RemoteValueJSON
	withPrefix := true
	corruptedJSONPatch := "W3sgY29ycnVwdGVkIGpzb24=" // base64 encoded "[{ corrupted json" value
	content := strings.Replace(
		consulResponseContent[format][withPrefix],
		"ewogICJjb25zdWxfanNvbl9hYmMiOiJ4eXoiCn0=",
		corruptedJSONPatch,
		1,
	)
	key := consulKeys[format]
	svr := startConsulKVMockServer(t, key, content, withPrefix)
	defer svr.Close()
	subject := xconf.NewConsulLoader(
		key,
		xconf.ConsulLoaderWithHost(svr.URL),
		xconf.ConsulLoaderWithPrefix(),
		xconf.ConsulLoaderWithValueFormat(format),
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var jsonErr *json.SyntaxError
	assertTrue(t, errors.As(err, &jsonErr))
}

func testConsulLoaderReturnsErrFromYAMLValueDeserialization(t *testing.T) {
	t.Parallel()

	// arrange
	format := xconf.RemoteValueYAML
	withPrefix := true
	corruptedYAMLPatch := "LS0tXG5pbnZhbGlkXG4gIHlhbWwgY29udGVudA==" // base64 encoded "---\ninvalid\n  yaml content" value
	content := strings.Replace(
		consulResponseContent[format][withPrefix],
		"LS0tCmNvbnN1bF95YW1sX2FiYzogeHl6",
		corruptedYAMLPatch,
		1,
	)
	key := consulKeys[format]
	svr := startConsulKVMockServer(t, key, content, withPrefix)
	defer svr.Close()
	subject := xconf.NewConsulLoader(
		key,
		xconf.ConsulLoaderWithHost(svr.URL),
		xconf.ConsulLoaderWithPrefix(),
		xconf.ConsulLoaderWithValueFormat(format),
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var yamlErr *yaml.TypeError
	assertTrue(t, errors.As(err, &yamlErr))
}

func testConsulLoaderRequestQueryAndHeaders(t *testing.T) {
	t.Parallel()

	// arrange
	format := xconf.RemoteValuePlain
	withPrefix := false
	content := consulResponseContent[format][withPrefix]
	key := consulKeys[format]
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// assert
		assertEqual(t, "some-test-dc", r.URL.Query().Get("dc"))
		assertEqual(t, "some-test-ns", r.URL.Query().Get("ns"))
		assertEqual(t, "some-test-auth-token", r.Header.Get("X-Consul-Token"))
		assertEqual(t, "some-test-custom-header-value", r.Header.Get("X-Test-Custom-Header"))
		assertEqual(t, "Go-ActForGood-Xconf/1.0", r.Header.Get("User-Agent"))

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, content)
	}))
	defer svr.Close()
	subject := xconf.NewConsulLoader(
		key,
		xconf.ConsulLoaderWithHost(svr.URL),
		xconf.ConsulLoaderWithQueryDataCenter("some-test-dc"),
		xconf.ConsulLoaderWithQueryNamespace("some-test-ns"),
		xconf.ConsulLoaderWithRequestHeader(xconf.ConsulHeaderAuthToken, "some-test-auth-token"),
		xconf.ConsulLoaderWithRequestHeader("X-Test-Custom-Header", "some-test-custom-header-value"),
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, getConsulExpectedConfigMapByFormatAndPrefix(format, withPrefix), config)
}

func testConsulLoaderWithBaseURLTakenFromEnv(t *testing.T) {
	// arrange
	format := xconf.RemoteValuePlain
	withPrefix := false
	content := consulResponseContent[format][withPrefix]
	key := consulKeys[format]

	svr := startConsulKVMockServer(t, key, content, withPrefix)
	defer svr.Close()

	t.Setenv("CONSUL_HTTP_ADDR", strings.TrimPrefix(svr.URL, "http://"))
	t.Setenv("CONSUL_HTTP_SSL", strings.TrimPrefix(svr.URL, "false"))

	subject := xconf.NewConsulLoader(key)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, getConsulExpectedConfigMapByFormatAndPrefix(format, withPrefix), config)
}

func testConsulLoaderWithCache(t *testing.T) {
	t.Parallel()

	// arrange
	format := xconf.RemoteValueJSON
	withPrefix := true

	// apply this patch with a broken value, there is no other way to test that cache works,
	// but to see that no error is returned on this broken content call.
	brokenContent := strings.Replace(
		consulResponseContent[format][withPrefix],
		"ewogICJjb25zdWxfanNvbl9hYmMiOiJ4eXoiCn0=",
		"broken",
		1,
	)
	freshContent := strings.Replace(
		consulResponseContent[format][withPrefix],
		`"ModifyIndex": 68`,
		`"ModifyIndex": 100`, // change the modify index
		1,
	)
	freshContent = strings.Replace(
		freshContent,
		`ewogICJjb25zdWxfanNvbl9hYmMiOiJ4eXoiCn0=`,
		`ewogICJjb25zdWxfanNvbl9hYmMiOiJBQkMiCn0=`, // change the content to {"consul_json_abc":"ABC"}
		1,
	)
	contentByCall := []string{
		// 1st call response
		consulResponseContent[format][withPrefix],
		// 2nd call response
		brokenContent,
		// 3rd call response
		brokenContent,
		// 4th call response
		freshContent,
	}

	serverCallsCnt := 0
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if serverCallsCnt < len(contentByCall) {
			_, _ = fmt.Fprintln(w, contentByCall[serverCallsCnt])
		}
		serverCallsCnt++
	}))
	defer svr.Close()
	subject := xconf.NewConsulLoader(
		"consul_json_key",
		xconf.ConsulLoaderWithHost(svr.URL),
		xconf.ConsulLoaderWithPrefix(),
		xconf.ConsulLoaderWithCache(),
		xconf.ConsulLoaderWithValueFormat(format),
	)
	expectedConfigMap := getConsulExpectedConfigMapByFormatAndPrefix(format, withPrefix)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfigMap, config)
	assertEqual(t, 1, serverCallsCnt)

	for i := 0; i < 2; i++ {
		// act - now config should be taken from cache.
		config, err = subject.Load()

		// assert
		assertNil(t, err) // no error is returned, server returns broken content, cache works!
		assertEqual(t, expectedConfigMap, config)
	}

	// act - now config should be refreshed.
	config, err = subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"consul_json_foo":           "bar",
			"consul_json_year":          float64(2022),
			"consul_json_temperature":   37.5,
			"consul_json_shopping_list": []interface{}{"bread", "milk", "eggs"},
			"consul_json_abc":           "ABC", // this was updated
		},
		config,
	)
	assertEqual(t, 4, serverCallsCnt)
}

func testConsulLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	format := xconf.RemoteValueYAML
	withPrefix := true
	key := consulKeys[format]
	content := consulResponseContent[format][withPrefix]
	svr := startConsulKVMockServer(t, key, content, withPrefix)
	defer svr.Close()
	subject := xconf.NewConsulLoader(
		key,
		xconf.ConsulLoaderWithHost(svr.URL),
		xconf.ConsulLoaderWithValueFormat(format),
		xconf.ConsulLoaderWithPrefix(),
		xconf.ConsulLoaderWithCache(),
	)
	expectedConfig := getConsulExpectedConfigMapByFormatAndPrefix(format, withPrefix)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["consul_yaml_int"] = 7777
	config1["consul_yaml_foo"] = "modified consul string"
	config1["consul_yaml_shopping_list"].([]interface{})[0] = "modified consul slice"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]interface{}{
			"consul_yaml_foo":           "bar",
			"consul_yaml_year":          2022,
			"consul_yaml_temperature":   37.5,
			"consul_yaml_shopping_list": []interface{}{"bread", "milk", "eggs"},
			"consul_yaml_abc":           "xyz",
		},
		expectedConfig,
	)
}

// startEtcdKVMockServer starts a Consul key-value http mock server.
func startConsulKVMockServer(t *testing.T, key, content string, withPrefix bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make expectations upon endpoint
		expectedEndpoint := "/v1/kv/" + key
		if withPrefix {
			expectedEndpoint += "?recurse="
		}
		assertEqual(t, expectedEndpoint, r.URL.String())

		// send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintln(w, content); err != nil {
			t.Error(err)
		}
	}))
}

// getConsulExpectedConfigMapByFormatAndPrefix returns expected config maps
// (correlated with consulResponseContent variable).
func getConsulExpectedConfigMapByFormatAndPrefix(format string, withPrefix bool) map[string]interface{} {
	var expectedConfigMap map[string]interface{}
	const subkeyVal = "xyz"
	switch format {
	case xconf.RemoteValueJSON:
		expectedConfigMap = map[string]interface{}{
			"consul_json_foo":           "bar",
			"consul_json_year":          float64(2022),
			"consul_json_temperature":   37.5,
			"consul_json_shopping_list": []interface{}{"bread", "milk", "eggs"},
		}
		if withPrefix {
			expectedConfigMap["consul_json_abc"] = subkeyVal
		}
	case xconf.RemoteValueYAML:
		expectedConfigMap = map[string]interface{}{
			"consul_yaml_foo":           "bar",
			"consul_yaml_year":          2022,
			"consul_yaml_temperature":   37.5,
			"consul_yaml_shopping_list": []interface{}{"bread", "milk", "eggs"},
		}
		if withPrefix {
			expectedConfigMap["consul_yaml_abc"] = subkeyVal
		}
	case xconf.RemoteValuePlain:
		expectedConfigMap = map[string]interface{}{"consul_plain_key": "1000"}
		if withPrefix {
			expectedConfigMap["consul_plain_key/subkey"] = subkeyVal
		}
	}

	return expectedConfigMap
}

func benchmarkConsulLoader(format string, withCache bool) func(b *testing.B) {
	return func(b *testing.B) {
		b.Helper()
		content := consulResponseContent[format][true]
		key := consulKeys[format]
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, content)
		}))
		defer svr.Close()

		opts := []xconf.ConsulLoaderOption{
			xconf.ConsulLoaderWithHost(svr.URL),
			xconf.ConsulLoaderWithPrefix(),
			xconf.ConsulLoaderWithValueFormat(format),
		}
		if withCache {
			opts = append(opts, xconf.ConsulLoaderWithCache())
		}
		subject := xconf.NewConsulLoader(key, opts...)

		b.ReportAllocs()
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			_, err := subject.Load()
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkConsulLoader_json_noCache(b *testing.B) {
	benchmarkConsulLoader(xconf.RemoteValueJSON, false)(b)
}

func BenchmarkConsulLoader_json_withCache(b *testing.B) {
	benchmarkConsulLoader(xconf.RemoteValueJSON, true)(b)
}

func BenchmarkConsulLoader_yaml_noCache(b *testing.B) {
	benchmarkConsulLoader(xconf.RemoteValueYAML, false)(b)
}

func BenchmarkConsulLoader_yaml_withCache(b *testing.B) {
	benchmarkConsulLoader(xconf.RemoteValueYAML, true)(b)
}

func BenchmarkConsulLoader_plain_noCache(b *testing.B) {
	benchmarkConsulLoader(xconf.RemoteValuePlain, false)(b)
}

func BenchmarkConsulLoader_plain_withCache(b *testing.B) {
	benchmarkConsulLoader(xconf.RemoteValuePlain, true)(b)
}

func ExampleConsulLoader() {
	// load all keys starting with "APP_"
	host := "http://127.0.0.1:8500"
	loader := xconf.NewConsulLoader(
		"APP_",
		xconf.ConsulLoaderWithHost(host),
		xconf.ConsulLoaderWithPrefix(),
	)

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}
}
