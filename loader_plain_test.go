// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf_test

import (
	"testing"

	"github.com/actforgood/xconf"
)

func TestPlainLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - explicit map is returned as config", testPlainLoaderSuccess)
	t.Run("success - safe-mutable config map", testPlainLoaderReturnsSafeMutableConfigMap)
}

func testPlainLoaderSuccess(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedConfig = map[string]interface{}{
			"plain_foo":           "bar",
			"plain_year":          2022,
			"plain_temperature":   37.5,
			"plain_shopping_list": []string{"bread", "milk", "eggs"},
		}
		subject = xconf.PlainLoader(expectedConfig)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config)
}

func testPlainLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedConfig = map[string]interface{}{
			"plain_string": "some string",
			"plain_slice":  []string{"foo", "bar", "baz"},
			"plain_map":    map[string]interface{}{"foo": "bar"},
		}
		subject = xconf.PlainLoader(expectedConfig)
	)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["plain_int"] = 12345
	config1["plain_string"] = "test plain string"
	config1["plain_slice"].([]string)[0] = "test plain slice"
	config1["plain_map"].(map[string]interface{})["foo"] = "test plain map"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]interface{}{
			"plain_string": "some string",
			"plain_slice":  []string{"foo", "bar", "baz"},
			"plain_map":    map[string]interface{}{"foo": "bar"},
		},
		expectedConfig,
	)
}
