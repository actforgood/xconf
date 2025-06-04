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
	"gopkg.in/ini.v1"
)

var iniConfigMap = map[string]any{
	"ini_foo": "bar",
	"time": map[string]any{
		"ini_year": "2022",
	},
	"temperature": map[string]any{
		"ini_celsius":    "37.5",
		"ini_fahrenheit": "99.5",
	},
}

const iniFilePath = "testdata/config.ini"

func TestIniFileLoader_withValidFile(t *testing.T) {
	t.Parallel()

	t.Run("success - valid file,valid content", testIniFileLoaderWithValidFile)
	t.Run("error - valid file,invalid content", testIniFileLoaderWithInvalidFileContent)
	t.Run("error - not found file", testIniFileLoaderWithNotFoundFile)
	t.Run("success - custom ini load options applied", testIniFileLoaderWithCustomIniLoadOptions)
	t.Run("success - safe-mutable config map", testIniFileLoaderReturnsSafeMutableConfigMap)
}

func testIniFileLoaderWithValidFile(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.NewIniFileLoader(iniFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, iniConfigMap, config)
}

func testIniFileLoaderWithInvalidFileContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = iniFilePath + invalidFileExt
		subject  = xconf.NewIniFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, ini.IsErrDelimiterNotFound(err))
}

func testIniFileLoaderWithNotFoundFile(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = "testdata/path/does/not/exist/config.ini"
		subject  = xconf.NewIniFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, os.IsNotExist(err))
}

func testIniFileLoaderWithCustomIniLoadOptions(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = "testdata/path/does/not/exist/config.ini"
		subject  = xconf.NewIniFileLoader(
			filePath,
			xconf.IniFileLoaderWithLoadOptions(ini.LoadOptions{Loose: true}),
		)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, 0, len(config))
}

func testIniFileLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.NewIniFileLoader(iniFilePath)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, iniConfigMap, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["ini_foo"] = "test ini string modified"
	config1["ini_another_key"] = "some ini value"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, iniConfigMap, config2)

	assertEqual(
		t,
		map[string]any{
			"ini_foo": "bar",
			"time": map[string]any{
				"ini_year": "2022",
			},
			"temperature": map[string]any{
				"ini_celsius":    "37.5",
				"ini_fahrenheit": "99.5",
			},
		},
		iniConfigMap,
	)
}

func BenchmarkIniFileLoader(b *testing.B) {
	subject := xconf.NewIniFileLoader(iniFilePath)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleIniFileLoader() {
	loader := xconf.NewIniFileLoader("testdata/config.ini")

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	fmt.Println(configMap["ini_foo"])
	fmt.Println(configMap["temperature"].(map[string]any)["ini_celsius"])
	fmt.Println(configMap["temperature"].(map[string]any)["ini_fahrenheit"])

	// Unordered output:
	// bar
	// 37.5
	// 99.5
}
