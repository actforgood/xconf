// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/actforgood/xconf"
	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type etcdKVServer struct {
	rangeCallback func(context.Context, *pb.RangeRequest) (*pb.RangeResponse, error)
}

func (svr *etcdKVServer) Range(ctx context.Context, req *pb.RangeRequest) (*pb.RangeResponse, error) {
	if svr.rangeCallback != nil {
		return svr.rangeCallback(ctx, req)
	}

	return &pb.RangeResponse{}, nil
}

func (svr *etcdKVServer) Put(context.Context, *pb.PutRequest) (*pb.PutResponse, error) {
	return &pb.PutResponse{}, nil
}

func (svr *etcdKVServer) DeleteRange(context.Context, *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	return &pb.DeleteRangeResponse{}, nil
}

func (svr *etcdKVServer) Txn(context.Context, *pb.TxnRequest) (*pb.TxnResponse, error) {
	return &pb.TxnResponse{}, nil
}

func (svr *etcdKVServer) Compact(context.Context, *pb.CompactionRequest) (*pb.CompactionResponse, error) {
	return &pb.CompactionResponse{}, nil
}

type etcdAuthServer struct {
	*pb.UnimplementedAuthServer
	authenticateCallback func(context.Context, *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error)
}

func (svr etcdAuthServer) Authenticate(
	ctx context.Context,
	req *pb.AuthenticateRequest,
) (*pb.AuthenticateResponse, error) {
	if svr.authenticateCallback != nil {
		return svr.authenticateCallback(ctx, req)
	}

	return &pb.AuthenticateResponse{Token: "mock-token"}, nil
}

// startEtcdKVMockServer starts an etcd key-value grpc mock server.
func startEtcdKVMockServer(
	t *testing.T,
	key string,
	returnedKvs []*mvccpb.KeyValue,
	returnedErr error,
) (*grpc.Server, string) {
	t.Helper()

	rangeCallback := func(_ context.Context, rr *pb.RangeRequest) (*pb.RangeResponse, error) {
		assertEqual(t, key, string(rr.Key))

		if returnedErr != nil {
			return nil, returnedErr
		}

		return &pb.RangeResponse{
			Kvs:   returnedKvs,
			More:  false,
			Count: int64(len(returnedKvs)),
		}, nil
	}
	kvSvr := etcdKVServer{rangeCallback: rangeCallback}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	svr := grpc.NewServer()
	pb.RegisterKVServer(svr, &kvSvr)
	go func(svr *grpc.Server, l net.Listener) {
		_ = svr.Serve(l)
	}(svr, ln)

	return svr, ln.Addr().String()
}

var etcdResponseKeys = map[string]map[bool][]*mvccpb.KeyValue{
	xconf.RemoteValueJSON: {
		true: {
			{
				Key: []byte("etcd_json_key"),
				Value: []byte(`{
					"etcd_json_foo": "bar",
					"etcd_json_year": 2022,
					"etcd_json_temperature": 37.5,
					"etcd_json_shopping_list": [
						"bread",
						"milk",
						"eggs"
					]
				}`),
			},
			{
				Key:   []byte("etcd_json_key/subkey"),
				Value: []byte(`{"etcd_json_abc": "xyz"}`),
			},
		},
		false: {
			{
				Key: []byte("etcd_json_key"),
				Value: []byte(`{
					"etcd_json_foo": "bar",
					"etcd_json_year": 2022,
					"etcd_json_temperature": 37.5,
					"etcd_json_shopping_list": [
				  		"bread",
				  		"milk",
				  		"eggs"
					]
			  	}`),
			},
		},
	},
	xconf.RemoteValueYAML: {
		true: {
			{
				Key: []byte("etcd_yaml_key"),
				Value: []byte(`etcd_yaml_foo: bar
etcd_yaml_year: 2022
etcd_yaml_temperature: 37.5
etcd_yaml_shopping_list:
  - bread
  - milk
  - eggs`),
			},
			{
				Key:   []byte("etcd_yaml_key/subkey"),
				Value: []byte("etcd_yaml_abc: xyz"),
			},
		},
		false: {
			{
				Key: []byte("etcd_yaml_key"),
				Value: []byte(`etcd_yaml_foo: bar
etcd_yaml_year: 2022
etcd_yaml_temperature: 37.5
etcd_yaml_shopping_list:
  - bread
  - milk
  - eggs`),
			},
		},
	},
	xconf.RemoteValuePlain: {
		true: {
			{
				Key:   []byte("etcd_plain_key"),
				Value: []byte("1000"),
			},
			{
				Key:   []byte("etcd_plain_key/subkey"),
				Value: []byte("xyz"),
			},
		},
		false: {
			{
				Key:   []byte("etcd_plain_key"),
				Value: []byte("1000"),
			},
		},
	},
}

