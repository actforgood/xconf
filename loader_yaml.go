// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLFileLoader loads YAML configuration from a file.
// The location of YAML content based file is given as parameter.
func YAMLFileLoader(filePath string) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		return YAMLReaderLoader(f).Load()
	})
}

// YAMLReaderLoader loads YAML configuration from an io.Reader.
func YAMLReaderLoader(reader io.Reader) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		if seekReader, ok := reader.(io.Seeker); ok {
			_, _ = seekReader.Seek(0, io.SeekStart) // move to the beginning in case of a re-load needed.
		}
		var configMap map[string]interface{}
		dec := yaml.NewDecoder(reader)
		if err := dec.Decode(&configMap); err != nil {
			return nil, err
		}

		return configMap, nil
	})
}
