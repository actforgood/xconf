// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"os"
)

// EnvLoader loads configuration from OS's ENV.
func EnvLoader() Loader {
	return LoaderFunc(func() (map[string]any, error) {
		envs := os.Environ()

		configMap := make(map[string]any, len(envs))
		const kvSeparator = '='
		for _, env := range envs {
			for i := range len(env) {
				if env[i] == kvSeparator {
					configMap[env[:i]] = env[i+1:]

					break
				}
			}
		}

		return configMap, nil
	})
}