var etcdKeys = map[string]string{
	xconf.RemoteValueJSON:  "etcd_json_key",
	xconf.RemoteValueYAML:  "etcd_yaml_key",
	xconf.RemoteValuePlain: "etcd_plain_key",
}

func TestEtcdLoader(t *testing.T) {
	// Note: do not run this test with t.Parallel() as it can affect others by setting ENVs.

	t.Run("success - json single key", testEtcdLoaderByFormatAndPrefix(xconf.RemoteValueJSON, false))
	t.Run("success - json prefix key", testEtcdLoaderByFormatAndPrefix(xconf.RemoteValueJSON, true))
	t.Run("success - yaml single key", testEtcdLoaderByFormatAndPrefix(xconf.RemoteValueYAML, false))
	t.Run("success - yaml prefix key", testEtcdLoaderByFormatAndPrefix(xconf.RemoteValueYAML, true))
	t.Run("success - plain single key", testEtcdLoaderByFormatAndPrefix(xconf.RemoteValuePlain, false))
	t.Run("success - plain prefix key", testEtcdLoaderByFormatAndPrefix(xconf.RemoteValuePlain, true))
	t.Run("error - client init error", testEtcdLoaderReturnsClientInitErr(false))
	t.Run("error - grpc call fails", testEtcdLoaderReturnsResponseErr(false))
	t.Run("error - json value deserialization fails", testEtcdLoaderReturnsErrFromJSONValueDeserialization(false))
	t.Run("error - yaml value deserialization fails", testEtcdLoaderReturnsErrFromYAMLValueDeserialization)
	t.Run("success - with auth", testEtcdLoaderWithAuth)
	t.Run("success - default etcd endpoints taken from env", testEtcdLoaderWithEndpointsTakenFromEnv)
	t.Run("success - with watcher - init client and config", testEtcdLoaderWithWatcher)
	t.Run("error - with watcher - client init error", testEtcdLoaderReturnsClientInitErr(true))
	t.Run("error - with watcher - grpc call fails", testEtcdLoaderReturnsResponseErr(true))
	t.Run(
		"error - with watcher - json value deserialization fails",
		testEtcdLoaderReturnsErrFromJSONValueDeserialization(true),
	)
	t.Run("success - safe-mutable config map", testEtcdLoaderReturnsSafeMutableConfigMap)
}

func testEtcdLoaderByFormatAndPrefix(format string, withPrefix bool) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		key := etcdKeys[format]
		content := etcdResponseKeys[format][withPrefix]
		svr, addr := startEtcdKVMockServer(t, key, content, nil)
		ctx, cancelCtx := context.WithTimeout(context.Background(), 15*time.Second)
		defer func() {
			cancelCtx()
			svr.Stop()
		}()
		opts := []xconf.EtcdLoaderOption{
			xconf.EtcdLoaderWithEndpoints([]string{addr}),
			xconf.EtcdLoaderWithContext(ctx),
			xconf.EtcdLoaderWithValueFormat(format),
		}
		if withPrefix {
			opts = append(opts, xconf.EtcdLoaderWithPrefix())
		}
		subject := xconf.NewEtcdLoader(key, opts...)
		defer func() {
			err := subject.Close()
			assertNil(t, err)
		}()

		// act
		config, err := subject.Load()

		// assert
		assertNil(t, err)
		assertEqual(t, getEtcdExpectedConfigMapByFormatAndPrefix(format, withPrefix), config)
	}
}

