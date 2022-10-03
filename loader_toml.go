// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"io"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// TOMLFileLoader loads TOML configuration from a file.
// The location of TOML content based file is given as parameter.
func TOMLFileLoader(filePath string) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		return TOMLReaderLoader(f).Load()
	})
}

// TOMLReaderLoader loads TOML configuration from an io.Reader.
func TOMLReaderLoader(reader io.Reader) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		if seekReader, ok := reader.(io.Seeker); ok {
			_, _ = seekReader.Seek(0, io.SeekStart) // move to the beginning in case of a re-load needed.
		}
		var configMap map[string]interface{}
		dec := toml.NewDecoder(reader)
		if err := dec.Decode(&configMap); err != nil {
			return nil, err
		}

		return configMap, nil
	})
}
