// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

import (
	"strings"

	"github.com/spf13/cast"
)

// AlterValueFunc is a function that manipulates a config's value.
type AlterValueFunc func(value interface{}) interface{}

// AlterValueLoader decorates another loader to manipulate a config's value.
// The transformation function is applied to all passed keys.
func AlterValueLoader(loader Loader, transformation AlterValueFunc, keys ...string) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		configMap, err := loader.Load()
		if err != nil {
			return configMap, err
		}

		for _, key := range keys {
			if value, found := configMap[key]; found {
				configMap[key] = transformation(value)
			}
		}

		return configMap, nil
	})
}

// ToStringList makes a slice of strings from a string value,
// who's items are separated by given separator parameter.
//
// If the original value is not a string, the value remains unaltered.
//
// Example: "bread,eggs,milk" => ["bread", "eggs", "milk"].
//
func ToStringList(sep string) AlterValueFunc {
	return func(value interface{}) interface{} {
		if strValue, ok := value.(string); ok {
			return strings.Split(strValue, sep)
		}

		return value
	}
}

// ToIntList makes a slice of integers from a string value,
// who's items are separated by given separator parameter.
//
// If the original value is not a string, the value remains unaltered.
//
// Example: "10,100,1000" => [10, 100, 1000].
//
func ToIntList(sep string) AlterValueFunc {
	return func(value interface{}) interface{} {
		if strValue, ok := value.(string); ok {
			return cast.ToIntSlice(strings.Split(strValue, sep))
		}

		return value
	}
}
