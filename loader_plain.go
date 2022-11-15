// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

// PlainLoader is an explicit go configuration map retriever.
// It simply returns a copy of the given config map parameter.
//
// It can be used for example:
//
// -  in a [MultiLoader] (with allowing keys overwrite) as the first loader
// in order to specify default configurations.
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
