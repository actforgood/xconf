// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf_test

import (
	"testing"

	"github.com/actforgood/xconf"
)

func TestNopConfig(t *testing.T) {
	t.Parallel()

	// arrange
	var _ xconf.Config = (*xconf.NopConfig)(nil) // test it implements its contract
	subject := xconf.NopConfig{}

	// act
	resultNil := subject.Get("some-key-1")
	resultDefault := subject.Get("some-key-2", "default value")

	// assert
	assertNil(t, resultNil)
	assertEqual(t, "default value", resultDefault)
}
