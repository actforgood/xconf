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
}

// NewIniFileLoader instantiates a new IniFileLoader object that loads
// INI configuration from a file.
// The location of INI content based file is given as parameter.
func NewIniFileLoader(filePath string, opts ...IniFileLoaderOption) IniFileLoader {
	loader := IniFileLoader{
		filePath: filePath,
		loadOpts: ini.LoadOptions{},
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
