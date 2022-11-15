//go:build integration

// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/actforgood/xconf"
)

// Note: data from this test file can be generated with ./scripts/consul_data_provider.sh

func TestConsulLoader_withJSON_integration(t *testing.T) {
	const key = "json-key"
	const format = xconf.RemoteValueJSON

	t.Run("single key", testConsulLoaderIntegration(format, key, false, ""))
	t.Run("prefix key", testConsulLoaderIntegration(format, key, true, ""))
	t.Run("datacenter query", testConsulLoaderIntegration(format, key, false, "dc1"))
}

func TestConsulLoader_withYAML_integration(t *testing.T) {
	const key = "yaml-key"
	const format = xconf.RemoteValueYAML

	t.Run("single key", testConsulLoaderIntegration(format, key, false, ""))
	t.Run("prefix key", testConsulLoaderIntegration(format, key, true, ""))
	t.Run("datacenter query", testConsulLoaderIntegration(format, key, false, "dc1"))
}

func TestConsulLoader_withPlain_integration(t *testing.T) {
	const key = "plain-key"
	const format = xconf.RemoteValuePlain

	t.Run("single key", testConsulLoaderIntegration(format, key, false, ""))
	t.Run("prefix key", testConsulLoaderIntegration(format, key, true, ""))
	t.Run("datacenter query", testConsulLoaderIntegration(format, key, false, "dc1"))
}

func TestConsulLoader_withNotFoundKey_integration(t *testing.T) {
	// arrange
	subject := xconf.NewConsulLoader("this-key-does-not-exist")

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	assertTrue(t, errors.Is(err, xconf.ErrConsulKeyNotFound))
}

func testConsulLoaderIntegration(format, key string, withPrefix bool, qDataCenter string) func(t *testing.T) {
	return func(t *testing.T) {
		// arrange
		ctx, cancelCtx := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelCtx()

		opts := []xconf.ConsulLoaderOption{
			xconf.ConsulLoaderWithValueFormat(format),
			xconf.ConsulLoaderWithContext(ctx),
		}
		if withPrefix {
			opts = append(opts, xconf.ConsulLoaderWithPrefix())
		}
		if qDataCenter != "" {
			opts = append(opts, xconf.ConsulLoaderWithQueryDataCenter(qDataCenter))
		}
		subject := xconf.NewConsulLoader(key, opts...)

		// act
		config, err := subject.Load()

		// assert
		assertNil(t, err)
		assertEqual(t, getConsulExpectedConfigMapIntegration(format, withPrefix), config)
	}
}

// getConsulExpectedConfigMapIntegration returns expected config maps for integration tests.
func getConsulExpectedConfigMapIntegration(format string, withPrefix bool) map[string]interface{} {
	var expectedConfigMap map[string]interface{}
	switch format {
	case xconf.RemoteValueJSON:
		expectedConfigMap = map[string]interface{}{
			"foo":           "bar",
			"year":          float64(2022),
			"temperature":   37.5,
			"shopping_list": []interface{}{"bread", "milk", "eggs"},
		}
		if withPrefix {
			expectedConfigMap["abc"] = "xyz" // nolint
		}
	case xconf.RemoteValueYAML:
		expectedConfigMap = map[string]interface{}{
			"foo":           "bar",
			"year":          2022,
			"temperature":   37.5,
			"shopping_list": []interface{}{"bread", "milk", "eggs"},
		}
		if withPrefix {
			expectedConfigMap["abc"] = "xyz"
		}
	case xconf.RemoteValuePlain:
		expectedConfigMap = map[string]interface{}{"plain-key": "1000"}
		if withPrefix {
			expectedConfigMap["plain-key/subkey"] = "30s"
		}
	}

	return expectedConfigMap
}
