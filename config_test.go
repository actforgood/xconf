// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/actforgood/xconf"
)

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("valid object", testNewDefaultConfigReturnsValidObject)
	t.Run("error", testNewDefaultConfigReturnsError)
	t.Run("finalizer is called", testNewDefaultConfigFinalizerIsCalled)
}

func testNewDefaultConfigReturnsValidObject(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		_       xconf.Config = (*xconf.DefaultConfig)(nil) // check also implemented interfaces
		_       io.Closer    = (*xconf.DefaultConfig)(nil)
		loader               = xconf.PlainLoader(map[string]interface{}{"foo": "bar"})
		subject              = xconf.NewDefaultConfig
	)

	// act
	result, err := subject(loader)

	// assert
	assertNil(t, err)
	if assertNotNil(t, result) {
		_ = result.Close()
	}
}

func testNewDefaultConfigReturnsError(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = errors.New("intentionally triggered test error")
		loader      = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			return nil, expectedErr
		})
		subject = xconf.NewDefaultConfig
	)

	// act
	result, err := subject(loader)

	// assert
	assertTrue(t, errors.Is(err, expectedErr))
	assertNil(t, result)
}

func testNewDefaultConfigFinalizerIsCalled(t *testing.T) {
	t.Parallel()

	// test finalizer is called if we "forget" to call Close.
	// arrange
	var (
		callsCnt uint32
		loader   = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			atomic.AddUint32(&callsCnt, 1)
			if atomic.LoadUint32(&callsCnt) == 1 {
				return map[string]interface{}{"foo": "bar"}, nil
			}

			return map[string]interface{}{"foo": "baz"}, nil
		})
		_, err = xconf.NewDefaultConfig(
			loader,
			xconf.DefaultConfigWithReloadInterval(700*time.Millisecond),
		)
	)
	requireNil(t, err)

	// act
	runtime.GC()
	time.Sleep(900 * time.Millisecond)

	assertEqual(t, uint32(1), atomic.LoadUint32(&callsCnt))
}

func TestDefaultConfig_Get(t *testing.T) {
	t.Parallel()

	t.Run("get key with no default", testDefaultConfigGetKeyNoDefault)
	t.Run("get key with default", testDefaultConfigGetKeyWithDefault)
	t.Run("get key case insensitive", testDefaultConfigGetKeyCaseInsensitive)
	t.Run("get reloaded key", testDefaultConfigGetKeyReloaded)
	t.Run("reload error is handled", testDefaultConfigWithReloadErrorHandler)
	t.Run("cast - get string key", testDefaultConfigGetStringKey)
	t.Run("cast - get int key", testDefaultConfigGetIntKey)
	t.Run("cast - get int64 key", testDefaultConfigGetInt64Key)
	t.Run("cast - get int32 key", testDefaultConfigGetInt32Key)
	t.Run("cast - get int16 key", testDefaultConfigGetInt16Key)
	t.Run("cast - get int8 key", testDefaultConfigGetInt8Key)
	t.Run("cast - get uint key", testDefaultConfigGetUintKey)
	t.Run("cast - get uint64 key", testDefaultConfigGetUint64Key)
	t.Run("cast - get uint32 key", testDefaultConfigGetUint32Key)
	t.Run("cast - get uint16 key", testDefaultConfigGetUint16Key)
	t.Run("cast - get uint8 key", testDefaultConfigGetUint8Key)
	t.Run("cast - get float64 key", testDefaultConfigGetFloat64Key)
	t.Run("cast - get float32 key", testDefaultConfigGetFloat32Key)
	t.Run("cast - get bool key", testDefaultConfigGetBoolKey)
	t.Run("cast - get duration key", testDefaultConfigGetDurationKey)
	t.Run("cast - get time key", testDefaultConfigGetTimeKey)
	t.Run("cast - get string slice key", testDefaultConfigGetStringSliceKey)
	t.Run("cast - get int slice key", testDefaultConfigGetIntSliceKey)
	t.Run("cast - not a covered type", testDefaultConfigGetKeyWithNotCoveredDefaultValueType)
}

