// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

// NopConfig is a no-operation xconf.Config.
type NopConfig struct{}

// Get returns default value, if present, or nil.
func (NopConfig) Get(_ string, def ...any) any {
	if len(def) > 0 {
		return def[0]
	}

	return nil
}
