// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/actforgood/xconf"
)

func TestAlterValueLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - value is transformed", testAlterValueLoaderSuccess)
	t.Run("error - original, decorated loader", testAlterValueLoaderReturnsErrFromDecoratedLoader)
	t.Run("success - safe-mutable config map", testAlterValueLoaderReturnsSafeMutableConfigMap)
}

func testAlterValueLoaderSuccess(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"foo": "foo val",
			"bar": "bar val",
			"baz": 100,
			"x":   "y",
		})
		subject = xconf.AlterValueLoader(
			loader,
			func(value interface{}) interface{} { return value.(string) + " - modified" },
			"foo", "bar", "this-key-does-not-exist",
		)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"foo": "foo val - modified",
			"bar": "bar val - modified",
			"baz": 100,
			"x":   "y",
		},
		config,
	)
}

func testAlterValueLoaderReturnsErrFromDecoratedLoader(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = errors.New("intentionally triggered decorated loader error")
		loader      = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			return nil, expectedErr
		})
		subject = xconf.AlterValueLoader(
			loader,
			func(value interface{}) interface{} { return value },
			"foo", "bar",
		)
	)

	// act
	config, err := subject.Load()

	// assert
	assertTrue(t, errors.Is(err, expectedErr))
	assertNil(t, config)
}

func testAlterValueLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"string": "some string",
			"slice":  []string{"foo", "bar", "baz"},
			"map":    map[string]interface{}{"foo": "bar"},
		})
		subject = xconf.AlterValueLoader(
			loader,
			func(value interface{}) interface{} {
				value.(map[string]interface{})["foo"] = "f_o_o"

				return value
			},
			"map",
		)
		expectedConfig = map[string]interface{}{
			"string": "some string",
			"slice":  []string{"foo", "bar", "baz"},
			"map":    map[string]interface{}{"foo": "f_o_o"},
		}
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["int"] = 9999
	config1["slice"].([]string)[0] = "test alter value slice"
	config1["map"].(map[string]interface{})["foo"] = "test alter value map"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]interface{}{
			"string": "some string",
			"slice":  []string{"foo", "bar", "baz"},
			"map":    map[string]interface{}{"foo": "f_o_o"},
		},
		expectedConfig,
	)
}

func TestToStringList(t *testing.T) {
	t.Parallel()

	// arrange
	tests := [...]struct {
		name           string
		inputValue     interface{}
		expectedResult interface{}
	}{
		{
			name:           "value is single item list",
			inputValue:     "bread",
			expectedResult: []string{"bread"},
		},
		{
			name:           "value is three items list",
			inputValue:     "bread,eggs,milk",
			expectedResult: []string{"bread", "eggs", "milk"},
		},
		{
			name:           "value is not string, expect original value",
			inputValue:     10,
			expectedResult: 10,
		},
	}
	subject := xconf.ToStringList(",")

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			// act
			result := subject(test.inputValue)

			// assert
			assertEqual(t, test.expectedResult, result)
		})
	}
}

func TestToIntList(t *testing.T) {
	t.Parallel()

	// arrange
	tests := [...]struct {
		name           string
		inputValue     interface{}
		expectedResult interface{}
	}{
		{
			name:           "value is single item list",
			inputValue:     "10",
			expectedResult: []int{10},
		},
		{
			name:           "value is three items list",
			inputValue:     "10::100::1000",
			expectedResult: []int{10, 100, 1000},
		},
		{
			name:           "value is not string, expect original value",
			inputValue:     10.99,
			expectedResult: 10.99,
		},
	}
	subject := xconf.ToIntList("::")

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			// act
			result := subject(test.inputValue)

			// assert
			assertEqual(t, test.expectedResult, result)
		})
	}
}

func BenchmarkAlterValueLoader(b *testing.B) {
	origLoader := xconf.PlainLoader(map[string]interface{}{
		"foo":           "foo val",
		"bar":           100,
		"shopping_list": "bread,eggs,milk",
	})
	subject := xconf.AlterValueLoader(
		origLoader,
		xconf.ToStringList(","),
		"shopping_list",
	)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleAlterValueLoader() {
	origLoader := xconf.PlainLoader(map[string]interface{}{
		"foo":           "foo val",
		"bar":           100,
		"shopping_list": "bread,eggs,milk",
		"weekend_days":  "friday,saturday,sunday",
	})
	loader := xconf.AlterValueLoader(
		origLoader,
		xconf.ToStringList(","),
		"shopping_list", "weekend_days",
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
	// bar: 100
	// shopping_list: [bread eggs milk]
	// weekend_days: [friday saturday sunday]
}
