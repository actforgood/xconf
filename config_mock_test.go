// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"testing"

	"github.com/actforgood/xconf"
)

func TestMockConfig(t *testing.T) {
	t.Parallel()

	// arrange
	var _ xconf.Config = (*xconf.MockConfig)(nil) // test it implements its contract
	subject := xconf.NewMockConfig(
		"foo", "bar",
		"year", 2022,
		100, "not a string key", // test that this is skipped, as key is not string.
		"odd number of elements", // test that this is skipped, as elements no. is odd.
	)
	subject.SetGetCallback(func(key string, def ...any) {
		switch subject.GetCallsCount() {
		case 1:
			assertEqual(t, "foo", key)
			assertNil(t, def)
		case 2:
			assertEqual(t, "year", key)
			if assertEqual(t, 1, len(def)) {
				assertEqual(t, 2099, def[0])
			}
		case 3:
			assertEqual(t, "non-existent-key", key)
			if assertEqual(t, 1, len(def)) {
				assertEqual(t, "default value", def[0])
			}
		case 4:
			assertEqual(t, "odd number of elements", key)
			assertNil(t, def)
		case 5:
			assertEqual(t, "foo", key)
			assertNil(t, def)
		}
	})

	// act
	resultFoo := subject.Get("foo")
	resultYear := subject.Get("year", 2099)
	resultDefault := subject.Get("non-existent-key", "default value")
	resultNil := subject.Get("odd number of elements")
	subject.SetKeyValues("foo", "baz") // reset a key
	resultFooReset := subject.Get("foo")

	// assert
	assertEqual(t, "bar", resultFoo)
	assertEqual(t, "baz", resultFooReset)
	assertEqual(t, 2022, resultYear)
	assertEqual(t, "default value", resultDefault)
	assertNil(t, resultNil)
	assertEqual(t, 5, subject.GetCallsCount())
}
