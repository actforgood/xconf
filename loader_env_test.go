// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/actforgood/xconf"
)

func TestEnvLoader(t *testing.T) {
	t.Run("success - os env gets loaded", testEnvLoaderSuccess)
	t.Run("success - safe-mutable config map", testEnvLoaderReturnsSafeMutableConfigMap)
}

func testEnvLoaderSuccess(t *testing.T) {
	// arrange
	subject := xconf.EnvLoader()
	envName := getRandomEnvName()
	t.Setenv(envName, "bar")

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	if assertTrue(t, len(config) >= 1) {
		assertEqual(t, "bar", config[envName])
	}
}

func testEnvLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	// arrange
	subject := xconf.EnvLoader()
	envName := getRandomEnvName()
	t.Setenv(envName, "bar")

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	if assertTrue(t, len(config1) >= 1) {
		assertEqual(t, "bar", config1[envName])
	}

	// modify first returned value, expect second returned value to be initial one.
	config1[envName] = "baz"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	if assertTrue(t, len(config1) >= 1) {
		assertEqual(t, "bar", config2[envName])
	}
}

// setUpEnv sets OS env with provided value.
// Returns the previous value, if env name already exists.
func setUpEnv(envName, value string) string {
	prevValue := os.Getenv(envName)
	_ = os.Setenv(envName, value)

	return prevValue
}

// tearDownEnv unsets the OS env provided or restores previous value.
func tearDownEnv(envName, prevValue string) {
	if prevValue != "" {
		_ = os.Setenv(envName, prevValue)
	} else {
		_ = os.Unsetenv(envName)
	}
}

// getRandomEnvName returns a "XCONF_TEST_ENV_LOADER_FOO_<randomInt>" env name.
func getRandomEnvName() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(9999999))
	if err != nil {
		return ""
	}
	randInt := nBig.Int64()

	return "XCONF_TEST_ENV_LOADER_FOO_" + strconv.FormatInt(randInt, 10)
}

func BenchmarkEnvLoader(b *testing.B) {
	subject := xconf.EnvLoader()

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleEnvLoader() {
	// setup an env
	envName := getRandomEnvName()
	prevValue := setUpEnv(envName, "bar")
	defer tearDownEnv(envName, prevValue)

	loader := xconf.EnvLoader()

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	fmt.Println(configMap[envName])

	// Output:
	// bar
}
