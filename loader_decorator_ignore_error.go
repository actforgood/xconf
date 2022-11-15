// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"errors"
)

// IgnoreErrorLoader decorates another loader to ignore the error returned by it,
// if error is present in the list of errors passed as second parameter.
// You can ignore, for example, [os.ErrNotExist] for a file based Loader if that file is not
// mandatory to exist, or Consul's [ErrConsulKeyNotFound], etc.
func IgnoreErrorLoader(loader Loader, errs ...error) Loader {
	return LoaderFunc(func() (map[string]interface{}, error) {
		configMap, err := loader.Load()
		if err != nil {
			for _, ignoreErr := range errs {
				if errors.Is(err, ignoreErr) {
					return map[string]interface{}{}, nil
				}
			}
		}

		return configMap, err
	})
}
