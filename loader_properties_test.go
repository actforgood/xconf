// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/actforgood/xconf"
)

var propertiesConfigMap = map[string]any{
	"properties_foo":         "bar",
	"properties_baz":         "bar",
	"properties_year":        "2022",
	"properties_temperature": "37.5",
}

const propertiesFilePath = "testdata/config.properties"

func TestPropertiesFileLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid file,valid content", testPropertiesFileLoaderWithValidFile)
	t.Run("error - valid file,invalid content", testPropertiesFileLoaderWithInvalidFileContent)
	t.Run("error - not found file", testPropertiesFileLoaderWithNotFoundFile)
	t.Run("success - safe-mutable config map", testPropertiesFileLoaderReturnsSafeMutableConfigMap)
}

func testPropertiesFileLoaderWithValidFile(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.PropertiesFileLoader(propertiesFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, propertiesConfigMap, config)
}

func testPropertiesFileLoaderWithInvalidFileContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = propertiesFilePath + invalidFileExt
		subject  = xconf.PropertiesFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertNotNil(t, err)
}

func testPropertiesFileLoaderWithNotFoundFile(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = "testdata/path/does/not/exist/config.properties"
		subject  = xconf.PropertiesFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, os.IsNotExist(err))
}

func testPropertiesFileLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.PropertiesFileLoader(propertiesFilePath)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, propertiesConfigMap, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["properties_foo"] = "test properties string modified"
	config1["properties_another_key"] = "some properties value"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, propertiesConfigMap, config2)

	assertEqual(
		t,
		map[string]any{
			"properties_foo":         "bar",
			"properties_baz":         "bar",
			"properties_year":        "2022",
			"properties_temperature": "37.5",
		},
		propertiesConfigMap,
	)
}

func TestPropertiesBytesLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid content", testPropertiesBytesLoaderWithValidContent)
	t.Run("error - invalid content", testPropertiesBytesLoaderWithInvalidContent)
	t.Run("success - safe-mutable config map", testPropertiesBytesLoaderReturnsSafeMutableConfigMap)
}

func testPropertiesBytesLoaderWithValidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `properties_foo = bar
properties_baz = ${properties_foo}
properties_year=2022
properties_temperature=37.5`
		subject = xconf.PropertiesBytesLoader([]byte(content))
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, propertiesConfigMap, config)
}

func testPropertiesBytesLoaderWithInvalidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `foo=bar
baz=${foo invalid properties content`
		subject = xconf.PropertiesBytesLoader([]byte(content))
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertNotNil(t, err)
}

func testPropertiesBytesLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `properties_foo = bar
properties_baz = ${properties_foo}
properties_year=2022
properties_temperature=37.5`
		subject = xconf.PropertiesBytesLoader([]byte(content))
	)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, propertiesConfigMap, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["properties_foo"] = "test properties string modified"
	config1["properties_another_key"] = "some properties value"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, propertiesConfigMap, config2)

	assertEqual(
		t,
		map[string]any{
			"properties_foo":         "bar",
			"properties_baz":         "bar",
			"properties_year":        "2022",
			"properties_temperature": "37.5",
		},
		propertiesConfigMap,
	)
}

func BenchmarkPropertiesFileLoader(b *testing.B) {
	subject := xconf.PropertiesFileLoader(propertiesFilePath)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExamplePropertiesFileLoader() {
	loader := xconf.PropertiesFileLoader("testdata/config.properties")

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// properties_foo: bar
	// properties_baz: bar
	// properties_year: 2022
	// properties_temperature: 37.5
}
