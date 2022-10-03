// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"errors"
	"path/filepath"
)

// ErrUnknownConfigFileExt is an error returned by [FileLoader] if file extension
// does not match any supported format.
var ErrUnknownConfigFileExt = errors.New("unknown configuration file extension")

// FileLoader is a factory for appropriate XFileLoader based on file's extension.
// This is useful when you don't want to tie an application to a certain config format.
// Supported extensions are: .json, .yml, .yaml, .ini, .properties, .env, .toml.
func FileLoader(filePath string) Loader {
	fileExtension := filepath.Ext(filePath)
	switch fileExtension {
	case ".json":
		return JSONFileLoader(filePath)
	case ".yml":
		return YAMLFileLoader(filePath)
	case ".yaml":
		return YAMLFileLoader(filePath)
	case ".env":
		return DotEnvFileLoader(filePath)
	case ".ini":
		return NewIniFileLoader(filePath)
	case ".toml":
		return TOMLFileLoader(filePath)
	case ".properties":
		return PropertiesFileLoader(filePath)
	}

	return LoaderFunc(func() (map[string]interface{}, error) {
		return nil, ErrUnknownConfigFileExt
	})
}
