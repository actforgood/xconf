//go:build integration
// +build integration

// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/actforgood/xconf"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Note: data from this test file can be generated with ./scripts/etcd_data_provider.sh

func TestEtcdLoader_withJSON_integration(t *testing.T) {
	key := "json-key"
	format := xconf.RemoteValueJSON

	t.Run("single key", testEtcdLoaderIntegration(format, key, false))
	t.Run("prefix key", testEtcdLoaderIntegration(format, key, true))
}

func TestEtcdLoader_withYAML_integration(t *testing.T) {
	key := "yaml-key"
	format := xconf.RemoteValueYAML

	t.Run("single key", testEtcdLoaderIntegration(format, key, false))
	t.Run("prefix key", testEtcdLoaderIntegration(format, key, true))
}

func TestEtcdLoader_withPlain_integration(t *testing.T) {
	key := "plain-key"
	format := xconf.RemoteValuePlain

	t.Run("single key", testEtcdLoaderIntegration(format, key, false))
	t.Run("prefix key", testEtcdLoaderIntegration(format, key, true))
}

func TestEtcdLoader_withWatcher_success(t *testing.T) {
	// arrange
	ctx, cancelCtx := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelCtx()

	// setup an aux client that creates/updates/deletes 3 keys we will play with.
	setUpClient, err := clientv3.New(clientv3.Config{
		Endpoints:   getDefaultEtcdEndpoints(),
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Skip("prerequisites failed: could not setup etcd client", err)
	}
	defer setUpClient.Close()
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO", "foo"); err != nil {
		t.Skip("prerequisites failed: could not create 'foo' key", err)
	}
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_BAR", "bar"); err != nil {
		t.Skip("prerequisites failed: could not create 'bar' key", err)
	}
	defer func() { // remove the keys we played with.
		_, _ = setUpClient.Delete(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO")
		_, _ = setUpClient.Delete(ctx, "ETCD_TEST_INTEGRATION_WATCH_BAR")
		_, _ = setUpClient.Delete(ctx, "ETCD_TEST_INTEGRATION_WATCH_BAZ")
	}()

	opts := []xconf.EtcdLoaderOption{
		xconf.EtcdLoaderWithValueFormat(xconf.RemoteValuePlain),
		xconf.EtcdLoaderWithContext(ctx),
		xconf.EtcdLoaderWithWatcher(),
		xconf.EtcdLoaderWithPrefix(),
	}
	subject := xconf.NewEtcdLoader("ETCD_TEST_INTEGRATION_WATCH_", opts...)
	defer subject.Close() // nicely close resources

	// act
	config, err := subject.Load()

	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"ETCD_TEST_INTEGRATION_WATCH_FOO": "foo",
			"ETCD_TEST_INTEGRATION_WATCH_BAR": "bar",
		},
		config,
	)

	// update foo, delete bar, create baz
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO", "foo - updated"); err != nil {
		t.Skip("prerequisites failed: could not update 'foo' key", err)
	}
	if _, err := setUpClient.Delete(ctx, "ETCD_TEST_INTEGRATION_WATCH_BAR"); err != nil {
		t.Skip("prerequisites failed: could not delete 'bar' key", err)
	}
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_BAZ", "baz"); err != nil {
		t.Skip("prerequisites failed: could not create 'baz' key", err)
	}
	expectedNewConfig := map[string]interface{}{
		"ETCD_TEST_INTEGRATION_WATCH_FOO": "foo - updated",
		"ETCD_TEST_INTEGRATION_WATCH_BAZ": "baz", // new key
	}

	// give watcher some time to act
	maxTry := 4
	sleep := time.Second
	for i := 0; i < maxTry; i++ {
		time.Sleep(sleep)

		// act again - see new configuration
		config, err = subject.Load()

		if reflect.DeepEqual(expectedNewConfig, config) || i == maxTry {
			assertNil(t, err)
			assertEqual(t, expectedNewConfig, config)

			break
		}

		sleep *= 2
	}
}

