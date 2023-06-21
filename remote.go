// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"bytes"
	"encoding/json"

	"gopkg.in/yaml.v3"
)

const (
	// RemoteValueJSON indicates that content under a key is in JSON format.
	RemoteValueJSON = "json"
	// RemoteValueYAML indicates that content under a key is in YAML format.
	RemoteValueYAML = "yaml"
	// RemoteValuePlain indicates that content under a key is plain text.
	RemoteValuePlain = "plain"
)

// getRemoteKVPairConfigMap returns configuration map for a key, according to format.
func getRemoteKVPairConfigMap(key string, value []byte, format string) (map[string]any, error) {
	var (
		configMap map[string]any
		err       error
	)
	switch format {
	case RemoteValueJSON:
		if err = json.Unmarshal(value, &configMap); err != nil {
			return nil, err
		}
	case RemoteValueYAML:
		if err = yaml.Unmarshal(value, &configMap); err != nil {
			return nil, err
		}
	default: // plain
		configMap = map[string]any{
			key: string(bytes.TrimSpace(value)),
		}
	}

	return configMap, nil
}
