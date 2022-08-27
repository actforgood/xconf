// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

// NopConfig is a no-operation xconf.Config.
type NopConfig struct{}

// Get returns default value, if present, or nil.
func (NopConfig) Get(_ string, def ...interface{}) interface{} {
	if len(def) > 0 {
		return def[0]
	}

	return nil
}