func testDefaultConfigGetKeyNoDefault(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"foo":  "bar",
			"Foo":  "Bar",
			"year": 2022,
		})
		subject, err = xconf.NewDefaultConfig(loader)
	)
	requireNil(t, err)
	defer subject.Close()

	// act
	result1 := subject.Get("foo")
	result2 := subject.Get("Foo")
	result3 := subject.Get("year")
	result4 := subject.Get("this-key-does-not-exist")

	// assert
	assertEqual(t, "bar", result1)
	assertEqual(t, "Bar", result2)
	assertEqual(t, 2022, result3)
	assertNil(t, result4)
}

func testDefaultConfigGetKeyWithDefault(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"foo":  "bar",
			"Foo":  "Bar",
			"year": 2022,
		})
		subject, err = xconf.NewDefaultConfig(loader)
	)
	requireNil(t, err)
	defer subject.Close()

	// act
	result1 := subject.Get("foo", "foo-default")
	result2 := subject.Get("Foo", "Foo-default")
	result3 := subject.Get("year", 2099)
	result4 := subject.Get("this-key-does-not-exist", "some-default")

	// assert
	assertEqual(t, "bar", result1)
	assertEqual(t, "Bar", result2)
	assertEqual(t, 2022, result3)
	assertEqual(t, "some-default", result4)
}

func testDefaultConfigGetKeyCaseInsensitive(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"foo":  "bar",
			"year": 2022,
		})
		subject, err = xconf.NewDefaultConfig(
			loader,
			xconf.DefaultConfigWithIgnoreCaseSensitivity(),
		)
		tests = []string{"foo", "Foo", "FOO", "foO", "fOO"}
	)
	requireNil(t, err)
	defer subject.Close()

	for _, key := range tests {
		// act
		result := subject.Get(key)

		// assert
		assertEqual(t, "bar", result)
	}
}

func testDefaultConfigGetKeyReloaded(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		callsCnt uint32
		loader   = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			atomic.AddUint32(&callsCnt, 1)
			if atomic.LoadUint32(&callsCnt) == 1 {
				return map[string]interface{}{"foo": "bar"}, nil
			}

			return map[string]interface{}{"foo": "baz"}, nil
		})
		subject, err = xconf.NewDefaultConfig(
			loader,
			xconf.DefaultConfigWithReloadInterval(300*time.Millisecond),
		)
	)
	requireNil(t, err)
	defer subject.Close()

	// act
	result := subject.Get("foo")

	// assert
	assertEqual(t, "bar", result)
	assertEqual(t, uint32(1), atomic.LoadUint32(&callsCnt))

	// act
	time.Sleep(500 * time.Millisecond)
	result = subject.Get("foo")

	// assert
	assertEqual(t, "baz", result)
	assertTrue(t, atomic.LoadUint32(&callsCnt) > 1)
}

func testDefaultConfigWithReloadErrorHandler(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loaderCallsCnt uint32
		expectedErr    = errors.New("intentionally triggered Load error")
		loader         = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			atomic.AddUint32(&loaderCallsCnt, 1)
			if atomic.LoadUint32(&loaderCallsCnt) == 2 {
				return nil, expectedErr
			}

			return map[string]interface{}{"foo": "bar"}, nil
		})
		errHandlerCallsCnt uint32
		errHandler         = func(err error) {
			atomic.AddUint32(&errHandlerCallsCnt, 1)
			assertEqual(t, expectedErr, err)
		}
		subject, err = xconf.NewDefaultConfig(
			loader,
			xconf.DefaultConfigWithReloadInterval(300*time.Millisecond),
			xconf.DefaultConfigWithReloadErrorHandler(errHandler),
		)
	)
	requireNil(t, err)
	defer subject.Close()

	// act
	result := subject.Get("foo")

	// assert
	assertEqual(t, "bar", result)
	assertEqual(t, uint32(1), atomic.LoadUint32(&loaderCallsCnt))

	// act
	time.Sleep(500 * time.Millisecond)
	result = subject.Get("foo")

	// assert
	assertEqual(t, "bar", result) // result is still bar, the old value
	assertTrue(t, atomic.LoadUint32(&loaderCallsCnt) > 1)
	assertEqual(t, uint32(1), atomic.LoadUint32(&errHandlerCallsCnt))
}

