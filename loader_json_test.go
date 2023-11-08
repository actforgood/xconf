// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/actforgood/xconf"
)

var jsonConfigMap = map[string]any{
	"json_foo":           "bar",
	"json_year":          float64(2022),
	"json_temperature":   37.5,
	"json_shopping_list": []any{"bread", "milk", "eggs"},
}

const jsonFilePath = "testdata/config.json"

func TestJSONReaderLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid json content", testJSONReaderLoaderWithValidContent)
	t.Run("error - invalid json content", testJSONReaderLoaderWithInvalidContent)
	t.Run("success - safe-mutable config map", testJSONReaderLoaderReturnsSafeMutableConfigMap)
}

func testJSONReaderLoaderWithValidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `{
"json_foo":"bar",
"json_year":2022,
"json_temperature":37.5,
"json_shopping_list":["bread","milk","eggs"]
}`
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.JSONReaderLoader(reader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, jsonConfigMap, config)
}

func testJSONReaderLoaderWithInvalidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = "{invalid json content\n"
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.JSONReaderLoader(reader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var jsonErr *json.SyntaxError
	assertTrue(t, errors.As(err, &jsonErr))
}

func testJSONReaderLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `{
"json_string":"some string",
"json_slice":["foo","bar","baz"],
"json_map":{"foo":"bar"}
}`
		reader         = bytes.NewReader([]byte(content))
		subject        = xconf.JSONReaderLoader(reader)
		expectedConfig = map[string]any{
			"json_string": "some string",
			"json_slice":  []any{"foo", "bar", "baz"},
			"json_map":    map[string]any{"foo": "bar"},
		}
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["json_int"] = 2222
	config1["json_string"] = "test json string"
	config1["json_slice"].([]any)[0] = "test json slice"
	config1["json_map"].(map[string]any)["foo"] = "test json map"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]any{
			"json_string": "some string",
			"json_slice":  []any{"foo", "bar", "baz"},
			"json_map":    map[string]any{"foo": "bar"},
		},
		expectedConfig,
	)
}

func TestJSONFileLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid file,valid content", testJSONFileLoaderWithValidFile)
	t.Run("error - valid file,invalid content", testJSONFileLoaderWithInvalidFileContent)
	t.Run("error - not found file", testJSONFileLoaderWithNotFoundFile)
	t.Run("success - safe-mutable config map", testJSONFileLoaderReturnsSafeMutableConfigMap)
}

func testJSONFileLoaderWithValidFile(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.JSONFileLoader(jsonFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, jsonConfigMap, config)
}

func testJSONFileLoaderWithInvalidFileContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = jsonFilePath + invalidFileExt
		subject  = xconf.JSONFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var jsonErr *json.SyntaxError
	assertTrue(t, errors.As(err, &jsonErr))
}

func testJSONFileLoaderWithNotFoundFile(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = "testdata/path/does/not/exist/config.json"
		subject  = xconf.JSONFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, os.IsNotExist(err))
}

func testJSONFileLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.JSONFileLoader(jsonFilePath)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, jsonConfigMap, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["json_foo"] = "test json string modified"
	config1["json_year"] = float64(2099)
	config1["json_shopping_list"].([]any)[0] = "test json slice modified"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, jsonConfigMap, config2)

	assertEqual(
		t,
		map[string]any{
			"json_foo":           "bar",
			"json_year":          float64(2022),
			"json_temperature":   37.5,
			"json_shopping_list": []any{"bread", "milk", "eggs"},
		},
		jsonConfigMap,
	)
}

func BenchmarkJSONFileLoader(b *testing.B) {
	subject := xconf.JSONFileLoader(jsonFilePath)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleJSONFileLoader() {
	loader := xconf.JSONFileLoader("testdata/config.json")

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
}
