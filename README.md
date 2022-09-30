# Xconf

[![Build Status](https://github.com/actforgood/xconf/actions/workflows/build.yml/badge.svg)](https://github.com/actforgood/xconf/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/license-MIT-blue)](https://raw.githubusercontent.com/actforgood/xconf/main/LICENSE)
[![Coverage Status](https://coveralls.io/repos/github/actforgood/xconf/badge.svg?branch=main)](https://coveralls.io/github/actforgood/xconf?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/actforgood/xconf.svg)](https://pkg.go.dev/github.com/actforgood/xconf)  

---

Package `xconf` provides a configuration registry for an application.  
Configurations can be extracted from a file / env / remote system.  
Supported formats are json, yaml, ini, (java) properties, plain.


### Installation

```shell
$ go get -u github.com/actforgood/xconf
```


### Configuration loaders
You can create your own configuration retriever implementing `Loader` interface.
Package provides these Loaders for you:  

- `EnvLoader` - loads environment variables.
- `DotEnvFileLoader` - loads configuration from a .env file.
- `JSONFileLoader` - loads json configuration from a file.
- `JSONReaderLoader` - loads json configuration from a `io.Reader`.
- `YAMLFileLoader` - loads yaml configuration from a file.
- `YAMLReaderLoader` - loads yaml configuration from a `io.Reader`.
- `IniFileLoader` -  loads ini configuration from a file.
- `PropertiesFileLoader` - loads java style properties configuration from a file.
- `PropertiesBytesLoader` - loads java style properties configuration from a bytes slice.
- `ConsulLoader` - loads json/yaml/plain configuration from a remote Consul KV Store.
- `EtcdLoader` - loads json/yaml/plain configuration from a remote Etcd KV Store.
- `PlainLoader` - explicit configuration provider.
- `FileLoader` -  factory for `<JSON|YAML|Ini|DotEnv|Properties>FileLoader`s based on file extension.
- `MultiLoader` - loads (and merges, if configured) configuration from multiple loaders.


Upon above loaders there are available decorators which can help you achieve more sophisticated outcome:  

- `FilterKVLoader` - filters other loader's configurations (based on keys and or their values).  
Example of applicability: I load configurations from environment, but I only want the ones prefixed with "MY_APP_" - I can apply this loader with `FilterKVWhitelistFunc(FilterKeyWithPrefix("MY_APP_")` filter function.
- `AlterValueLoader` - changes the value for a configuration key.  
Example of applicability: I load configurations from environment and for a given key I want its value to be a slice (not a string as envs are read/stored by default) - I can apply this loader with `ToStringList` altering function.
- `IgnoreErrorLoader` - ignores the error returned by another loader.  
Example of applicability: I load configuration from environment and from file (using a `MultiLoader`), but it's not mandatory for that file to exist (file it's just an auxiliary source for my configurations, that may exist) - I can use this loader to ignore "file does not exist" error.
- `FileCacheLoader` - caches configuration from a `[X]FileLoader` until file gets modified (to be used if loader is called multiple times).
- `FlattenLoader` - creates easy to access nested configuration leaf keys symlinks.
- `AliasLoader` - creates aliases for other keys.


### Configuration contract
The main configuration contract this package provides looks like:

```go
type Config interface {
	Get(key string, def ...interface{}) interface{}
}
```
with a default implementation obtained with:
```go
// NewDefaultConfig instantiates a new default config object.
// The first parameter is the loader used as a source of getting the key-value configuration map.
// The second parameter represents a list of optional functions to configure the object.
func NewDefaultConfig(loader Loader, opts ...DefaultConfigOption) (*DefaultConfig, error)
```

The `DefaultConfig` has an option of reloading configurations (interval based), if you want to retrieve updated configuration
at runtime.
There are 2 (proposed) ways of working with it:  

- injecting a `Config` reference and calling `Get(key)` every time you need a configuration.
- registering your class as an observer to get notified about config changes.

Example of usage (first case) (note: code does not compile):
```go
// cart_service.go
const (
	defaultMaxQtyCfgVal uint = 100
	maxQtyCfgKey             = "MAX_ALLOWED_QTY_TO_ORDER"
)

type CartService struct {
	config xconf.Config
}

func NewCartService(config xconf.Config) *CartService {
	return &CartService{
		config: config,
	}
}

func (cartSvc *CartService) AddProduct(sku string, qty uint) error {
	// ...
	if customerType != B2B {
		totalQty := currentQty + qty
		maxQty := cartSvc.config.Get(maxQtyCfgKey, defaultMaxQtyCfgVal).(uint)
		if totalQty > maxQty {
			return ErrMaxQtyExceeded
		}
	}
	// ...
	return nil
}

func main() {
	// somewhere in the bootstrap of your application ...
	var (
		loader  xconf.Loader // = ... your desired source(s)
		config  xconf.Config
		cartSvc *CartService
	)
	config, err := xconf.NewDefaultConfig(
		loader,
		xconf.DefaultConfigWithReloadInterval(time.Minute), // reload every minute
	)
	if err != nil {
		panic(err)
	}
	cartSvc = NewCartService(config)

	// somewhere in the application business flow ...
	_ = cartSvc.AddProduct("IPHONE", 1)

	// somewhere in the shutdown of your application ...
	if closableConfig, ok := config.(io.Closer); ok {
		_ = closableConfig.Close()
	}
}
```


Example of usage (second case) (note: code does not compile):
```go
// redis_wrapper.go
const (
	RedisHostCfgKey = "REDIS_HOST"
	DefaultRedisHostCfgVal = "127.0.0.1:6379"
)

type RedisClient interface {
	Ping() error
	Get(key string) (string, error)
	Set(key string, value interface{}, expiration time.Duration) (string, error)
	Close() error
}

type RedisClientWrapper struct {
	client *redis.Client // official client
	mu     sync.RWMutex
}

func NewRedisClientWrapper(host string) *RedisClientWrapper {
	officialClient = ...
	return &RedisClientWrapper {
		client: officialClient,
	}
}

func (wrapper *RedisClientWrapper) Get(key string) (string, error) {
	wrapper.mu.RLock()
	defer wrapper.mu.RUnlock()

	return wrapper.client.Get(key).Result()
}

func (wrapper *RedisClientWrapper) OnConfigChange(config xconf.Config, changedKeys ...string) {
	for _, changedKey := range changedKeys {
		if changedKey == RedisHostCfgKey { // or use strings.EqualFold() if you enabled DefaultConfigWithIgnoreCaseSensitivity.
			wrapper.mu.Lock()
			_ = wrapper.client.Close() // close previous client
			newClient := ... // reinitialize client based on config.Get(RedisHostCfgKey).(string)
			wrapper.client = newClient
			wrapper.mu.Unlock()
		}
	}
}

func main() {
	// somewhere in the bootstrap of your application ...
	var (
		loader      xconf.Loader // = ... your desired source(s)
		config      xconf.Config
		redisClient RedisClient
	)
	config, err := xconf.NewDefaultConfig(
		loader,
		xconf.DefaultConfigWithReloadInterval(30 * time.Second), // reload every 30 seconds
	)
	if err != nil {
		panic(err)
	}
	redisHost := config.Get(RedisHostCfgKey, DefaultRedisHostCfgVal).(string)
	redisClient = NewRedisClient(redisHost)
	config.RegisterObserver(redisClient.OnConfigChange) // register redis wrapper as an observer

	// somewhere in the application business flow ...
	_, _ = redisClient.Get("something")

	// somewhere in the shutdown of your application ...
	if closableConfig, ok := config.(io.Closer); ok {
		_ = closableConfig.Close()
	}
	_ = redisClient.Close()
}
```

### Unmarshal configuration map to structs
This is not the subject of this package, but as a mention, you can achieve that if needed, with a package like github.com/mitchellh/mapstructure.  
Example:
```go
package main

import (
	"bytes"
	"fmt"

	"github.com/actforgood/xconf"
	"github.com/mitchellh/mapstructure"
)

type DBConfig struct {
	Host string
	Port int
	Auth Auth
}

type Auth struct {
	Username string
	Password string
}

func main() {
	var (
		jsonConfig = `{
	"db": {
		"host": "127.0.0.1",
		"port": 3306,
		"auth": {
			"username": "JohnDoe",
			"password": "verySecretPwd"
		}
	}		
}`
		dbConfig    DBConfig               // the struct to populate with configuration
		dbConfigMap map[string]interface{} // the configuration map for "db" key
		loader      = xconf.JSONReaderLoader(bytes.NewReader([]byte(jsonConfig)))
	)

	// example using directly a Loader:
	configMap, err := loader.Load()
	if err != nil {
		panic(err)
	}
	dbConfigMap = configMap["db"].(map[string]interface{})
	if err := mapstructure.Decode(dbConfigMap, &dbConfig); err != nil {
		panic(err)
	}
	fmt.Printf("%+v", dbConfig)

	// example using the Config contract:
	config, err := xconf.NewDefaultConfig(loader)
	if err != nil {
		panic(err)
	}
	dbConfigMap = config.Get("db").(map[string]interface{})
	if err := mapstructure.Decode(dbConfigMap, &dbConfig); err != nil {
		panic(err)
	}
	fmt.Printf("%+v", dbConfig)

	// both Printf will produce: {Host:127.0.0.1 Port:3306 Auth:{Username:JohnDoe Password:verySecretPwd}}
}
```

### TODOs
Things that can be added to package, extended:  

- Support more formats (like HCL)  
- Add also a writer/persister functionality (currently you can only read configurations) to different sources and formats (JSONFileWriter/YAMLFileWriter/EtcdWriter/ConsulWriter/...) implementing a common contract like:
```go
type ConfigWriter interface {
	Write(configMap map[string]interface{}) error
}
```
- Add a typed struct with methods like `GetString`, `GetInt`...

### Misc 
Feel free to use this pkg if you like it and fits your needs.  
Check also other packages like spf13/viper ...


### License
This package is released under a MIT license. See [LICENSE](LICENSE).  
Other 3rd party packages directly used by this package are released under their own licenses.  

* github.com/joho/godotenv - [MIT License](https://github.com/joho/godotenv/blob/main/LICENCE)  
* github.com/magiconair/properties - [BSD (2 Clause) License](https://github.com/magiconair/properties/blob/main/LICENSE.md)  
* gopkg.in/ini.v1 - [Apache 2.0 License](https://github.com/go-ini/ini/blob/main/LICENSE)  
* gopkg.in/yaml.v3 - [MIT And Apache License](https://github.com/go-yaml/yaml/blob/v3.0.1/LICENSE)  
* go.etcd.io/etcd/client/v3 - [Apache 2.0 License](https://github.com/etcd-io/etcd/blob/main/LICENSE)  
* github.com/spf13/cast - [MIT License](https://github.com/spf13/cast/blob/master/LICENSE)  
* github.com/actforgood/xerr - [MIT License](https://github.com/actforgood/xerr/blob/main/LICENSE)  
* github.com/actforgood/xlog - [MIT License](https://github.com/actforgood/xlog/blob/main/LICENSE)  
