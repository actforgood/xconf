// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/actforgood/xconf"
)

func TestFilterKVLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - apply mixed filters", testFilterKVLoaderWithMixedFilters)
	t.Run("success - apply only whitelist filters", testFilterKVLoaderOnlyWithWhitelistFilters)
	t.Run("success - apply only blacklist filters", testFilterKVLoaderOnlyWithBlacklistFilters)
	t.Run("error - original, decorated loader", testFilterKVLoaderReturnsErrFromDecoratedLoader)
	t.Run("success - safe-mutable config map", testFilterKVLoaderReturnsSafeMutableConfigMap)
}

func testFilterKVLoaderWithMixedFilters(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"FOO_1": "whitelisted by filter 1",
			"FOO_2": "whitelisted by filter 1",
			"FOO_3": "whitelisted by filter 2",
			"FOO_4": "blacklisted by filter 3",
			"FOO_5": "ignored, not blacklisted and not whitelisted",
		})
		filter1 = xconf.FilterKVWhitelistFunc(func(key string, _ interface{}) bool {
			return key == "FOO_1" || key == "FOO_2"
		})
		filter2 = xconf.FilterKVWhitelistFunc(func(_ string, value interface{}) bool {
			return value.(string) == "whitelisted by filter 2"
		})
		filter3 = xconf.FilterKVBlacklistFunc(func(key string, _ interface{}) bool {
			return key == "FOO_4"
		})
		subject = xconf.FilterKVLoader(loader, filter1, filter2, filter3)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"FOO_1": "whitelisted by filter 1",
			"FOO_2": "whitelisted by filter 1",
			"FOO_3": "whitelisted by filter 2",
		},
		config,
	)
}

func testFilterKVLoaderOnlyWithWhitelistFilters(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"FOO1": "whitelisted by filter 1",
			"FOO2": "whitelisted by filter 1",
			"FOO3": "whitelisted by filter 2",
			"FOO4": "ignored, not blacklisted and not whitelisted",
			"FOO5": "ignored, not blacklisted and not whitelisted",
		})
		filter1 = xconf.FilterKVWhitelistFunc(func(key string, _ interface{}) bool {
			return key == "FOO1" || key == "FOO2"
		})
		filter2 = xconf.FilterKVWhitelistFunc(func(_ string, value interface{}) bool {
			return value.(string) == "whitelisted by filter 2"
		})
		subject = xconf.FilterKVLoader(loader, filter1, filter2)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"FOO1": "whitelisted by filter 1",
			"FOO2": "whitelisted by filter 1",
			"FOO3": "whitelisted by filter 2",
		},
		config,
	)
}

func testFilterKVLoaderOnlyWithBlacklistFilters(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"FOO1": "blacklisted by filter 1",
			"FOO2": "blacklisted by filter 1",
			"FOO3": "blacklisted by filter 2",
			"FOO4": "remains, not blacklisted",
			"FOO5": "remains, not blacklisted",
		})
		filter1 = xconf.FilterKVBlacklistFunc(func(key string, _ interface{}) bool {
			return key == "FOO1" || key == "FOO2"
		})
		filter2 = xconf.FilterKVBlacklistFunc(func(_ string, value interface{}) bool {
			return value.(string) == "blacklisted by filter 2"
		})
		subject = xconf.FilterKVLoader(loader, filter1, filter2)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"FOO4": "remains, not blacklisted",
			"FOO5": "remains, not blacklisted",
		},
		config,
	)
}

func testFilterKVLoaderReturnsErrFromDecoratedLoader(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = errors.New("intentionally triggered decorated loader err")
		loader      = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			return nil, expectedErr
		})
		filter = xconf.FilterKVBlacklistFunc(func(key string, _ interface{}) bool {
			return key == "whatever"
		})
		subject = xconf.FilterKVLoader(loader, filter)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, errors.Is(err, expectedErr))
}

func testFilterKVLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"filter_string":        "some string",
			"filter_int":           567,
			"filter_slice":         []interface{}{"foo", "bar", "baz"},
			"filter_string_map":    map[string]interface{}{"foo": "bar"},
			"filter_interface_map": map[interface{}]interface{}{1: "one"},
		})
		filter = xconf.FilterKVWhitelistFunc(func(key string, _ interface{}) bool {
			return key == "filter_string" || key == "filter_slice" ||
				key == "filter_string_map" || key == "filter_interface_map"
		})
		subject        = xconf.FilterKVLoader(loader, filter)
		expectedConfig = map[string]interface{}{
			"filter_string":        "some string",
			"filter_slice":         []interface{}{"foo", "bar", "baz"},
			"filter_string_map":    map[string]interface{}{"foo": "bar"},
			"filter_interface_map": map[interface{}]interface{}{1: "one"},
		}
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["filter_int"] = 1111
	config1["filter_string"] = "test filter string"
	config1["filter_slice"].([]interface{})[0] = "test filter slice"
	config1["filter_string_map"].(map[string]interface{})["foo"] = "test filter map"
	config1["filter_interface_map"].(map[interface{}]interface{})[1] = "test filter map"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]interface{}{
			"filter_string":        "some string",
			"filter_slice":         []interface{}{"foo", "bar", "baz"},
			"filter_string_map":    map[string]interface{}{"foo": "bar"},
			"filter_interface_map": map[interface{}]interface{}{1: "one"},
		},
		expectedConfig,
	)
}

