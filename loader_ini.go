// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"gopkg.in/ini.v1"
)

// IniFileLoader is a loader that returns configuration from
// an INI content based file.
type IniFileLoader struct {
	// filePath is ini content based file to be parsed.
	filePath string
	// loadOpts are the original package parse options.
	loadOpts ini.LoadOptions
	// keyFunc is a function that returns a flatten key name
	// based on a section and a key under it.
	// Deprecated: to be removed in a future release.
	// [FlatternLoader] can be used to achieve what this function does.
	keyFunc func(section, key string) string
}

// NewIniFileLoader instantiates a new IniFileLoader object that loads
// INI configuration from a file.
// The location of INI content based file is given as parameter.
func NewIniFileLoader(filePath string, opts ...IniFileLoaderOption) IniFileLoader {
	loader := IniFileLoader{
		filePath: filePath,
		loadOpts: ini.LoadOptions{},
		keyFunc:  defaultIniKeyFunc,
	}

	// apply options, if any.
	for _, opt := range opts {
		opt(&loader)
	}

	return loader
}

// Load returns a configuration key-value map from a INI file,
// or an error if something bad happens along the process.
func (loader IniFileLoader) Load() (map[string]interface{}, error) {
	cfg, err := ini.LoadSources(loader.loadOpts, loader.filePath)
	if err != nil {
		return nil, err
	}

	configMap := make(map[string]interface{})
	sections := cfg.Sections()
	for _, section := range sections {
		sectionKeys := section.Keys()
		if section.Name() != ini.DefaultSection {
			configMap[section.Name()] = make(map[string]interface{}, len(sectionKeys))
		}
		for _, key := range sectionKeys {
			// deprecation, to be removed - start
			keyName := loader.keyFunc(section.Name(), key.Name())
			configMap[keyName] = key.Value()
			// deprecation, to be removed - stop

			if section.Name() == ini.DefaultSection {
				configMap[key.Name()] = key.Value()
			} else {
				configMap[section.Name()].(map[string]interface{})[key.Name()] = key.Value()
			}
		}
	}

	return configMap, nil
}

// IniFileLoaderOption defines optional function for configuring
// an INI File Loader.
type IniFileLoaderOption func(*IniFileLoader)

// IniFileLoaderWithLoadOptions sets given ini load options on the loader.
// By default, an empty object is used.
func IniFileLoaderWithLoadOptions(iniLoadOpts ini.LoadOptions) IniFileLoaderOption {
	return func(loader *IniFileLoader) {
		loader.loadOpts = iniLoadOpts
	}
}

// IniFileLoaderWithSectionKeyFunc sets given configuration key name provider based
// on a key and the section it belongs to.
//
// Deprecated: do not use it anymore, it will be removed in a future release!
// You can use [FlattenLoader] wrapper to get flatten keys when they belong to a section
// different from 'default' section, if needed.
// Neither current usage of accessing keys should be avoided.
// Example: for a key "bar" in section "[foo]"" deprecated implementation gives access to a key "foo/bar".
// This kind of key generation will be removed. You should use new implementation output in the form of map ({"foo": {"bar": ...}}).
func IniFileLoaderWithSectionKeyFunc(keyFunc func(section, key string) string) IniFileLoaderOption {
	return func(loader *IniFileLoader) {
		loader.keyFunc = keyFunc
	}
}

// defaultKeyFunc is the default implementation for providing the key name
// in the final configuration key-value map for an ini key under a section.
// Example: given the ini content:
//
//	foo=bar
//	[time]
//	year=2022
//
// it will produce "foo" and "time/year" for the 2 above keys.
//
// Deprecated: to be removed along with IniFileLoaderWithSectionKeyFunc logic.
var defaultIniKeyFunc = func(section, key string) string {
	if section == ini.DefaultSection {
		return key
	}

	return section + "/" + key
}
