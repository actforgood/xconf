// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"io"
	"os"

	"github.com/joho/godotenv"
)

// DotEnvFileLoader loads .env configuration from a file.
// The location of .env content based file is given as parameter.
func DotEnvFileLoader(filePath string) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		return DotEnvReaderLoader(f).Load()
	})
}

// DotEnvReaderLoader loads .env configuration from an [io.Reader].
func DotEnvReaderLoader(reader io.Reader) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		if seekReader, ok := reader.(io.Seeker); ok {
			_, _ = seekReader.Seek(0, io.SeekStart) // move to the beginning in case of a re-load needed.
		}
		envs, err := godotenv.Parse(reader)
		if err != nil {
			return nil, err
		}

		configMap := make(map[string]interface{}, len(envs))
		for key, value := range envs {
			configMap[key] = value
		}

		return configMap, nil
	})
}
