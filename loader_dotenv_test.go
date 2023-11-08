// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/actforgood/xconf"
)

var dotEnvConfigMap = map[string]any{
	"DOTENV_FOO":           "bar",
	"DOTENV_YEAR":          "2022",
	"DOTENV_TEMPERATURE":   "37.5",
	"DOTENV_SHOPPING_LIST": "bread,milk,eggs",
}

const dotEnvFilePath = "testdata/.env"

func TestDotEnvReaderLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid .env content", testDotEnvReaderLoaderWithValidContent)
	t.Run("error - invalid .env content", testDotEnvReaderLoaderWithInvalidContent)
	t.Run("success - safe-mutable config map", testDotEnvReaderLoaderReturnsSafeMutableConfigMap)
}

func testDotEnvReaderLoaderWithValidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `DOTENV_FOO=bar
DOTENV_YEAR=2022
DOTENV_TEMPERATURE=37.5
DOTENV_SHOPPING_LIST=bread,milk,eggs`
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.DotEnvReaderLoader(reader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, dotEnvConfigMap, config)
}

func testDotEnvReaderLoaderWithInvalidContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `foo
invalid dot env content`
		reader  = bytes.NewReader([]byte(content))
		subject = xconf.DotEnvReaderLoader(reader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertNotNil(t, err)
}

func testDotEnvReaderLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		content = `DOTENV_FOO=bar
DOTENV_YEAR=2022`
		reader         = bytes.NewReader([]byte(content))
		subject        = xconf.DotEnvReaderLoader(reader)
		expectedConfig = map[string]any{
			"DOTENV_FOO":  "bar",
			"DOTENV_YEAR": "2022",
		}
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["DOTENV_FOO"] = "bar bar bar"
	config1["DOTENV_YEAR"] = 2050
	config1["DOTENV_TEMPERATURE"] = 38.5

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]any{
			"DOTENV_FOO":  "bar",
			"DOTENV_YEAR": "2022",
		},
		expectedConfig,
	)
}

func TestDotEnvFileLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - valid file,valid content", testDotEnvFileLoaderWithValidFile)
	t.Run("error - valid file,invalid content", testDotEnvFileLoaderWithInvalidFileContent)
	t.Run("error - not found file", testDotEnvFileLoaderWithNotFoundFile)
	t.Run("success - safe-mutable config map", testDotEnvFileLoaderReturnsSafeMutableConfigMap)
}

func testDotEnvFileLoaderWithValidFile(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.DotEnvFileLoader(dotEnvFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, dotEnvConfigMap, config)
}

func testDotEnvFileLoaderWithInvalidFileContent(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = dotEnvFilePath + invalidFileExt
		subject  = xconf.DotEnvFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertNotNil(t, err)
}

func testDotEnvFileLoaderWithNotFoundFile(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		filePath = "testdata/path/does/not/exist/.env"
		subject  = xconf.DotEnvFileLoader(filePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, os.IsNotExist(err))
}

func testDotEnvFileLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.DotEnvFileLoader(dotEnvFilePath)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, dotEnvConfigMap, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["DOTENV_FOO"] = "bar bar bar"
	config1["DOTENV_YEAR"] = 2050
	config1["DOTENV_TEMPERATURE"] = 38.5

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, dotEnvConfigMap, config2)

	assertEqual(
		t,
		map[string]any{
			"DOTENV_FOO":           "bar",
			"DOTENV_YEAR":          "2022",
			"DOTENV_TEMPERATURE":   "37.5",
			"DOTENV_SHOPPING_LIST": "bread,milk,eggs",
		},
		dotEnvConfigMap,
	)
}

func BenchmarkDotEnvFileLoader(b *testing.B) {
	subject := xconf.DotEnvFileLoader(dotEnvFilePath)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleDotEnvFileLoader() {
	loader := xconf.DotEnvFileLoader("testdata/.env")

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// DOTENV_FOO: bar
	// DOTENV_YEAR: 2022
	// DOTENV_TEMPERATURE: 37.5
	// DOTENV_SHOPPING_LIST: bread,milk,eggs
}