func testEtcdLoaderReturnsClientInitErr(withWatcher bool) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		opts := []xconf.EtcdLoaderOption{xconf.EtcdLoaderWithEndpoints([]string{})}
		if withWatcher {
			opts = append(opts, xconf.EtcdLoaderWithWatcher())
		}
		subject := xconf.NewEtcdLoader("some-key", opts...)
		defer subject.Close()

		// act
		config, err := subject.Load()

		// assert
		assertNil(t, config)
		assertTrue(t, errors.Is(err, clientv3.ErrNoAvailableEndpoints))
	}
}

func testEtcdLoaderReturnsResponseErr(withWatcher bool) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		expectedErr := errors.New("etcd intentionally triggered call error")
		key := "some-etcd-key"
		svr, addr := startEtcdKVMockServer(t, key, nil, expectedErr)
		defer svr.Stop()
		opts := []xconf.EtcdLoaderOption{xconf.EtcdLoaderWithEndpoints([]string{addr})}
		if withWatcher {
			opts = append(opts, xconf.EtcdLoaderWithWatcher())
		}
		subject := xconf.NewEtcdLoader(key, opts...)
		defer subject.Close()

		// act
		config, err := subject.Load()

		// assert
		assertNil(t, config)
		if assertNotNil(t, err) {
			assertTrue(t, strings.Contains(err.Error(), expectedErr.Error()))
		}
	}
}

func testEtcdLoaderReturnsErrFromJSONValueDeserialization(withWatcher bool) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		// arrange
		key := "etcd_json_key_"
		returnedKvs := []*mvccpb.KeyValue{
			{
				Key:   []byte("etcd_json_key_1"),
				Value: []byte(`{"etcd_json_foo": "bar"}`),
			},
			{
				Key:   []byte("etcd_json_key_2"),
				Value: []byte(`{ broken json`),
			},
		}
		svr, addr := startEtcdKVMockServer(t, key, returnedKvs, nil)
		defer svr.Stop()
		opts := []xconf.EtcdLoaderOption{
			xconf.EtcdLoaderWithEndpoints([]string{addr}),
			xconf.EtcdLoaderWithPrefix(),
			xconf.EtcdLoaderWithValueFormat(xconf.RemoteValueJSON),
		}
		if withWatcher {
			opts = append(opts, xconf.EtcdLoaderWithWatcher())
		}
		subject := xconf.NewEtcdLoader(key, opts...)
		defer subject.Close()

		// act
		config, err := subject.Load()

		// assert
		assertNil(t, config)
		var jsonErr *json.SyntaxError
		assertTrue(t, errors.As(err, &jsonErr))
	}
}

func testEtcdLoaderReturnsErrFromYAMLValueDeserialization(t *testing.T) {
	t.Parallel()

	// arrange
	key := "etcd_yaml_key_"
	returnedKvs := []*mvccpb.KeyValue{
		{
			Key:   []byte("etcd_yaml_key_1"),
			Value: []byte("etcd_yaml_foo: bar"),
		},
		{
			Key:   []byte("etcd_yaml_key_2"),
			Value: []byte(`---\ninvalid\n  yaml content`),
		},
	}
	svr, addr := startEtcdKVMockServer(t, key, returnedKvs, nil)
	defer svr.Stop()
	subject := xconf.NewEtcdLoader(
		key,
		xconf.EtcdLoaderWithEndpoints([]string{addr}),
		xconf.EtcdLoaderWithPrefix(),
		xconf.EtcdLoaderWithValueFormat(xconf.RemoteValueYAML),
	)
	defer subject.Close()

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, config)
	var yamlErr *yaml.TypeError
	assertTrue(t, errors.As(err, &yamlErr))
}

