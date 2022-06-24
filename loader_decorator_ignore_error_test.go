// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/actforgood/xconf"
)

func TestIgnoreErrorLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - decorated loader err is ignored", testIgnoreErrorLoaderErrorIsIgnored)
	t.Run("success - decorated loader err is not ignored", testIgnoreErrorLoaderErrorIsNotIgnored)
	t.Run("success - decorated loader returns no err", testIgnoreErrorLoaderWithNoError)
	t.Run("success - safe-mutable config map", testIgnoreErrorLoaderReturnsSafeMutableConfigMap)
}

func testIgnoreErrorLoaderErrorIsIgnored(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			return nil, os.ErrNotExist
		})
		subject = xconf.IgnoreErrorLoader(loader, os.ErrInvalid, os.ErrNotExist)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, map[string]interface{}{}, config)
}

func testIgnoreErrorLoaderErrorIsNotIgnored(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = errors.New("intentionally triggered some other type of error")
		loader      = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			return nil, expectedErr
		})
		subject = xconf.IgnoreErrorLoader(loader, os.ErrInvalid, os.ErrNotExist)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, errors.Is(err, expectedErr))
}

func testIgnoreErrorLoaderWithNoError(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedConfig = map[string]interface{}{
			"foo": "bar",
		}
		loader  = xconf.PlainLoader(expectedConfig)
		subject = xconf.IgnoreErrorLoader(loader, os.ErrInvalid, os.ErrNotExist)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config)
}

func testIgnoreErrorLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			return nil, os.ErrNotExist
		})
		subject        = xconf.IgnoreErrorLoader(loader, os.ErrInvalid, os.ErrNotExist)
		expectedConfig = map[string]interface{}{}
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["abc"] = "ABC"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]interface{}{},
		expectedConfig,
	)
}

func BenchmarkIgnoreErrorLoader(b *testing.B) {
	loader := xconf.LoaderFunc(func() (map[string]interface{}, error) {
		return nil, os.ErrNotExist
	})
	subject := xconf.IgnoreErrorLoader(loader, os.ErrInvalid, os.ErrNotExist)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _ = subject.Load()
	}
}

func ExampleIgnoreErrorLoader() {
	// in this example we assume we want to load configs from
	// a main source (OS Env for example - here we provide a PlainLoader)
	// and eventually from a JSON configuration file.
	loader := xconf.NewMultiLoader(
		true, // allow keys overwrite
		xconf.PlainLoader(map[string]interface{}{
			"APP_FOO_1": "bar 1",
			"APP_FOO_2": "bar 2",
		}),
		xconf.IgnoreErrorLoader(
			xconf.JSONFileLoader("/this/path/might/not/exist/config.json"),
			os.ErrNotExist,
		),
	)

	configMap, _ := loader.Load()
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// APP_FOO_1: bar 1
	// APP_FOO_2: bar 2
}
