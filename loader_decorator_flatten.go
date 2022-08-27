// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

import "github.com/spf13/cast"

// FlattenLoader decorates another loader to add shortcuts to leaves' information
// in a nested configuration key.
//
// Example, given the configuration:
//
//	{
//	  "mysql": {
//	    "host": "127.0.0.1",
//	    "port": 3306
//	  }
//	}
//
// 2 additional flat keys will be added to above standard configuration: "mysql.host", "mysql.port"
// for easy access of leaf-keys.
// Note: original nested configuration is still kept by default, if you want to remove it, apply
// FlattenLoaderWithFlatKeysOnly option.
type FlattenLoader struct {
	loader    Loader // original, decorated loader.
	flatOnly  bool   // flag that indicates whether nested keys should be removed and only their flat version should be kept.
	separator string // separator for flat nested keys.
}

// NewFlattenLoader instantiates a new FlattenLoader object that adds
// flat version for nested keys for easily access.
func NewFlattenLoader(loader Loader, opts ...FlattenLoaderOption) FlattenLoader {
	flattenLoader := FlattenLoader{
		loader:    loader,
		flatOnly:  false,
		separator: ".",
	}

	// apply options, if any.
	for _, opt := range opts {
		opt(&flattenLoader)
	}

	return flattenLoader
}

// Load returns a configuration key-value map from original loader, enriched with
// shortcuts to leaves' information in nested configuration key(s).
func (decorator FlattenLoader) Load() (map[string]interface{}, error) {
	configMap, err := decorator.loader.Load()
	if err != nil {
		return configMap, err
	}

	flatConfigMap := configMap
	decorator.flattenConfigMap(0, "", configMap, flatConfigMap)

	return flatConfigMap, nil
}

// getFlatKey returns a flat key representing the concatenation of
// previous (level) key and current (level) key.
func (decorator FlattenLoader) getFlatKey(lvl uint, prevKey, currKey string) string {
	if lvl > 0 {
		return prevKey + decorator.separator + currKey
	}

	return currKey
}

// flattenConfigMap appends flat keys to finalConfigMap,
// and eventually removes nested keys from it.
func (decorator FlattenLoader) flattenConfigMap(
	lvl uint,
	prevKey string,
	currConfigMap map[string]interface{},
	finalConfigMap map[string]interface{},
) {
	for key, value := range currConfigMap {
		switch val := value.(type) {
		case map[string]interface{}:
			decorator.flattenConfigMap(
				lvl+1,
				decorator.getFlatKey(lvl, prevKey, key),
				val,
				finalConfigMap,
			)

			if lvl == 0 && decorator.flatOnly {
				delete(finalConfigMap, key) // don't preserve original (nested configuration) keys
			}
		case map[interface{}]interface{}:
			cfgMap := cast.ToStringMap(val)
			decorator.flattenConfigMap(
				lvl+1,
				decorator.getFlatKey(lvl, prevKey, key),
				cfgMap,
				finalConfigMap,
			)

			if lvl == 0 && decorator.flatOnly {
				delete(finalConfigMap, key) // don't preserve original (nested configuration) keys
			}
		default:
			finalConfigMap[decorator.getFlatKey(lvl, prevKey, key)] = value
		}
	}
}

// FlattenLoaderOption defines optional function for configuring
// a Flatten Loader.
type FlattenLoaderOption func(*FlattenLoader)

// FlattenLoaderWithSeparator sets the separator for the new, flat keys.
// By default, is set to "."(dot).
func FlattenLoaderWithSeparator(keySeparator string) FlattenLoaderOption {
	return func(loader *FlattenLoader) {
		loader.separator = keySeparator
	}
}

// FlattenLoaderWithFlatKeysOnly triggers nested keys to be removed,
// and only their flat version to be kept.
func FlattenLoaderWithFlatKeysOnly() FlattenLoaderOption {
	return func(loader *FlattenLoader) {
		loader.flatOnly = true
	}
}
