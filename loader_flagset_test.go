// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"flag"
	"fmt"
	"io"
	"testing"

	"github.com/actforgood/xconf"
)

var flagSetConfigMap = map[string]interface{}{
	"flag_foo":           "bar",
	"flag_year":          "2022",
	"flag_temperature":   "37.5",
	"flag_shopping_list": "bread,milk,eggs",
}

func TestFlagSetLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - visit all", testFlagSetLoaderVisitAll)
	t.Run("success - visit only set", testFlagSetLoaderVisitOnlySet)
	t.Run("success - safe-mutable config map", testFlagSetLoaderReturnsSafeMutableConfigMap)
}

func testFlagSetLoaderVisitAll(t *testing.T) {
	t.Parallel()

	// arrange
	flgSet, err := setUpFlagSet()
	requireNil(t, err)
	subject := xconf.FlagSetLoader(flgSet)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, flagSetConfigMap, config)
}

func testFlagSetLoaderVisitOnlySet(t *testing.T) {
	t.Parallel()

	// arrange
	flgSet, err := setUpFlagSet()
	requireNil(t, err)
	subject := xconf.FlagSetLoader(flgSet, false)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"flag_foo":         "bar",
			"flag_temperature": "37.5",
		},
		config,
	)
}

func testFlagSetLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	flgSet, err := setUpFlagSet()
	requireNil(t, err)
	subject := xconf.FlagSetLoader(flgSet)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, flagSetConfigMap, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["flag_foo"] = "test flag string modified"
	config1["flag_year"] = "2099"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, flagSetConfigMap, config2)

	assertEqual(
		t,
		map[string]interface{}{
			"flag_foo":           "bar",
			"flag_year":          "2022",
			"flag_temperature":   "37.5",
			"flag_shopping_list": "bread,milk,eggs",
		},
		flagSetConfigMap,
	)
}

// setUpFlagSet sets up the flag set used in tests.
func setUpFlagSet() (*flag.FlagSet, error) {
	flgSet := flag.NewFlagSet("test-flag-set", flag.ContinueOnError)
	flgSet.SetOutput(io.Discard)

	_ = flgSet.String("flag_foo", "baz", "foo...")
	_ = flgSet.Int("flag_year", 2022, "year...")
	_ = flgSet.Float64("flag_temperature", 37.5, "temperature...")
	_ = flgSet.String("flag_shopping_list", "bread,milk,eggs", "shopping list...")

	args := []string{"-flag_foo=bar", "-flag_temperature=37.5"}

	return flgSet, flgSet.Parse(args)
}

func BenchmarkFlagSetLoader(b *testing.B) {
	flgSet, err := setUpFlagSet()
	if err != nil {
		b.Fatal(err)
	}
	subject := xconf.FlagSetLoader(flgSet)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleFlagSetLoader() {
	// setup a test flag set
	flgSet := flag.NewFlagSet("example-flag-set", flag.ContinueOnError)
	_ = flgSet.String("flag_foo", "baz", "foo description")
	_ = flgSet.Int("flag_year", 2022, "year description")
	_ = flgSet.Float64("flag_temperature", 37.5, "temperature description")
	_ = flgSet.String("flag_shopping_list", "bread,milk,eggs", "shopping list description")
	args := []string{"-flag_foo=bar", "-flag_temperature=37.5"} // you will usually pass os.Args[1:] here
	if err := flgSet.Parse(args); err != nil {
		panic(err)
	}

	loader := xconf.FlagSetLoader(flgSet)
	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// flag_foo: bar
	// flag_year: 2022
	// flag_temperature: 37.5
	// flag_shopping_list: bread,milk,eggs
}