func TestFilterKeyWithPrefix(t *testing.T) {
	t.Parallel()

	// arrange
	tests := [...]struct {
		name           string
		prefix         string
		inputKey       string
		inputValue     interface{}
		expectedResult bool
	}{
		{
			name:           "key has given prefix, return true",
			prefix:         "APP",
			inputKey:       "APP_FOO",
			inputValue:     "bar",
			expectedResult: true,
		},
		{
			name:           "key has given prefix, return true",
			prefix:         "APP_",
			inputKey:       "APP_FOO",
			inputValue:     123,
			expectedResult: true,
		},
		{
			name:           "key does not have given prefix, return false",
			prefix:         "APP_",
			inputKey:       "FOO",
			inputValue:     "bar",
			expectedResult: false,
		},
		{
			name:           "key does not have given prefix, case sensitivity, return false",
			prefix:         "app",
			inputKey:       "APP_FOO",
			inputValue:     "bar",
			expectedResult: false,
		},
		{
			name:           "key does not have given prefix, but value has, return false",
			prefix:         "APP",
			inputKey:       "FOO",
			inputValue:     "APPbar",
			expectedResult: false,
		},
	}
	subject := xconf.FilterKeyWithPrefix

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			// act
			result := subject(test.prefix)(test.inputKey, test.inputValue)

			// assert
			assertEqual(t, test.expectedResult, result)
		})
	}
}

func TestFilterKeyWithSuffix(t *testing.T) {
	t.Parallel()

	// arrange
	tests := [...]struct {
		name           string
		suffix         string
		inputKey       string
		inputValue     interface{}
		expectedResult bool
	}{
		{
			name:           "key has given suffix, return true",
			suffix:         "_SERVICE_HOST",
			inputKey:       "REDIS_SERVICE_HOST",
			inputValue:     "10.0.0.11",
			expectedResult: true,
		},
		{
			name:           "key has given suffix, return true",
			suffix:         "HOST",
			inputKey:       "REDIS_SERVICE_HOST",
			inputValue:     "10.0.0.11",
			expectedResult: true,
		},
		{
			name:           "key does not have given suffix, return false",
			suffix:         "_HOST",
			inputKey:       "FOO",
			inputValue:     "bar",
			expectedResult: false,
		},
		{
			name:           "key does not have given suffix, case sensitivity, return false",
			suffix:         "_host",
			inputKey:       "REDIS_SERVICE_HOST",
			inputValue:     "10.0.0.11",
			expectedResult: false,
		},
		{
			name:           "key does not have given suffix, but value has, return false",
			suffix:         "BAZ",
			inputKey:       "TEST",
			inputValue:     "barBAZ",
			expectedResult: false,
		},
	}
	subject := xconf.FilterKeyWithSuffix

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			// act
			result := subject(test.suffix)(test.inputKey, test.inputValue)

			// assert
			assertEqual(t, test.expectedResult, result)
		})
	}
}

func TestFilterEmptyValue(t *testing.T) {
	t.Parallel()

	// arrange
	tests := [...]struct {
		name           string
		inputKey       string
		inputValue     interface{}
		expectedResult bool
	}{
		{
			name:           "value is empty string, return true",
			inputKey:       "EMPTY_STR_KEY",
			inputValue:     "",
			expectedResult: true,
		},
		{
			name:           "value is nil, return true",
			inputKey:       "NULLABLE_KEY",
			inputValue:     nil,
			expectedResult: true,
		},
		{
			name:           "value is not empty, return false",
			inputKey:       "NOT_EMPTY_STR_KEY",
			inputValue:     "baz",
			expectedResult: false,
		},
		{
			name:           "value is not empty again, return false",
			inputKey:       "NOT_EMPTY_INT_KEY",
			inputValue:     0,
			expectedResult: false,
		},
		{
			name:           "key is empty, value is not, return false",
			inputKey:       "",
			inputValue:     "abc",
			expectedResult: false,
		},
	}
	subject := xconf.FilterEmptyValue

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			// act
			result := subject(test.inputKey, test.inputValue)

			// assert
			assertEqual(t, test.expectedResult, result)
		})
	}
}

func TestFilterExactKeys(t *testing.T) {
	t.Parallel()

	// arrange
	tests := [...]struct {
		name           string
		keys           []string
		inputKey       string
		inputValue     interface{}
		expectedResult bool
	}{
		{
			name:           "key is present in the list, return true",
			keys:           []string{"BAR", "FOO"},
			inputKey:       "FOO",
			inputValue:     "whatever",
			expectedResult: true,
		},
		{
			name:           "key is not present in the list, return false",
			keys:           []string{"BAR", "FOO"},
			inputKey:       "BAZ",
			inputValue:     "whatever",
			expectedResult: false,
		},
		{
			name:           "key is not present in the list, value is, return false",
			keys:           []string{"BAR", "FOO"},
			inputKey:       "BAZ",
			inputValue:     "FOO",
			expectedResult: false,
		},
		{
			name:           "key is not present in the list, case sensitive, return false",
			keys:           []string{"BAR", "FOO"},
			inputKey:       "foo",
			inputValue:     "whatever",
			expectedResult: false,
		},
	}
	subject := xconf.FilterExactKeys

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			// act
			result := subject(test.keys...)(test.inputKey, test.inputValue)

			// assert
			assertEqual(t, test.expectedResult, result)
		})
	}
}

