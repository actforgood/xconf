//go:build integration

// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
	const key = "json-key"
	const format = xconf.RemoteValueJSON

	t.Run("single key", testEtcdLoaderIntegration(format, key, false))
	t.Run("prefix key", testEtcdLoaderIntegration(format, key, true))
}

func TestEtcdLoader_withYAML_integration(t *testing.T) {
	const key = "yaml-key"
	const format = xconf.RemoteValueYAML

	t.Run("single key", testEtcdLoaderIntegration(format, key, false))
	t.Run("prefix key", testEtcdLoaderIntegration(format, key, true))
}

func TestEtcdLoader_withPlain_integration(t *testing.T) {
	const key = "plain-key"
	const format = xconf.RemoteValuePlain

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
		t.Fatal("prerequisites failed: could not setup etcd client:", err)
	}
	defer setUpClient.Close()
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO", "foo"); err != nil {
		t.Fatal("prerequisites failed: could not create 'foo' key:", err)
	}
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_BAR", "bar"); err != nil {
		t.Fatal("prerequisites failed: could not create 'bar' key:", err)
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
		t.Fatal("prerequisites failed: could not update 'foo' key:", err)
	}
	if _, err := setUpClient.Delete(ctx, "ETCD_TEST_INTEGRATION_WATCH_BAR"); err != nil {
		t.Fatal("prerequisites failed: could not delete 'bar' key:", err)
	}
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_BAZ", "baz"); err != nil {
		t.Fatal("prerequisites failed: could not create 'baz' key:", err)
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
		t.Fatal("prerequisites failed: could not setup etcd client:", err)
	}
	defer setUpClient.Close()
	if _, err := setUpClient.Put(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO_JSON", `{"etcd_foo":"bar"}`); err != nil {
		t.Fatal("prerequisites failed: could not create 'foo' json key:", err)
	}
	defer func() {
		_, _ = setUpClient.Delete(ctx, "ETCD_TEST_INTEGRATION_WATCH_FOO_JSON") // rm the key we played with.
	}()

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
		t.Fatal("prerequisites failed: could not update 'foo' json key:", err)
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
		t.Fatal("prerequisites failed: could not update 'foo' json key:", err)
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

func TestEtcdLoader_withTLS_integration(t *testing.T) {
	// arrange
	endpoints, tlsConfig, err := getEtcdTLSInfo()
	if err != nil {
		t.Fatal(err)
	}
	const key = "plain-key"
	const format = xconf.RemoteValuePlain
	ctx, cancelCtx := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelCtx()
	opts := []xconf.EtcdLoaderOption{
		xconf.EtcdLoaderWithValueFormat(format),
		xconf.EtcdLoaderWithContext(ctx),
		xconf.EtcdLoaderWithPrefix(),
		xconf.EtcdLoaderWithEndpoints(endpoints),
		xconf.EtcdLoaderWithTLS(tlsConfig),
	}
	subject := xconf.NewEtcdLoader(key, opts...)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, getEtcdExpectedConfigMapIntegration(format, true), config)
}

// getEtcdExpectedConfigMapIntegration returns expected config maps for integration tests.
func getEtcdExpectedConfigMapIntegration(format string, withPrefix bool) map[string]interface{} {
	return getConsulExpectedConfigMapIntegration(format, withPrefix) // same as consul...
}

// getDefaultEtcdEndpoints tries to get etcd endpoints from ENV.
// It defaults on localhost address.
func getDefaultEtcdEndpoints() (endpoints []string) {
	// try to get from env variables
	if eps := os.Getenv("ETCD_ENDPOINTS"); eps != "" {
		endpoints = strings.Split(eps, ",")
	} else {
		endpoints = []string{"127.0.0.1:2379"}
	}

	return
}

// getEtcdTLSInfo returns etcd endpoint and config.
func getEtcdTLSInfo() ([]string, *tls.Config, error) {
	var (
		tlsCfg         = new(tls.Config)
		endpoints      []string
		caCertFilePath string
	)
	const defaultCaCertFilePath = "scripts/tls/certs/ca_cert.pem"

	if eps := os.Getenv("ETCDS_ENDPOINTS"); eps != "" {
		endpoints = strings.Split(eps, ",")
		hostname, _, err := net.SplitHostPort(endpoints[0])
		if err != nil {
			return nil, nil, fmt.Errorf("could not parse address %q: %w", endpoints[0], err)
		}
		tlsCfg.ServerName = hostname
		caCertFilePath = fmt.Sprintf(
			"%s%c%s",
			os.Getenv("GITHUB_WORKSPACE"), os.PathSeparator, defaultCaCertFilePath,
		)
	} else {
		endpoints = []string{"localhost:2389"}
		tlsCfg.ServerName = "localhost"
		caCertFilePath = defaultCaCertFilePath
	}

	certContent, err := os.ReadFile(caCertFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read CA cert: %w", err)
	}
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(certContent); !ok {
		return nil, nil, fmt.Errorf("could not parse PEM CA certificate %q", caCertFilePath)
	}
	tlsCfg.MinVersion = tls.VersionTLS12
	tlsCfg.RootCAs = certPool

	return endpoints, tlsCfg, nil
}