func TestEtcdLoader_withWatcher_error(t *testing.T) {
	// arrange
	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Minute)
	defer cancelCtx()

	// setup an aux client that creates/updates the key we will play with.
	setUpClient, err := clientv3.New(clientv3.Config{
		Endpoints:   getDefaultEtcdEndpoints(),
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Skip("prerequisites failed: could not setup etcd client", err)
	}
	defer setUpClient.Close()
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO_JSON", `{"etcd_foo":"bar"}`); err != nil {
		t.Skip("prerequisites failed: could not create 'foo' json key", err)
	}
	defer setUpClient.Delete(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO_JSON") // rm the key we played with.

	opts := []xconf.EtcdLoaderOption{
		xconf.EtcdLoaderWithValueFormat(xconf.RemoteValueJSON),
		xconf.EtcdLoaderWithContext(ctx),
		xconf.EtcdLoaderWithWatcher(),
	}
	subject := xconf.NewEtcdLoader("ETCD_TEST_INTEGRATION_WATCH_FOO_JSON", opts...)
	defer subject.Close() // nicely close resources

	// act
	config, err := subject.Load()

	assertNil(t, err)
	assertEqual(t, map[string]interface{}{"etcd_foo": "bar"}, config)

	// we update foo, with corrupted json
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO_JSON", "{corrupted json"); err != nil {
		t.Skip("prerequisites failed: could not update 'foo' json key", err)
	}
	// give watcher some time to act
	maxTry := 4
	sleep := time.Second
	for i := 0; i < maxTry; i++ {
		time.Sleep(sleep)

		// act again - see error is returned
		config, err = subject.Load()

		if err != nil || i == maxTry {
			// old "version" is returned, but also error.
			assertEqual(t, map[string]interface{}{"etcd_foo": "bar"}, config)
			var jsonErr *json.SyntaxError
			assertTrue(t, errors.As(err, &jsonErr))

			break
		}

		sleep *= 2
	}

	// we update foo, fixing the json
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO_JSON", `{"etcd_foo":"baz"}`); err != nil {
		t.Skip("prerequisites failed: could not update 'foo' json key", err)
	}
	// give watcher some time to act
	sleep = time.Second
	for i := 0; i < maxTry; i++ {
		time.Sleep(sleep)

		// act again - see everything went back to normal
		config, err = subject.Load()

		if err == nil || i == maxTry {
			assertNil(t, err)
			assertEqual(t, map[string]interface{}{"etcd_foo": "baz"}, config)

			break
		}

		sleep *= 2
	}
}

func testEtcdLoaderIntegration(format, key string, withPrefix bool) func(t *testing.T) {
	return func(t *testing.T) {
		// arrange
		ctx, cancelCtx := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelCtx()

		opts := []xconf.EtcdLoaderOption{
			xconf.EtcdLoaderWithValueFormat(format),
			xconf.EtcdLoaderWithContext(ctx),
		}
		if withPrefix {
			opts = append(opts, xconf.EtcdLoaderWithPrefix())
		}
		subject := xconf.NewEtcdLoader(key, opts...)

		// act
		config, err := subject.Load()

		// assert
		assertNil(t, err)
		assertEqual(t, getEtcdExpectedConfigMapIntegration(format, withPrefix), config)
	}
}

// getEtcdExpectedConfigMapIntegration returns expected config maps for integration tests.
func getEtcdExpectedConfigMapIntegration(format string, withPrefix bool) map[string]interface{} {
	return getConsulExpectedConfigMapIntegration(format, withPrefix) // same as consul...
}

// getDefaultEtcdEndpoints tries to get etcd endpoints from ENV.
// It defaults on localhost address.
func getDefaultEtcdEndpoints() []string {
	endpoints := []string{"127.0.0.1:2379"}

	// try to get from env variables
	if eps := os.Getenv("ETCD_ENDPOINTS"); eps != "" {
		endpoints = strings.Split(eps, ",")
	}

	return endpoints
}
