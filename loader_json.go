// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"encoding/json"
	"io"
	"os"
)

// JSONFileLoader loads JSON configuration from a file.
// The location of JSON content based file is given as parameter.
func JSONFileLoader(filePath string) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		return JSONReaderLoader(f).Load()
	})
}

// JSONReaderLoader loads JSON configuration from an [io.Reader].
func JSONReaderLoader(reader io.Reader) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		if seekReader, ok := reader.(io.Seeker); ok {
			_, _ = seekReader.Seek(0, io.SeekStart) // move to the beginning in case of a re-load needed.
		}
		var configMap map[string]interface{}
		dec := json.NewDecoder(reader)
		if err := dec.Decode(&configMap); err != nil {
			return nil, err
		}

		return configMap, nil
	})
}
