// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

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
	// keyFunc is a function that returns the final configuration key name
	// based on a section and a key under it.
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
		for _, key := range section.Keys() {
			keyName := loader.keyFunc(section.Name(), key.Name())
			configMap[keyName] = key.Value()
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
// By default a function that returns the same key for default section, and <section/key>
// for a different section from default is used.
//
// You may want for example to provide a custom function that ignores the section:
//		xconf.IniFileLoaderWithSectionKeyFunc(func(_, key string) string {
//			return key
// 		})
//
func IniFileLoaderWithSectionKeyFunc(keyFunc func(section, key string) string) IniFileLoaderOption {
	return func(loader *IniFileLoader) {
		loader.keyFunc = keyFunc
	}
}

// defaultKeyFunc is the default implementation for providing the key name
// in the final configuration key-value map for an ini key under a section.
// Example: given the ini content:
//
// 		foo=bar
// 		[time]
// 		year=2022
//
// it will produce "foo" and "time/year" for the 2 above keys.
var defaultIniKeyFunc = func(section, key string) string {
	if section == ini.DefaultSection {
		return key
	}

	return section + "/" + key
}
