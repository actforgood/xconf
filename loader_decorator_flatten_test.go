// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/actforgood/xconf"
)

func TestFlattenLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - flat keys from map[string]interface{}", testFlattenLoaderWithFlatKeysFromNestedStringMap)
	t.Run("success - flat keys from map[interface{}]interface{}", testFlattenLoaderWithFlatKeysFromNestedInterfaceMap)
	t.Run("success - with loader options", testFlattenLoaderWithOptions)
	t.Run("error - original, decorated loader", testFlattenLoaderReturnsErrFromDecoratedLoader)
	t.Run("success - safe-mutable config map", testFlattenLoaderReturnsSafeMutableConfigMap)
}

func testFlattenLoaderReturnsErrFromDecoratedLoader(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = errors.New("intentionally triggered decorated loader error")
		loader      = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			return nil, expectedErr
		})
		subject = xconf.NewFlattenLoader(loader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertTrue(t, errors.Is(err, expectedErr))
	assertNil(t, config)
}

func testFlattenLoaderWithFlatKeysFromNestedStringMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.JSONReaderLoader(bytes.NewReader([]byte(`{
			"db": {
			  "mysql": {
				"host": "192.168.10.10",
				"port": 3306
			  },
			  "postgresql": {
				"host": "192.168.10.11",
				"port": 5432
			  },
			  "adapter": "mysql"
			},
			"foo": "bar",
			"a": {
			  "b": {
				"c": {
				  "d": "e"
				}
			  }
			}
		}`)))
		subject = xconf.NewFlattenLoader(loader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"foo": "bar",
			"db": map[string]interface{}{
				"mysql": map[string]interface{}{
					"host": "192.168.10.10",
					"port": float64(3306),
				},
				"postgresql": map[string]interface{}{
					"host": "192.168.10.11",
					"port": float64(5432),
				},
				"adapter": "mysql",
			},
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": map[string]interface{}{
						"d": "e",
					},
				},
			},
			"a.b.c.d":            "e",
			"db.adapter":         "mysql",
			"db.mysql.host":      "192.168.10.10",
			"db.mysql.port":      float64(3306),
			"db.postgresql.host": "192.168.10.11",
			"db.postgresql.port": float64(5432),
		},
		config,
	)
}

func testFlattenLoaderWithFlatKeysFromNestedInterfaceMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.YAMLReaderLoader(bytes.NewReader([]byte(`
db:
  mysql:
    host: 192.168.10.10
    port: 3306
  postgresql:
    host: 192.168.10.11
    port: 5432
  adapter: mysql
foo: bar
1:
  2:
    3:
      4: 5
`)))

		subject = xconf.NewFlattenLoader(loader)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"foo": "bar",
			"db": map[string]interface{}{
				"mysql": map[string]interface{}{
					"host": "192.168.10.10",
					"port": 3306,
				},
				"postgresql": map[string]interface{}{
					"host": "192.168.10.11",
					"port": 5432,
				},
				"adapter": "mysql",
			},
			"1": map[interface{}]interface{}{
				2: map[interface{}]interface{}{
					3: map[interface{}]interface{}{
						4: 5,
					},
				},
			},
			"1.2.3.4":            5,
			"db.adapter":         "mysql",
			"db.mysql.host":      "192.168.10.10",
			"db.mysql.port":      3306,
			"db.postgresql.host": "192.168.10.11",
			"db.postgresql.port": 5432,
		},
		config,
	)
}

func testFlattenLoaderWithOptions(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"foo": "bar",
			"db": map[string]interface{}{
				"mysql": map[string]interface{}{
					"host": "192.168.10.10",
					"port": 3306,
				},
				"postgresql": map[interface{}]interface{}{
					"host": "192.168.10.11",
					"port": 5432,
				},
				"adapter": "mysql",
			},
			"a": map[interface{}]interface{}{
				"b": map[string]interface{}{
					"c": map[interface{}]interface{}{
						"d": "e",
					},
				},
			},
		})
		subject = xconf.NewFlattenLoader(
			loader,
			xconf.FlattenLoaderWithSeparator("^"),
			xconf.FlattenLoaderWithFlatKeysOnly(),
		)
	)

	// act
	config, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(
		t,
		map[string]interface{}{
			"foo":                "bar",
			"a^b^c^d":            "e",
			"db^adapter":         "mysql",
			"db^mysql^host":      "192.168.10.10",
			"db^mysql^port":      3306,
			"db^postgresql^host": "192.168.10.11",
			"db^postgresql^port": 5432,
		},
		config,
	)
}

func testFlattenLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]interface{}{
			"foo": "bar",
			"db": map[string]interface{}{
				"mysql": map[string]interface{}{
					"host": "192.168.10.10",
					"port": 3306,
				},
				"postgresql": map[string]interface{}{
					"host": "192.168.10.11",
					"port": 5432,
				},
				"adapter": "mysql",
			},
			"a": map[interface{}]interface{}{
				"b": map[interface{}]interface{}{
					"c": map[interface{}]interface{}{
						"d": "e",
					},
				},
			},
		})
		subject        = xconf.NewFlattenLoader(loader)
		expectedConfig = map[string]interface{}{
			"foo": "bar",
			"db": map[string]interface{}{
				"mysql": map[string]interface{}{
					"host": "192.168.10.10",
					"port": 3306,
				},
				"postgresql": map[string]interface{}{
					"host": "192.168.10.11",
					"port": 5432,
				},
				"adapter": "mysql",
			},
			"a": map[interface{}]interface{}{
				"b": map[interface{}]interface{}{
					"c": map[interface{}]interface{}{
						"d": "e",
					},
				},
			},
			"a.b.c.d":            "e",
			"db.adapter":         "mysql",
			"db.mysql.host":      "192.168.10.10",
			"db.mysql.port":      3306,
			"db.postgresql.host": "192.168.10.11",
			"db.postgresql.port": 5432,
		}
	)

	// act
	config1, err1 := subject.Load()

	// assert
	assertNil(t, err1)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["foo"] = "fooooooo"
	config1["db"].(map[string]interface{})["mysql"].(map[string]interface{})["port"] = 3307
	config1["a.b.c.d"] = "EEE"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]interface{}{
			"foo": "bar",
			"db": map[string]interface{}{
				"mysql": map[string]interface{}{
					"host": "192.168.10.10",
					"port": 3306,
				},
				"postgresql": map[string]interface{}{
					"host": "192.168.10.11",
					"port": 5432,
				},
				"adapter": "mysql",
			},
			"a": map[interface{}]interface{}{
				"b": map[interface{}]interface{}{
					"c": map[interface{}]interface{}{
						"d": "e",
					},
				},
			},
			"a.b.c.d":            "e",
			"db.adapter":         "mysql",
			"db.mysql.host":      "192.168.10.10",
			"db.mysql.port":      3306,
			"db.postgresql.host": "192.168.10.11",
			"db.postgresql.port": 5432,
		},
		expectedConfig,
	)
}

func BenchmarkFlattenLoader(b *testing.B) {
	origLoader := xconf.PlainLoader(map[string]interface{}{
		"foo": "bar",
		"db": map[string]interface{}{
			"mysql": map[string]interface{}{
				"host": "127.0.0.1",
				"port": 3306,
			},
			"adapter": "mysql",
		},
	})
	subject := xconf.NewFlattenLoader(origLoader)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := subject.Load()
		if err != nil {
			b.Error(err)
		}
	}
}

func ExampleFlattenLoader() {
	jsonConfig := []byte(`{
	"db": {
		"mysql": {
			"host": "192.168.10.10",
			"port": 3306
		},
		"adapter": "mysql"
	},
	"foo": "bar"
}`)
	origLoader := xconf.JSONReaderLoader(bytes.NewReader(jsonConfig))
	loader := xconf.NewFlattenLoader(origLoader)

	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}

	fmt.Println(configMap["foo"])
	fmt.Println(configMap["db"].(map[string]interface{})["mysql"].(map[string]interface{})["host"])
	fmt.Println(configMap["db.mysql.host"]) // much easier way to access information compared to previous statement.

	// Output:
	// bar
	// 192.168.10.10
	// 192.168.10.10
}
