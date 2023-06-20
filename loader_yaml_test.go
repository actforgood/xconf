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

	"github.com/actforgood/xconf"
	"gopkg.in/yaml.v3"
)

var yamlConfigMap = map[string]any{
	"yaml_foo":           "bar",
	"yaml_year":          2022,
	"yaml_temperature":   37.5,
	"yaml_shopping_list": []any{"bread", "milk", "eggs"},
}

const yamlFilePath = "testdata/config.yaml"

func TestYAMLReaderLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid yaml content", testYAMLReaderLoaderWithValidContent)
	t.Run("error - invalid yaml content", testYAMLReaderLoaderWithInvalidContent)
	t.Run("success - safe-mutable config map", testYAMLReaderLoaderReturnsSafeMutableConfigMap)
}

func testYAMLReaderLoaderWithValidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `---
yaml_foo: bar
yaml_year: 2022
yaml_temperature: 37.5
yaml_shopping_list:
  - bread
  - milk
  - eggs`
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.YAMLReaderLoader(reader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, yamlConfigMap, config)
}

func testYAMLReaderLoaderWithInvalidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = "---\ninvalid\n  yaml content"
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.YAMLReaderLoader(reader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var yamlErr *yaml.TypeError
	assertTrue(t, errors.As(err, &yamlErr))
}

func testYAMLReaderLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `---
yaml_string: some string
yaml_slice:
  - foo
  - bar
  - baz
yaml_string_map:
  foo: bar
yaml_interface_map:
  1: one
`
		expectedConfig = map[string]any{
			"yaml_string":        "some string",
			"yaml_slice":         []any{"foo", "bar", "baz"},
			"yaml_string_map":    map[string]any{"foo": "bar"},
			"yaml_interface_map": map[any]any{1: "one"},
		}
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.YAMLReaderLoader(reader)
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["yaml_int"] = 1111
	config1["yaml_string"] = "test yaml string"
	config1["yaml_slice"].([]any)[0] = "test yaml slice"
	config1["yaml_string_map"].(map[string]any)["foo"] = "test yaml map"
	config1["yaml_interface_map"].(map[any]any)[1] = "test yaml map"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]any{
			"yaml_string":        "some string",
			"yaml_slice":         []any{"foo", "bar", "baz"},
			"yaml_string_map":    map[string]any{"foo": "bar"},
			"yaml_interface_map": map[any]any{1: "one"},
		},
		expectedConfig,
	)
}

func TestYAMLFileLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid file,valid content", testYAMLFileLoaderWithValidFile)
	t.Run("error - valid file,invalid content", testYAMLFileLoaderWithInvalidFileContent)
	t.Run("error - not found file", testYAMLFileLoaderWithNotFoundFile)
	t.Run("success - safe-mutable config map", testYAMLFileLoaderReturnsSafeMutableConfigMap)
}

func testYAMLFileLoaderWithValidFile(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.YAMLFileLoader(yamlFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, yamlConfigMap, config)
}

func testYAMLFileLoaderWithInvalidFileContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = yamlFilePath + ".invalid"
		subject  = xconf.YAMLFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var yamlErr *yaml.TypeError
	assertTrue(t, errors.As(err, &yamlErr))
}

func testYAMLFileLoaderWithNotFoundFile(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = "testdata/path/does/not/exist/config.yaml"
		subject  = xconf.YAMLFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, os.IsNotExist(err))
}

func testYAMLFileLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.YAMLFileLoader(yamlFilePath)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, yamlConfigMap, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["yaml_foo"] = "test yaml string modified"
	config1["yaml_year"] = 2099
	config1["yaml_shopping_list"].([]any)[0] = "test yaml slice modified"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, yamlConfigMap, config2)

	assertEqual(
		t,
		map[string]any{
			"yaml_foo":           "bar",
			"yaml_year":          2022,
			"yaml_temperature":   37.5,
			"yaml_shopping_list": []any{"bread", "milk", "eggs"},
		},
		yamlConfigMap,
	)
}

func BenchmarkYAMLFileLoader(b *testing.B) {
	subject := xconf.YAMLFileLoader(yamlFilePath)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleYAMLFileLoader() {
	loader := xconf.YAMLFileLoader("testdata/config.yaml")

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// yaml_foo: bar
	// yaml_year: 2022
	// yaml_temperature: 37.5
	// yaml_shopping_list: [bread milk eggs]
}