func BenchmarkFilterKVLoader(b *testing.B) {
	loader := xconf.PlainLoader(map[string]interface{}{
		"FOO_1": "bar 1",
		"FOO_2": "bar 2",
		"FOO_3": "bar 3",
		"FOO_4": "bar 4",
		"FOO_5": "bar 5",
	})
	filter1 := xconf.FilterKVWhitelistFunc(func(key string, _ interface{}) bool {
		return key == "FOO_1" || key == "FOO_2"
	})
	filter2 := xconf.FilterKVWhitelistFunc(func(_ string, value interface{}) bool {
		return value.(string) == "bar 3"
	})
	filter3 := xconf.FilterKVBlacklistFunc(func(key string, _ interface{}) bool {
		return key == "FOO_4"
	})
	subject := xconf.FilterKVLoader(loader, filter1, filter2, filter3)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _ = subject.Load()
	}
}

func ExampleFilterEmptyValue() {
	origLoader := xconf.PlainLoader(map[string]interface{}{
		"redis_dial_timeout": "5s",
		"redis_dsn":          "",
	})
	loader := xconf.FilterKVLoader(
		origLoader,
		xconf.FilterKVBlacklistFunc(xconf.FilterEmptyValue),
	)

	configMap, _ := loader.Load()

	fmt.Println(configMap)

	// Output:
	// map[redis_dial_timeout:5s]
}

func ExampleFilterKeyWithSuffix() {
	origLoader := xconf.PlainLoader(map[string]interface{}{
		"REDIS_SERVICE_HOST": "10.0.0.11",
		"REDIS_SERVICE_PORT": "6379",
		"OS":                 "Windows",
	})
	loader := xconf.FilterKVLoader(
		origLoader,
		// K8s style accessible Services
		xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithSuffix("_SERVICE_HOST")),
		xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithSuffix("_SERVICE_PORT")),
	)

	configMap, _ := loader.Load()
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// REDIS_SERVICE_HOST: 10.0.0.11
	// REDIS_SERVICE_PORT: 6379
}

func ExampleFilterKeyWithPrefix() {
	origLoader := xconf.PlainLoader(map[string]interface{}{
		"APP_FOO_1": "bar 1",
		"APP_FOO_2": "bar 2",
		"OS":        "Windows",
	})
	loader := xconf.FilterKVLoader(
		origLoader,
		xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithPrefix("APP_")),
	)

	configMap, _ := loader.Load()
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// APP_FOO_1: bar 1
	// APP_FOO_2: bar 2
}

func ExampleFilterExactKeys() {
	origLoader := xconf.PlainLoader(map[string]interface{}{
		"FOO": "foo value",
		"BAR": "bar value",
		"BAZ": "baz value",
	})
	loader := xconf.FilterKVLoader(
		origLoader,
		xconf.FilterKVWhitelistFunc(xconf.FilterExactKeys("FOO", "BAR")),
	)

	configMap, _ := loader.Load()
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// FOO: foo value
	// BAR: bar value
}

func ExampleFilterKVLoader() {
	// in this example we assume our application's configs are
	// prefixed with APP_ and we want to allow them,
	// we also want to allow K8s services,
	// we also want to get rid of empty value configs.
	origLoader := xconf.PlainLoader(map[string]interface{}{
		"APP_FOO_1":          "bar 1",     // whitelisted
		"APP_FOO_2":          "bar 2",     // whitelisted
		"APP_FOO_3":          "",          // blacklisted
		"REDIS_SERVICE_HOST": "10.0.0.11", // whitelisted
		"REDIS_SERVICE_PORT": "6379",      // whitelisted
		"MYSQL_SERVICE_HOST": "10.0.0.12", // whitelisted
		"MYSQL_SERVICE_PORT": "3306",      // whitelisted
		"OS":                 "darwin",
		"HOME":               "/Users/JohnDoe",
		"USER":               "JohnDoe",
	})
	loader := xconf.FilterKVLoader(
		origLoader,
		xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithPrefix("APP_")),
		xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithSuffix("_SERVICE_HOST")),
		xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithSuffix("_SERVICE_PORT")),
		xconf.FilterKVBlacklistFunc(xconf.FilterEmptyValue),
	)

	configMap, _ := loader.Load()
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// APP_FOO_1: bar 1
	// APP_FOO_2: bar 2
	// REDIS_SERVICE_HOST: 10.0.0.11
	// REDIS_SERVICE_PORT: 6379
	// MYSQL_SERVICE_HOST: 10.0.0.12
	// MYSQL_SERVICE_PORT: 3306
}