func testEtcdLoaderWithAuth(t *testing.T) {
	t.Parallel()

	// arrange
	authenticateCallsCnt := 0
	authUsr, authPwd := "john-doe", "some-secret-pwd"
	authSvr := etcdAuthServer{
		authenticateCallback: func(_ context.Context, req *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error) {
			authenticateCallsCnt++
			assertEqual(t, authUsr, req.Name)
			assertEqual(t, authPwd, req.Password)

			return &pb.AuthenticateResponse{
				Token: "some-token",
			}, nil
		},
	}

	format := xconf.RemoteValuePlain
	withPrefix := false
	key := etcdKeys[format]
	content := etcdResponseKeys[format][withPrefix]
	kvSvr := etcdKVServer{
		rangeCallback: func(_ context.Context, req *pb.RangeRequest) (*pb.RangeResponse, error) {
			assertEqual(t, key, string(req.Key))

			return &pb.RangeResponse{
				Kvs:   content,
				More:  false,
				Count: int64(len(content)),
			}, nil
		},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	svr := grpc.NewServer()
	pb.RegisterKVServer(svr, &kvSvr)
	pb.RegisterAuthServer(svr, &authSvr)
	go func(svr *grpc.Server, l net.Listener) {
		_ = svr.Serve(l)
	}(svr, ln)
	defer svr.Stop()

	subject := xconf.NewEtcdLoader(
		key,
		xconf.EtcdLoaderWithEndpoints([]string{ln.Addr().String()}),
		xconf.EtcdLoaderWithAuth(authUsr, authPwd),
	)
	defer subject.Close()

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, getEtcdExpectedConfigMapByFormatAndPrefix(format, withPrefix), config)
	assertEqual(t, 1, authenticateCallsCnt)
}

func testEtcdLoaderWithEndpointsTakenFromEnv(t *testing.T) {
	// arrange
	format := xconf.RemoteValuePlain
	withPrefix := false
	content := etcdResponseKeys[format][withPrefix]
	key := etcdKeys[format]

	svr, addr := startEtcdKVMockServer(t, key, content, nil)
	defer svr.Stop()

	t.Setenv("ETCD_ENDPOINTS", addr)

	subject := xconf.NewEtcdLoader(key)
	defer subject.Close()

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, getEtcdExpectedConfigMapByFormatAndPrefix(format, withPrefix), config)
}

func testEtcdLoaderWithWatcher(t *testing.T) {
	t.Parallel()
	// Note: this test covers partially this functionality.
	// watching live changes is covered by an integration test.

	// arrange
	format := xconf.RemoteValuePlain
	withPrefix := false
	key := etcdKeys[format]
	content := etcdResponseKeys[format][withPrefix]
	svr, addr := startEtcdKVMockServer(t, key, content, nil)
	ctx, cancelCtx := context.WithTimeout(context.Background(), 15*time.Second)
	defer func() {
		cancelCtx()
		svr.Stop()
	}()
	opts := []xconf.EtcdLoaderOption{
		xconf.EtcdLoaderWithEndpoints([]string{addr}),
		xconf.EtcdLoaderWithContext(ctx),
		xconf.EtcdLoaderWithValueFormat(format),
		xconf.EtcdLoaderWithPrefix(),
		xconf.EtcdLoaderWithWatcher(),
	}
	subject := xconf.NewEtcdLoader(key, opts...)
	defer func() {
		err := subject.Close()
		assertNil(t, err)
	}()

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, getEtcdExpectedConfigMapByFormatAndPrefix(format, withPrefix), config)
}

func testEtcdLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	format := xconf.RemoteValueYAML
	withPrefix := true
	key := etcdKeys[format]
	content := etcdResponseKeys[format][withPrefix]
	svr, addr := startEtcdKVMockServer(t, key, content, nil)
	defer svr.Stop()
	subject := xconf.NewEtcdLoader(
		key,
		xconf.EtcdLoaderWithEndpoints([]string{addr}),
		xconf.EtcdLoaderWithPrefix(),
		xconf.EtcdLoaderWithValueFormat(format),
	)
	defer subject.Close()
	expectedConfig := getEtcdExpectedConfigMapByFormatAndPrefix(format, withPrefix)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["etcd_yaml_int"] = 8888
	config1["etcd_yaml_foo"] = "modified etcd string"
	config1["etcd_yaml_shopping_list"].([]any)[0] = "modified etcd slice"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]any{
			"etcd_yaml_foo":           "bar",
			"etcd_yaml_year":          2022,
			"etcd_yaml_temperature":   37.5,
			"etcd_yaml_shopping_list": []any{"bread", "milk", "eggs"},
			"etcd_yaml_abc":           "xyz",
		},
		expectedConfig,
	)
}

