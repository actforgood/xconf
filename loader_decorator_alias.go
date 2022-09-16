// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import "errors"

// ErrAliasPairBroken is an error returned by AliasLoader when the variadic list of aliases
// and their keys consists of odd no. of elements.
var ErrAliasPairBroken = errors.New("alias - missing key")

// AliasLoader decorates another loader to set aliases for keys.
// The aliases will be added to decorated loader's configuration map.
// The second parameter represents a list of alias and keys they're for
// under the form "aliasForKey1, key1, aliasForKey2, key2".
func AliasLoader(loader Loader, aliasKeyKey ...string) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		if len(aliasKeyKey)%2 == 1 {
			return nil, ErrAliasPairBroken
		}

		configMap, err := loader.Load()
		if err != nil {
			return configMap, err
		}

		for i := 0; i < len(aliasKeyKey); i += 2 {
			alias := aliasKeyKey[i]
			key := aliasKeyKey[i+1]
			if value, found := configMap[key]; found {
				//  Note: here if the alias already exists, it will get overwritten.
				configMap[alias] = value
			}
		}

		return configMap, nil
	})
}