func testDefaultConfigGetStringKey(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := "baz"
	tests := [...]struct {
		name           string
		loader         xconf.Loader
		expectedResult interface{}
	}{
		{
			name:           "string value",
			loader:         xconf.PlainLoader(map[string]interface{}{"test-string-key": "bar"}),
			expectedResult: "bar",
		},
		{
			name:           "int value",
			loader:         xconf.PlainLoader(map[string]interface{}{"test-string-key": 1234}),
			expectedResult: "1234",
		},
		{
			name:           "uint value",
			loader:         xconf.PlainLoader(map[string]interface{}{"test-string-key": uint(1234)}),
			expectedResult: "1234",
		},
		{
			name:           "float value",
			loader:         xconf.PlainLoader(map[string]interface{}{"test-string-key": 1234.56}),
			expectedResult: "1234.56",
		},
		{
			name:           "bool value",
			loader:         xconf.PlainLoader(map[string]interface{}{"test-string-key": true}),
			expectedResult: "true",
		},
		{
			name: "non-convertible value return default",
			loader: xconf.LoaderFunc(func() (map[string]interface{}, error) {
				// Note: this case should never arise, no current implemented loaders can produce such a value.
				return map[string]interface{}{"test-string-key": func() {}}, nil
			}),
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(test.loader)
			requireNil(t, err)

			// act
			result := subject.Get("test-string-key", defaultValue)
			_, isExpectedType := result.(string)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetIntKey(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := 999
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "int value",
			value:          1234,
			expectedResult: 1234,
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: 1234,
		},
		{
			name:           "string value",
			value:          "1234",
			expectedResult: 1234,
		},
		{
			name:           "float value",
			value:          1234.56,
			expectedResult: 1234,
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: 1,
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: 0,
		},
		{
			name:           "non-convertible value return default",
			value:          "not an int64",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-int-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-int-key", defaultValue)
			_, isExpectedType := result.(int)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetInt64Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := int64(999)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "int64 value",
			value:          int64(1234),
			expectedResult: int64(1234),
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: int64(1234),
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: int64(1234),
		},
		{
			name:           "string value",
			value:          "1234",
			expectedResult: int64(1234),
		},
		{
			name:           "float value",
			value:          1234.56,
			expectedResult: int64(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: int64(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: int64(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an int64",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-int64-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-int64-key", defaultValue)
			_, isExpectedType := result.(int64)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetInt32Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := int32(999)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "int32 value",
			value:          int32(1234),
			expectedResult: int32(1234),
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: int32(1234),
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: int32(1234),
		},
		{
			name:           "string value",
			value:          "1234",
			expectedResult: int32(1234),
		},
		{
			name:           "float value",
			value:          1234.56,
			expectedResult: int32(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: int32(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: int32(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an int32",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-int32-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-int32-key", defaultValue)
			_, isExpectedType := result.(int32)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetInt16Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := int16(999)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "int16 value",
			value:          int16(1234),
			expectedResult: int16(1234),
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: int16(1234),
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: int16(1234),
		},
		{
			name:           "string value",
			value:          "1234",
			expectedResult: int16(1234),
		},
		{
			name:           "float value",
			value:          1234.56,
			expectedResult: int16(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: int16(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: int16(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an int16",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-int16-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-int16-key", defaultValue)
			_, isExpectedType := result.(int16)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetInt8Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := int8(99)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "int8 value",
			value:          int8(100),
			expectedResult: int8(100),
		},
		{
			name:           "int value",
			value:          100,
			expectedResult: int8(100),
		},
		{
			name:           "uint value",
			value:          uint(100),
			expectedResult: int8(100),
		},
		{
			name:           "string value",
			value:          "100",
			expectedResult: int8(100),
		},
		{
			name:           "float value",
			value:          100.56,
			expectedResult: int8(100),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: int8(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: int8(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an int8",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-int8-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-int8-key", defaultValue)
			_, isExpectedType := result.(int8)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetUintKey(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := uint(999)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: uint(1234),
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: uint(1234),
		},
		{
			name:           "string value",
			value:          "1234",
			expectedResult: uint(1234),
		},
		{
			name:           "float value",
			value:          1234.56,
			expectedResult: uint(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: uint(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: uint(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an uint",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-uint-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-uint-key", defaultValue)
			_, isExpectedType := result.(uint)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetUint64Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := uint64(999)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "uint64 value",
			value:          uint64(1234),
			expectedResult: uint64(1234),
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: uint64(1234),
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: uint64(1234),
		},
		{
			name:           "string value",
			value:          "1234",
			expectedResult: uint64(1234),
		},
		{
			name:           "float value",
			value:          1234.56,
			expectedResult: uint64(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: uint64(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: uint64(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an uint64",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-uint64-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-uint64-key", defaultValue)
			_, isExpectedType := result.(uint64)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetUint32Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := uint32(999)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "uint32 value",
			value:          uint32(1234),
			expectedResult: uint32(1234),
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: uint32(1234),
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: uint32(1234),
		},
		{
			name:           "string value",
			value:          "1234",
			expectedResult: uint32(1234),
		},
		{
			name:           "float value",
			value:          1234.56,
			expectedResult: uint32(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: uint32(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: uint32(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an uint32",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-uint32-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-uint32-key", defaultValue)
			_, isExpectedType := result.(uint32)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetUint16Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := uint16(999)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "uint16 value",
			value:          uint16(1234),
			expectedResult: uint16(1234),
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: uint16(1234),
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: uint16(1234),
		},
		{
			name:           "string value",
			value:          "1234",
			expectedResult: uint16(1234),
		},
		{
			name:           "float value",
			value:          1234.56,
			expectedResult: uint16(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: uint16(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: uint16(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an uint16",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-uint16-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-uint16-key", defaultValue)
			_, isExpectedType := result.(uint16)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetUint8Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := uint8(99)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "uint8 value",
			value:          uint8(100),
			expectedResult: uint8(100),
		},
		{
			name:           "int value",
			value:          100,
			expectedResult: uint8(100),
		},
		{
			name:           "uint value",
			value:          uint(100),
			expectedResult: uint8(100),
		},
		{
			name:           "string value",
			value:          "100",
			expectedResult: uint8(100),
		},
		{
			name:           "float value",
			value:          100.56,
			expectedResult: uint8(100),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: uint8(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: uint8(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not an uint8",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-uint8-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-uint8-key", defaultValue)
			_, isExpectedType := result.(uint8)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetFloat64Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := 999.99
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "float64 value",
			value:          1234.56,
			expectedResult: 1234.56,
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: float64(1234),
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: float64(1234),
		},
		{
			name:           "string value 1",
			value:          "1234.56",
			expectedResult: 1234.56,
		},
		{
			name:           "string value 2",
			value:          "1234",
			expectedResult: float64(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: float64(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: float64(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not a float64",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-float64-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-float64-key", defaultValue)
			_, isExpectedType := result.(float64)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetFloat32Key(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := float32(999.99)
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "float32 value",
			value:          float32(1234.56),
			expectedResult: float32(1234.56),
		},
		{
			name:           "int value",
			value:          1234,
			expectedResult: float32(1234),
		},
		{
			name:           "uint value",
			value:          uint(1234),
			expectedResult: float32(1234),
		},
		{
			name:           "string value 1",
			value:          "1234.56",
			expectedResult: float32(1234.56),
		},
		{
			name:           "string value 2",
			value:          "1234",
			expectedResult: float32(1234),
		},
		{
			name:           "bool true value",
			value:          true,
			expectedResult: float32(1),
		},
		{
			name:           "bool false value",
			value:          false,
			expectedResult: float32(0),
		},
		{
			name:           "non-convertible value return default",
			value:          "not a float32",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-float32-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-float32-key", defaultValue)
			_, isExpectedType := result.(float32)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetBoolKey(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := true
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "bool value - true",
			value:          true,
			expectedResult: true,
		},
		{
			name:           "bool value - false",
			value:          false,
			expectedResult: false,
		},
		{
			name:           "string value - true",
			value:          "true",
			expectedResult: true,
		},
		{
			name:           "string value - false",
			value:          "false",
			expectedResult: false,
		},
		{
			name:           "string value - True",
			value:          "True",
			expectedResult: true,
		},
		{
			name:           "string value - False",
			value:          "False",
			expectedResult: false,
		},
		{
			name:           "string value - 1",
			value:          "1",
			expectedResult: true,
		},
		{
			name:           "string value - 0",
			value:          "0",
			expectedResult: false,
		},
		{
			name:           "int value - 1",
			value:          1,
			expectedResult: true,
		},
		{
			name:           "int value - 0",
			value:          0,
			expectedResult: false,
		},
		{
			name:           "non-convertible value return default",
			value:          "not a bool",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-bool-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-bool-key", defaultValue)
			_, isExpectedType := result.(bool)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetDurationKey(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := 4 * time.Hour
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "duration value",
			value:          7 * time.Minute,
			expectedResult: 7 * time.Minute,
		},
		{
			name:           "string value gets parsed",
			value:          "14s",
			expectedResult: 14 * time.Second,
		},
		{
			name:           "int value - nanoseconds",
			value:          15,
			expectedResult: 15 * time.Nanosecond,
		},
		{
			name:           "non-convertible value return default",
			value:          "not a time.Duration",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-duration-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-duration-key", defaultValue)
			_, isExpectedType := result.(time.Duration)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetTimeKey(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := time.Date(2022, 5, 23, 22, 10, 35, 0, time.UTC) // 23 May 2022 22:10:35
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "time value",
			value:          time.Date(2022, 5, 23, 10, 15, 25, 0, time.UTC),
			expectedResult: time.Date(2022, 5, 23, 10, 15, 25, 0, time.UTC),
		},
		{
			name:           "string value",
			value:          "2022-05-23 18:21:59 +0200",
			expectedResult: time.Date(2022, 5, 23, 16, 21, 59, 0, time.UTC),
		},
		{
			name:           "string RFC3339 value",
			value:          "2022-05-23T18:23:49Z",
			expectedResult: time.Date(2022, 5, 23, 18, 23, 49, 0, time.UTC),
		},
		{
			name:           "only date",
			value:          "2022-05-23",
			expectedResult: time.Date(2022, 5, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			name:           "int Unix Timestamp value",
			value:          1653300750,
			expectedResult: time.Date(2022, 5, 23, 10, 12, 30, 0, time.UTC),
		},
		{
			name:           "non-convertible value return default",
			value:          "2006",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-time-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-time-key", defaultValue)
			resultTime, isExpectedType := result.(time.Time)

			// assert
			if assertTrue(t, isExpectedType) {
				assertEqual(t, test.expectedResult, resultTime.UTC())
			}

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetStringSliceKey(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := []string{"dog", "fox"}
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "string slice value",
			value:          []string{"foo", "bar", "baz"},
			expectedResult: []string{"foo", "bar", "baz"},
		},
		{
			name:           "interface slice value",
			value:          []interface{}{"foo", "bar", "baz"},
			expectedResult: []string{"foo", "bar", "baz"},
		},
		{
			name:           "int slice value",
			value:          []int{1, 2, 3},
			expectedResult: []string{"1", "2", "3"},
		},
		{
			name:           "single string is turned into a slice of len 1",
			value:          "foo",
			expectedResult: []string{"foo"},
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-string-slice-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-string-slice-key", defaultValue)
			_, isExpectedType := result.([]string)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetIntSliceKey(t *testing.T) {
	t.Parallel()

	// arrange
	defaultValue := []int{99, 100}
	tests := [...]struct {
		name           string
		value          interface{}
		expectedResult interface{}
	}{
		{
			name:           "int slice value",
			value:          []int{1, 2, 3},
			expectedResult: []int{1, 2, 3},
		},
		{
			name:           "interface slice int value",
			value:          []interface{}{1, 2, 3},
			expectedResult: []int{1, 2, 3},
		},
		{
			name:           "interface slice string value",
			value:          []interface{}{"1", "2", "3"},
			expectedResult: []int{1, 2, 3},
		},
		{
			name:           "string slice value",
			value:          []string{"1", "2", "3"},
			expectedResult: []int{1, 2, 3},
		},
		{
			name:           "non-convertible value return default",
			value:          "not a slice of ints",
			expectedResult: defaultValue,
		},
	}

	for _, testData := range tests {
		test := testData // capture range variable
		t.Run(test.name, func(t *testing.T) {
			subject, err := xconf.NewDefaultConfig(
				xconf.PlainLoader(map[string]interface{}{"test-int-slice-key": test.value}),
			)
			requireNil(t, err)

			// act
			result := subject.Get("test-int-slice-key", defaultValue)
			_, isExpectedType := result.([]int)

			// assert
			assertEqual(t, test.expectedResult, result)
			assertTrue(t, isExpectedType)

			_ = subject.Close()
		})
	}
}

func testDefaultConfigGetKeyWithNotCoveredDefaultValueType(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"foo": 999,
		})
		subject, err = xconf.NewDefaultConfig(loader)
		defaultValue = map[string]int{"baz": 123456}
	)
	requireNil(t, err)
	defer subject.Close()

	// act
	result := subject.Get("foo", defaultValue)

	// assert
	assertEqual(t, 999, result)
}

func TestDefaultConfig_RegisterObserver(t *testing.T) {
	// setup an env config
	envNames := map[string]string{
		"XCONF_TEST_DEFAULT_CONFIG_FOO_UPDATED":   "foo to update",
		"XCONF_TEST_DEFAULT_CONFIG_FOO_DELETED":   "foo to delete",
		"XCONF_TEST_DEFAULT_CONFIG_FOO_UNTOUCHED": "foo to remain untouched",
		"XCONF_TEST_DEFAULT_CONFIG_FOO_NEW":       "foo to be added later",
	}

	for envName, value := range envNames {
		if envName == "XCONF_TEST_DEFAULT_CONFIG_FOO_NEW" {
			continue
		}
		t.Setenv(envName, value)
	}

	loader := xconf.FilterKVLoader(
		xconf.EnvLoader(),
		xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithPrefix("XCONF_TEST_DEFAULT_CONFIG_FOO_")),
	)
	subject, err := xconf.NewDefaultConfig(
		loader,
		xconf.DefaultConfigWithReloadInterval(100*time.Millisecond),
	)
	requireNil(t, err)
	defer subject.Close()

	// setup 2 observers
	observer1CallsCnt, observer2CallsCnt := 0, 0
	subject.RegisterObserver(configObserverFactory(t, &observer1CallsCnt))
	subject.RegisterObserver(configObserverFactory(t, &observer2CallsCnt))

	// first act & assert
	result1 := subject.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_UPDATED")
	result2 := subject.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_DELETED")
	result3 := subject.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_UNTOUCHED")
	result4 := subject.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_NEW")
	assertEqual(t, "foo to update", result1)
	assertEqual(t, "foo to delete", result2)
	assertEqual(t, "foo to remain untouched", result3)
	assertNil(t, result4)
	assertEqual(t, 0, observer1CallsCnt)
	assertEqual(t, 0, observer2CallsCnt)

	// prepare second act
	if err := os.Setenv("XCONF_TEST_DEFAULT_CONFIG_FOO_UPDATED", "foo got updated"); err != nil {
		t.Fatal("prerequisite failed:", err)
	}
	if err := os.Unsetenv("XCONF_TEST_DEFAULT_CONFIG_FOO_DELETED"); err != nil {
		t.Fatal("prerequisite failed:", err)
	}
	if err := os.Setenv("XCONF_TEST_DEFAULT_CONFIG_FOO_NEW", "foo to be added later"); err != nil {
		t.Fatal("prerequisite failed:", err)
	}
	time.Sleep(300 * time.Millisecond)

	// second act & assert
	result1 = subject.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_UPDATED")
	result2 = subject.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_DELETED")
	result3 = subject.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_UNTOUCHED")
	result4 = subject.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_NEW")
	assertEqual(t, "foo got updated", result1)
	assertNil(t, result2)
	assertEqual(t, "foo to remain untouched", result3)
	assertEqual(t, "foo to be added later", result4)
	assertEqual(t, 1, observer1CallsCnt)
	assertEqual(t, 1, observer2CallsCnt)
}

func configObserverFactory(t *testing.T, observerCallsCount *int) xconf.ConfigObserver {
	return func(cfg xconf.Config, changedKeys ...string) {
		*observerCallsCount++

		// check params
		assertNotNil(t, cfg)
		expectedChangedKeys := map[string]struct{}{
			"XCONF_TEST_DEFAULT_CONFIG_FOO_UPDATED": {},
			"XCONF_TEST_DEFAULT_CONFIG_FOO_DELETED": {},
			"XCONF_TEST_DEFAULT_CONFIG_FOO_NEW":     {},
		}
		assertTrue(t, len(expectedChangedKeys) == len(changedKeys))
		for _, changedKey := range changedKeys {
			_, found := expectedChangedKeys[changedKey]
			assertTrue(t, found)
		}

		// make assertions updated changed keys.
		result1 := cfg.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_UPDATED")
		result2 := cfg.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_DELETED")
		result3 := cfg.Get("XCONF_TEST_DEFAULT_CONFIG_FOO_NEW")
		assertEqual(t, "foo got updated", result1)
		assertNil(t, result2)
		assertEqual(t, "foo to be added later", result3)
	}
}

func TestDefaultConfig_concurrency(t *testing.T) {
	t.Parallel()

	// arrange
	flgSet, flgSetParseErr := setUpFlagSet()
	requireNil(t, flgSetParseErr)
	var (
		customEnv = "XCONF_" + strings.ToUpper(t.Name()) // we update it
		loader    = xconf.AliasLoader(
			xconf.NewMultiLoader(
				true,
				xconf.PlainLoader(map[string]interface{}{
					"USER": "John Doe",
					"TMP":  os.TempDir(),
				}),
				xconf.FilterKVLoader(
					xconf.EnvLoader(),
					xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithPrefix("USER")),
					xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithPrefix("TMP")),
					xconf.FilterKVWhitelistFunc(xconf.FilterExactKeys(customEnv)),
				),
				xconf.NewFileCacheLoader(
					xconf.JSONFileLoader(jsonFilePath),
					jsonFilePath,
				),
				xconf.YAMLFileLoader(yamlFilePath),
				xconf.NewIniFileLoader(iniFilePath),
				xconf.PropertiesFileLoader(propertiesFilePath),
				xconf.FileLoader(tomlFilePath),
				xconf.AlterValueLoader(
					xconf.DotEnvFileLoader(dotEnvFilePath),
					xconf.ToStringList(","),
					"DOTENV_SHOPPING_LIST",
				),
				xconf.IgnoreErrorLoader(
					xconf.PropertiesFileLoader("/this/properties/config/path/does/not/exist"),
					os.ErrNotExist,
				),
				xconf.FlagSetLoader(flgSet),
			),
			"alias_for_json_foo", "json_foo",
			"alias_for_yaml_year", "yaml_year",
		)
		subject, err = xconf.NewDefaultConfig(
			loader,
			xconf.DefaultConfigWithIgnoreCaseSensitivity(),
			xconf.DefaultConfigWithReloadInterval(100*time.Millisecond), // set a low interval
		)
		wg sync.WaitGroup
	)
	requireNil(t, err)
	defer subject.Close()

	// add also 2 observers
	subject.RegisterObserver(getDummyConfigObserver(t, "observer #1"))
	subject.RegisterObserver(getDummyConfigObserver(t, "observer #2"))

	// act & assert
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func(cfg xconf.Config, waitGr *sync.WaitGroup) {
			defer waitGr.Done()

			// access 3 keys while config object may reload
			for i := 0; i < 50; i++ {
				result := cfg.Get("DOTENV_FOO")
				assertEqual(t, "bar", result)

				result = cfg.Get("json_foo")
				assertEqual(t, "bar", result)

				result = cfg.Get("alias_for_json_foo", "default val")
				assertEqual(t, "bar", result)
			}
		}(subject, &wg)
	}

	// start a goroutine that updates the custom env.
	wg.Add(1)
	go func(envName string, waitGr *sync.WaitGroup) {
		for i := 0; i < 5; i++ {
			envVal := "this is a test: " + strconv.FormatInt(time.Now().UnixNano(), 10)
			_ = os.Setenv(envName, envVal)
			time.Sleep(150 * time.Millisecond)
		}

		_ = os.Unsetenv(envName)
		waitGr.Done()
	}(customEnv, &wg)

	wg.Wait()
}

func getDummyConfigObserver(t *testing.T, name string) xconf.ConfigObserver {
	return func(cfg xconf.Config, changedKeys ...string) {
		for _, changedKey := range changedKeys {
			val := cfg.Get(changedKey, "some default value for this key from "+name)
			t.Logf(`%s : key "%s" is now "%s"`, name, changedKey, val)
		}
	}
}

func benchmarkDefaultConfigGet(withReload, withDefValue bool) func(b *testing.B) {
	// Note: the difference between with/without reload comes from
	// calling/not calling a Mutex.
	// The extra allocation between with/without default value comes from casting.
	// TODO: maybe think at a solution not to have this allocation.
	return func(b *testing.B) {
		b.Helper()
		var (
			loader = xconf.PlainLoader(map[string]interface{}{
				"foo": "bar",
			})
			opts []xconf.DefaultConfigOption
		)
		if withReload {
			opts = []xconf.DefaultConfigOption{xconf.DefaultConfigWithReloadInterval(100 * time.Millisecond)}
		}
		subject, err := xconf.NewDefaultConfig(loader, opts...)
		if err != nil {
			b.Error(err)
			b.FailNow()
		}
		defer subject.Close()

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if withDefValue {
					_ = subject.Get("foo", "baz")
				} else {
					_ = subject.Get("foo")
				}
			}
		})
	}
}

func BenchmarkDefaultConfig_Get_noDefaultValue_withoutReload(b *testing.B) {
	benchmarkDefaultConfigGet(false, false)(b)
}

func BenchmarkDefaultConfig_Get_noDefaultValue_withReload(b *testing.B) {
	benchmarkDefaultConfigGet(true, false)(b)
}

func BenchmarkDefaultConfig_Get_withDefaultValue_withoutReload(b *testing.B) {
	benchmarkDefaultConfigGet(false, true)(b)
}

func BenchmarkDefaultConfig_Get_withDefaultValue_withReload(b *testing.B) {
	benchmarkDefaultConfigGet(true, true)(b)
}

func ExampleDefaultConfig() {
	loader := xconf.NewMultiLoader(
		true,
		xconf.EnvLoader(),
		xconf.JSONFileLoader("testdata/config.json"),
	)
	cfg, err := xconf.NewDefaultConfig(
		loader,
		xconf.DefaultConfigWithIgnoreCaseSensitivity(),
		xconf.DefaultConfigWithReloadInterval(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer cfg.Close()

	fmt.Println(cfg.Get("json_foo", "baz"))

	// Output:
	// bar
}
