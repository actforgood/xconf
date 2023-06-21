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

func TestMultiLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - merged config from multiple loaders", testMultiLoaderSuccess)
	t.Run("error - from loaders", testMultiLoaderReturnsLoadErr)
	t.Run("error - key conflict", testMultiLoaderReturnsKeyConflictErr)
	t.Run("success - safe-mutable config map", testMultiLoaderReturnsSafeMutableConfigMap)
}

func testMultiLoaderSuccess(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader1 = xconf.PlainLoader(map[string]any{
			"loader_1_foo": "foo - from Loader 1",
			"loader_1_bar": "bar - from Loader 1",
			"key":          "value - from Loader 1",
		})
		loader2 = xconf.PlainLoader(map[string]any{
			"loader_2_foo": "foo - from Loader 2",
			"loader_2_bar": "bar - from Loader 2",
			"loader_1_bar": "bar - from Loader 2 - overwrite Loader 1",
			"key":          "value - from Loader 2",
		})
		loader3 = xconf.PlainLoader(map[string]any{
			"loader_3_foo": "foo - from Loader 3",
			"loader_3_bar": "bar - from Loader 3",
			"loader_2_bar": "bar - from Loader 3 - overwrite Loader 2",
			"key":          "value - from Loader 3",
		})
		subject = xconf.NewMultiLoader(true, loader1, loader2, loader3)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]any{
			"loader_1_foo": "foo - from Loader 1",
			"loader_2_foo": "foo - from Loader 2",
			"loader_3_foo": "foo - from Loader 3",
			"loader_1_bar": "bar - from Loader 2 - overwrite Loader 1",
			"loader_2_bar": "bar - from Loader 3 - overwrite Loader 2",
			"loader_3_bar": "bar - from Loader 3",
			"key":          "value - from Loader 3",
		},
		config,
	)
}

func testMultiLoaderReturnsLoadErr(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedLoader1Err = errors.New("loader 1 intentionally triggered error")
		expectedLoader3Err = errors.New("loader 3 intentionally triggered error")
		loader1            = xconf.LoaderFunc(func() (map[string]any, error) {
			return nil, expectedLoader1Err
		})
		loader2 = xconf.PlainLoader(map[string]any{
			"foo": "bar",
		})
		loader3 = xconf.LoaderFunc(func() (map[string]any, error) {
			return nil, expectedLoader3Err
		})
		subject = xconf.NewMultiLoader(false, loader1, loader2, loader3)
	)

	// act
	config, err := subject.Load()

	// assert
	assertTrue(t, errors.Is(err, expectedLoader1Err))
	assertTrue(t, errors.Is(err, expectedLoader3Err))
	assertNil(t, config)
}

func testMultiLoaderReturnsKeyConflictErr(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader1 = xconf.PlainLoader(map[string]any{
			"foo": "bar",
			"x":   "y",
		})
		loader2 = xconf.PlainLoader(map[string]any{
			"foo": "same key as for Loader 1",
		})
		loader3 = xconf.PlainLoader(map[string]any{
			"abc": "xyz",
		})
		subject = xconf.NewMultiLoader(false, loader1, loader2, loader3)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	if assertNotNil(t, err) {
		var conflictErr xconf.KeyConflictError
		assertTrue(t, errors.As(err, &conflictErr))
		assertEqual(t, `key "foo" already exists`, conflictErr.Error())
	}
}

func testMultiLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader1 = xconf.PlainLoader(map[string]any{
			"multi_string": "some string",
			"multi_slice":  []any{"foo", "bar", "baz"},
		})
		loader2 = xconf.PlainLoader(map[string]any{
			"multi_map": map[string]any{
				"foo": "bar",
			},
		})
		subject        = xconf.NewMultiLoader(true, loader1, loader2)
		expectedConfig = map[string]any{
			"multi_string": "some string",
			"multi_slice":  []any{"foo", "bar", "baz"},
			"multi_map":    map[string]any{"foo": "bar"},
		}
	)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["multi_int"] = 3333
	config1["multi_string"] = "test multi string"
	config1["multi_slice"].([]any)[0] = "test multi slice"
	config1["multi_map"].(map[string]any)["foo"] = "test multi map"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]any{
			"multi_string": "some string",
			"multi_slice":  []any{"foo", "bar", "baz"},
			"multi_map":    map[string]any{"foo": "bar"},
		},
		expectedConfig,
	)
}

func benchmarkMultiLoader(allowKeyOverwrite bool) func(b *testing.B) {
	return func(b *testing.B) {
		b.Helper()
		loader1 := xconf.PlainLoader(map[string]any{
			"loader_1": "Loader 1",
		})
		loader2 := xconf.PlainLoader(map[string]any{
			"loader_2": "Loader 2",
		})
		loader3 := xconf.PlainLoader(map[string]any{
			"loader_3": "Loader 3",
		})
		subject := xconf.NewMultiLoader(allowKeyOverwrite, loader1, loader2, loader3)

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

func BenchmarkMultiLoader_withAllowingKeyOverwrite(b *testing.B) {
	benchmarkMultiLoader(true)(b)
}

func BenchmarkMultiLoader_withoutAllowingKeyOverwrite(b *testing.B) {
	benchmarkMultiLoader(false)(b)
}

func ExampleMultiLoader() {
	loader := xconf.NewMultiLoader(
		true, // allow key overwrite
		xconf.PlainLoader(map[string]any{
			"json_foo":  "bar from plain, will get overwritten",
			"yaml_foo":  "bar from plain, will get overwritten",
			"plain_key": "plain value",
		}),
		xconf.JSONFileLoader("testdata/config.json"),
		xconf.YAMLFileLoader("testdata/config.yaml"),
	)

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// json_foo: bar
	// json_year: 2022
	// json_temperature: 37.5
	// json_shopping_list: [bread milk eggs]
	// yaml_foo: bar
	// yaml_year: 2022
	// yaml_temperature: 37.5
	// yaml_shopping_list: [bread milk eggs]
	// plain_key: plain value
}
