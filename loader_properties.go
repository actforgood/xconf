// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"os"

	"github.com/magiconair/properties"
)

// PropertiesFileLoader loads Java Properties configuration from a file.
// The location of properties content based file is given as parameter.
func PropertiesFileLoader(filePath string) Loader {
	return LoaderFunc(func() (map[string]any, error) {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		return PropertiesBytesLoader(content).Load()
	})
}

// PropertiesBytesLoader loads Properties configuration from bytes.
func PropertiesBytesLoader(propertiesContent []byte) Loader {
	return LoaderFunc(func() (map[string]any, error) {
		loader := properties.Loader{
			Encoding:         properties.UTF8,
			DisableExpansion: false,
		}
		cfg, err := loader.LoadBytes(propertiesContent)
		if err != nil {
			return nil, err
		}
		keys := cfg.Keys()

		configMap := make(map[string]any, len(keys))
		for _, key := range keys {
			value, _ := cfg.Get(key)
			configMap[key] = value
		}

		return configMap, nil
	})
}
