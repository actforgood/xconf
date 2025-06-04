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

const invalidFileExt = ".invalid"

func TestFileLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - with .json", testFileLoaderWithJSON)
	t.Run("success - with .yaml", testFileLoaderWithYAML)
	t.Run("success - with .yml", testFileLoaderWithYML)
	t.Run("success - with .env", testFileLoaderWithDotEnv)
	t.Run("success - with .ini", testFileLoaderWithIni)
	t.Run("success - with .toml", testFileLoaderWithTOML)
	t.Run("success - with .properties", testFileLoaderWithProperties)
	t.Run("error - unknown extension", testFileLoaderWithUnknownExt)
}

func testFileLoaderWithJSON(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.FileLoader(jsonFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, jsonConfigMap, config)
}

func testFileLoaderWithYAML(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.FileLoader(yamlFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, yamlConfigMap, config)
}

func testFileLoaderWithYML(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.FileLoader("testdata/config.yml")

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, yamlConfigMap, config)
}

func testFileLoaderWithDotEnv(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.FileLoader(dotEnvFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, dotEnvConfigMap, config)
}

func testFileLoaderWithIni(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.FileLoader(iniFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, iniConfigMap, config)
}

func testFileLoaderWithTOML(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.FileLoader(tomlFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, tomlConfigMap, config)
}

func testFileLoaderWithProperties(t *testing.T) {
	t.Parallel()

	// arrange
	subject := xconf.FileLoader(propertiesFilePath)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, propertiesConfigMap, config)
}

func testFileLoaderWithUnknownExt(t *testing.T) {
	t.Parallel()

	// arrange
	expectedErr := xconf.ErrUnknownConfigFileExt
	tests := [...]struct {
		name          string
		inputFilePath string
	}{
		{
			name:          ".go",
			inputFilePath: "loader_file.go",
		},
		{
			name:          ".test",
			inputFilePath: "config.test",
		},
		{
			name:          ".txt",
			inputFilePath: "/some/path/config.txt",
		},
		{
			name:          ".foo",
			inputFilePath: "./a/b/c/bar.foo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject := xconf.FileLoader(test.inputFilePath)

			// act
			config, err := subject.Load()

			// assert
			assertNil(t, config)
			assertTrue(t, errors.Is(err, expectedErr))
		})
	}
}

func ExampleFileLoader() {
	exampleFiles := []string{
		"testdata/config.json",
		"testdata/config.yaml",
		"testdata/config.yml",
		"testdata/.env",
		"testdata/config.ini",
		"testdata/config.properties",
		"testdata/config.toml",
	}
	for _, filePath := range exampleFiles {
		loader := xconf.FileLoader(filePath)
		configMap, err := loader.Load()
		if err != nil {
			panic(err)
		}
		fmt.Println(len(configMap))
	}

	// Output:
	// 4
	// 4
	// 4
	// 4
	// 3
	// 4
	// 7
}