// getEtcdExpectedConfigMapByFormatAndPrefix returns expected config maps
// (correlated with etcdResponseKeys variable).
func getEtcdExpectedConfigMapByFormatAndPrefix(format string, withPrefix bool) map[string]any {
	var expectedConfigMap map[string]any
	const subkeyVal = "xyz"
	switch format {
	case xconf.RemoteValueJSON:
		expectedConfigMap = map[string]any{
			"etcd_json_foo":           "bar",
			"etcd_json_year":          float64(2022),
			"etcd_json_temperature":   37.5,
			"etcd_json_shopping_list": []any{"bread", "milk", "eggs"},
		}
		if withPrefix {
			expectedConfigMap["etcd_json_abc"] = subkeyVal
		}
	case xconf.RemoteValueYAML:
		expectedConfigMap = map[string]any{
			"etcd_yaml_foo":           "bar",
			"etcd_yaml_year":          2022,
			"etcd_yaml_temperature":   37.5,
			"etcd_yaml_shopping_list": []any{"bread", "milk", "eggs"},
		}
		if withPrefix {
			expectedConfigMap["etcd_yaml_abc"] = subkeyVal
		}
	case xconf.RemoteValuePlain:
		expectedConfigMap = map[string]any{"etcd_plain_key": "1000"}
		if withPrefix {
			expectedConfigMap["etcd_plain_key/subkey"] = subkeyVal
		}
	}

	return expectedConfigMap
}

func benchmarkEtcdLoader(format string, withWatcher bool) func(b *testing.B) {
	return func(b *testing.B) {
		b.Helper()
		content := etcdResponseKeys[format][true]
		key := etcdKeys[format]
		kvSvr := etcdKVServer{
			rangeCallback: func(_ context.Context, _ *pb.RangeRequest) (*pb.RangeResponse, error) {
				return &pb.RangeResponse{
					Kvs:   content,
					More:  false,
					Count: int64(len(content)),
				}, nil
			},
		}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatal(err)
		}
		svr := grpc.NewServer()
		pb.RegisterKVServer(svr, &kvSvr)
		go func(svr *grpc.Server, l net.Listener) {
			_ = svr.Serve(l)
		}(svr, ln)
		defer svr.Stop()

		opts := []xconf.EtcdLoaderOption{
			xconf.EtcdLoaderWithEndpoints([]string{ln.Addr().String()}),
			xconf.EtcdLoaderWithPrefix(),
			xconf.EtcdLoaderWithValueFormat(format),
		}
		if withWatcher {
			opts = append(opts, xconf.EtcdLoaderWithWatcher())
		}
		subject := xconf.NewEtcdLoader(key, opts...)
		defer subject.Close()

		b.ReportAllocs()
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			_, err := subject.Load()
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkEtcdLoader_json(b *testing.B) {
	benchmarkEtcdLoader(xconf.RemoteValueJSON, false)(b)
}

func BenchmarkEtcdLoader_yaml(b *testing.B) {
	benchmarkEtcdLoader(xconf.RemoteValueYAML, false)(b)
}

func BenchmarkEtcdLoader_plain(b *testing.B) {
	benchmarkEtcdLoader(xconf.RemoteValuePlain, false)(b)
}

func BenchmarkEtcdLoader_json_withWatcher(b *testing.B) {
	benchmarkEtcdLoader(xconf.RemoteValueJSON, true)(b)
}

func BenchmarkEtcdLoader_yaml_withWatcher(b *testing.B) {
	benchmarkEtcdLoader(xconf.RemoteValueYAML, true)(b)
}

func BenchmarkEtcdLoader_plain_withWatcher(b *testing.B) {
	benchmarkEtcdLoader(xconf.RemoteValuePlain, true)(b)
}

func ExampleEtcdLoader() {
	// load all keys starting with "APP_"
	hosts := []string{"127.0.0.1:2379"}
	loader := xconf.NewEtcdLoader(
		"APP_",
		xconf.EtcdLoaderWithEndpoints(hosts),
		xconf.EtcdLoaderWithPrefix(),
	)

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	for key, value := range configMap {
		fmt.Println(key+":", value)
	}
}
