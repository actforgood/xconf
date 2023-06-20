// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/actforgood/xconf"
	"github.com/pelletier/go-toml/v2"
)

var tomlConfigMap = map[string]any{
	"toml_foo":           "bar",
	"toml_year":          int64(2022),
	"toml_temperature":   37.5,
	"toml_shopping_list": []any{"bread", "milk", "eggs"},
	"toml_enabled":       true,
	"toml_dob":           time.Date(1990, 5, 28, 7, 32, 0, 0, time.FixedZone("", 2*3600)),
	"toml_servers": map[string]any{
		"alpha": map[string]any{
			"ip":   "10.0.0.1",
			"role": "frontend",
		},
		"beta": map[string]any{
			"ip":   "10.0.0.2",
			"role": "backend",
		},
	},
}

const tomlFilePath = "testdata/config.toml"

func TestTOMLReaderLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid toml content", testTOMLReaderLoaderWithValidContent)
	t.Run("error - invalid toml content", testTOMLReaderLoaderWithInvalidContent)
	t.Run("success - safe-mutable config map", testTOMLReaderLoaderReturnsSafeMutableConfigMap)
}

func testTOMLReaderLoaderWithValidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `toml_foo = "bar"
toml_year = 2022
toml_temperature = 37.5
toml_shopping_list = [
	"bread",
	"milk",
	"eggs"
]
toml_enabled = true
toml_dob = 1990-05-28T07:32:00+02:00
		
[toml_servers]

	[toml_servers.alpha]
	ip = "10.0.0.1"
	role = "frontend"

	[toml_servers.beta]
	ip = "10.0.0.2"
	role = "backend"`
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.TOMLReaderLoader(reader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, tomlConfigMap, config)
}

func testTOMLReaderLoaderWithInvalidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `foo
invalid toml content`
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.TOMLReaderLoader(reader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var tomlErr *toml.DecodeError
	assertTrue(t, errors.As(err, &tomlErr))
}

func testTOMLReaderLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `toml_string = "some string"
toml_slice = ["foo", "bar", "baz"]
[toml_string_map]
  foo = "bar"
`
		expectedConfig = map[string]any{
			"toml_string":     "some string",
			"toml_slice":      []any{"foo", "bar", "baz"},
			"toml_string_map": map[string]any{"foo": "bar"},
		}
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.TOMLReaderLoader(reader)
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["toml_int"] = 88
	config1["toml_string"] = "test toml string"
	config1["toml_slice"].([]any)[0] = "test toml slice"
	config1["toml_string_map"].(map[string]any)["foo"] = "test toml map"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]any{
			"toml_string":     "some string",
			"toml_slice":      []any{"foo", "bar", "baz"},
			"toml_string_map": map[string]any{"foo": "bar"},
		},
		expectedConfig,
	)
}

func TestTOMLFileLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid file,valid content", testTOMLFileLoaderWithValidFile)
	t.Run("error - valid file,invalid content", testTOMLFileLoaderWithInvalidFileContent)
	t.Run("error - not found file", testTOMLFileLoaderWithNotFoundFile)
	t.Run("success - safe-mutable config map", testTOMLFileLoaderReturnsSafeMutableConfigMap)
}

func testTOMLFileLoaderWithValidFile(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.TOMLFileLoader(tomlFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, tomlConfigMap, config)
}

func testTOMLFileLoaderWithInvalidFileContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = tomlFilePath + ".invalid"
		subject  = xconf.TOMLFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var tomlErr *toml.DecodeError
	assertTrue(t, errors.As(err, &tomlErr))
}

func testTOMLFileLoaderWithNotFoundFile(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = "testdata/path/does/not/exist/config.toml"
		subject  = xconf.TOMLFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, os.IsNotExist(err))
}

func testTOMLFileLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.TOMLFileLoader(tomlFilePath)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, tomlConfigMap, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["toml_foo"] = "test toml string modified"
	config1["toml_year"] = 2099
	config1["toml_shopping_list"].([]any)[0] = "test toml slice modified"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, tomlConfigMap, config2)

	assertEqual(
		t,
		map[string]any{
			"toml_foo":           "bar",
			"toml_year":          int64(2022),
			"toml_temperature":   37.5,
			"toml_shopping_list": []any{"bread", "milk", "eggs"},
			"toml_enabled":       true,
			"toml_dob":           time.Date(1990, 5, 28, 7, 32, 0, 0, time.FixedZone("", 2*3600)),
			"toml_servers": map[string]any{
				"alpha": map[string]any{
					"ip":   "10.0.0.1",
					"role": "frontend",
				},
				"beta": map[string]any{
					"ip":   "10.0.0.2",
					"role": "backend",
				},
			},
		},
		tomlConfigMap,
	)
}

func BenchmarkTOMLFileLoader(b *testing.B) {
	subject := xconf.TOMLFileLoader(tomlFilePath)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleTOMLFileLoader() {
	loader := xconf.TOMLFileLoader("testdata/config.toml")

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	fmt.Println("toml_foo:", configMap["toml_foo"])
	fmt.Println("toml_year:", configMap["toml_year"])
	fmt.Println("toml_temperature:", configMap["toml_temperature"])
	fmt.Println("toml_shopping_list:", configMap["toml_shopping_list"])

	// Unordered output:
	// toml_foo: bar
	// toml_year: 2022
	// toml_temperature: 37.5
	// toml_shopping_list: [bread milk eggs]
}
