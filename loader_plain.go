// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

// PlainLoader is an explicit go configuration map retriever.
// It simply returns a copy of the given config map parameter.
//
// It can be used for example:
//
// -  in a MultiLoader (with allowing keys overwrite) as the first loader
// in order to specify default configurations.
//
// - in a MultiLoader  (with allowing keys overwrite) as the last loader
// and provided config map to contain cmd parsed flags (like flags should overwrite
// any other configuration...)
//
// - to provide any application hardcoded configs.
func PlainLoader(configMap map[string]interface{}) Loader {
	// make a copy to preserve state at current time.
	// (prevents user modification of configMap from outside while using the loader).
	configMapCopy := DeepCopyConfigMap(configMap)

	return LoaderFunc(func() (map[string]interface{}, error) {
		return DeepCopyConfigMap(configMapCopy), nil // make a copy for an eventual (safe) later mutation.
	})
}
