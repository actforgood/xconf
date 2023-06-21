// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

// Loader is responsible for loading a configuration
// key value map.
type Loader interface {
	// Load returns a configuration key value map or an error.
	//
	// It's Loader's responsibility to return a map that is safe for
	// an eventual later mutation (decorator pattern can be used to
	// modify a loader's returned configuration key value map and
	// that's why this must/should be accomplished safely; safely
	// from concurrency point of view / data integrity point of view;
	// in other words, Loader should return a disposable config map -
	// see also DeepCopyConfigMap utility and current usages as example).
	Load() (map[string]any, error)
}

// The LoaderFunc type is an adapter to allow the use of
// ordinary functions as Loaders. If fn is a function
// with the appropriate signature, LoaderFunc(fn) is a
// Loader that calls fn.
type LoaderFunc func() (map[string]any, error)

// Load calls fn().
func (fn LoaderFunc) Load() (map[string]any, error) {
	return fn()
}
