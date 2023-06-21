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

func TestAliasLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - aliases are set", testAliasLoaderSuccess)
	t.Run("error - invalid list (odd elements number)", testAliasLoaderReturnsErrAliasPairBroken)
	t.Run("error - original, decorated loader", testAliasLoaderReturnsErrFromDecoratedLoader)
	t.Run("success - safe-mutable config map", testAliasLoaderReturnsSafeMutableConfigMap)
}

func testAliasLoaderSuccess(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]any{
			"foo": 12345,
			"bar": "bar val",
		})
		subject = xconf.AliasLoader(
			loader,
			"alias_1_foo", "foo",
			"alias_2_foo", "foo",
			"alias_bar", "bar",
			"alias_unknown", "unknown", // this key does not exist
		)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]any{
			"foo":         12345,
			"bar":         "bar val",
			"alias_1_foo": 12345,
			"alias_2_foo": 12345,
			"alias_bar":   "bar val",
		},
		config,
	)
}

func testAliasLoaderReturnsErrAliasPairBroken(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = xconf.ErrAliasPairBroken
		loader      = xconf.PlainLoader(map[string]any{
			"foo": 12345,
			"bar": "bar val",
		})
		subject = xconf.AliasLoader(
			loader,
			"alias_foo", "foo",
			"alias_bar",
		)
	)

	// act
	config, err := subject.Load()

	// assert
	assertTrue(t, errors.Is(err, expectedErr))
	assertNil(t, config)
}

func testAliasLoaderReturnsErrFromDecoratedLoader(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = errors.New("intentionally triggered decorated loader error")
		loader      = xconf.LoaderFunc(func() (map[string]any, error) {
			return nil, expectedErr
		})
		subject = xconf.AliasLoader(
			loader,
			"some-alias-for", "some-key",
		)
	)

	// act
	config, err := subject.Load()

	// assert
	assertTrue(t, errors.Is(err, expectedErr))
	assertNil(t, config)
}

func testAliasLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]any{
			"string": "some string",
			"slice":  []string{"foo", "bar", "baz"},
			"map":    map[string]any{"foo": "bar"},
		})
		subject = xconf.AliasLoader(
			loader,
			"alias_slice", "slice",
			"alias_map", "map",
		)
		expectedConfig = map[string]any{
			"string":      "some string",
			"slice":       []string{"foo", "bar", "baz"},
			"map":         map[string]any{"foo": "bar"},
			"alias_slice": []string{"foo", "bar", "baz"},
			"alias_map":   map[string]any{"foo": "bar"},
		}
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["int"] = 5555
	config1["slice"].([]string)[0] = "test slice"
	config1["map"].(map[string]any)["foo"] = "test map"
	config1["alias_slice"].([]string)[1] = "test alias slice"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]any{
			"string":      "some string",
			"slice":       []string{"foo", "bar", "baz"},
			"map":         map[string]any{"foo": "bar"},
			"alias_slice": []string{"foo", "bar", "baz"},
			"alias_map":   map[string]any{"foo": "bar"},
		},
		expectedConfig,
	)
}

func BenchmarkAliasLoader(b *testing.B) {
	origLoader := xconf.PlainLoader(map[string]any{
		"foo": "foo val",
		"baz": "baz val",
	})
	subject := xconf.AliasLoader(origLoader, "FOO_REBRANDED", "foo")

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleAliasLoader() {
	origLoader := xconf.PlainLoader(map[string]any{
		"foo": "foo val",
		"bar": "bar val",
		"baz": "baz val",
	})
	loader := xconf.AliasLoader(
		origLoader,
		"NEW_FOO", "foo",
		"NEW_BAZ", "baz",
	)

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// foo: foo val
	// bar: bar val
	// baz: baz val
	// NEW_FOO: foo val
	// NEW_BAZ: baz val
}
