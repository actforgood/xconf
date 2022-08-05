// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
)

// Config provides prototype for returning configurations.
type Config interface {
	// Get returns a configuration value for a given key.
	// The first parameter is the key to return the value for.
	// The second parameter is optional, and represents a default
	// value in case key is not found. It also has a role in inferring
	// the type of key's value (if it exists) and thus key's value
	// will be casted to default's value type.
	Get(key string, def ...interface{}) interface{}
}

// DefaultConfig is the default implementation for the Config contract.
// It is based on a Loader to retrieve configuration from.
// Is implements io.Closer interface and thus Close should be called at
// your application shutdown in order to avoid memory leaks.
type DefaultConfig struct {
	*defaultConfig // so we can use finalizer
}

type defaultConfig struct {
	// loader to retrieve configuration from.
	loader Loader
	// configMap the loaded key-value configuration map.
	configMap map[string]interface{}
	// observers contain the list of registered observers for changed keys.
	observers []ConfigObserver
	// refreshInterval represents the interval to reload the configMap.
	// If it is <=0, reload will be disabled.
	reloadInterval time.Duration
	// reloadErrorHandler is an optional handler for errors occurred during reloading configuration.
	// You can log the error, for example.
	reloadErrorHandler func(error)
	// ticker is used to reload the configMap at reloadInterval.
	ticker *time.Ticker
	// ignoreCaseSensitivity is a flag indicating whether keys' case sensitivity should be ignored.
	ignoreCaseSensitivity bool
	// mu is a concurrency semaphore for accessing the configMap.
	mu *sync.RWMutex
	// wg is a wait group used to notify main thread that reload goroutine stopped.
	wg *sync.WaitGroup
	// closed is a channel to notify reload goroutine to stop.
	closed chan struct{}
}

// NewDefaultConfig instantiates a new default config object.
// The first parameter is the loader used as a source of getting the key-value configuration map.
// The second parameter represents a list of optional functions to configure the object.
func NewDefaultConfig(loader Loader, opts ...DefaultConfigOption) (*DefaultConfig, error) {
	config := &DefaultConfig{&defaultConfig{
		loader: loader,
		mu:     new(sync.RWMutex),
	}}

	// apply options, if any.
	for _, opt := range opts {
		opt(config)
	}

	err := config.setConfigMap()
	if err != nil {
		return nil, err
	}

	if config.reloadInterval > 0 {
		config.ticker = time.NewTicker(config.reloadInterval)
		config.wg = new(sync.WaitGroup)
		config.closed = make(chan struct{}, 1)
		config.wg.Add(1)
		go config.reloadAsync()
		// register also a finalizer, just in case, user forgets to call Close().
		// Note: user should do not rely on this, it's recommended to explicitly call Close().
		runtime.SetFinalizer(config, (*DefaultConfig).Close)
	}

	return config, nil
}

// Get returns a configuration value for a given key.
// The first parameter is the key to return the value for.
// The second parameter is optional, and represents a default
// value in case key is not found. It so has a role in inferring
// the type of key's value (if it exists) and thus key's value
// will be casted to default's value type.
// Only basic types (string, bool, int, uint, float, and their flavours),
// time.Duration, time.Time, []int, []string are covered.
// If a cast error occurs, the defaultValue is returned.
func (cfg *defaultConfig) Get(key string, def ...interface{}) interface{} {
	if cfg.ignoreCaseSensitivity {
		key = strings.ToUpper(key)
	}

	if cfg.reloadInterval > 0 {
		// micro-optimization; in case reload is disabled, we don't have
		// to protect with a mutex. See benchmarks.
		cfg.mu.RLock()
	}
	value, foundKey := cfg.configMap[key]
	if cfg.reloadInterval > 0 {
		cfg.mu.RUnlock()
	}

	if len(def) > 0 {
		defaultValue := def[0]
		if !foundKey {
			return defaultValue
		}
		if defaultValue != nil {
			return castValueByDefault(value, defaultValue)
		}
	}

	return value
}

// RegisterObserver adds a new observer that will get notified of keys changes.
func (cfg *defaultConfig) RegisterObserver(observer ConfigObserver) {
	cfg.mu.Lock()
	if cfg.observers == nil {
		cfg.observers = []ConfigObserver{observer}
	} else {
		cfg.observers = append(cfg.observers, observer)
	}
	cfg.mu.Unlock()
}

// setConfigMap loads the config map.
func (cfg *defaultConfig) setConfigMap() error {
	newConfigMap, err := cfg.loader.Load()
	if err != nil {
		return err
	}
	if cfg.ignoreCaseSensitivity {
		toUppercaseConfigMap(newConfigMap)
	}

	cfg.mu.Lock()
	oldConfigMap := cfg.configMap
	cfg.configMap = newConfigMap
	cfg.mu.Unlock()

	cfg.notifyObservers(oldConfigMap, newConfigMap)

	return nil
}

// notifyObservers computes changed (updated/deleted/new) keys on a config reload,
// and notifies registered observers about them, if there are any changed keys and observers.
func (cfg *defaultConfig) notifyObservers(oldConfigMap, newConfigMap map[string]interface{}) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	if cfg.observers == nil || reflect.DeepEqual(oldConfigMap, newConfigMap) {
		return
	}

	// max will be reached only if all old config map keys get deleted,
	// highly improbable
	maxChangedKeysCap := len(oldConfigMap) + len(newConfigMap)
	changedKeys := make([]string, 0, maxChangedKeysCap)
	for oldKey := range oldConfigMap { // compute updated/deleted keys
		if !reflect.DeepEqual(oldConfigMap[oldKey], newConfigMap[oldKey]) {
			changedKeys = append(changedKeys, oldKey)
		}
	}
	for newKey := range newConfigMap { // compute new keys
		if _, found := oldConfigMap[newKey]; !found {
			changedKeys = append(changedKeys, newKey)
		}
	}

	for _, notifyObserver := range cfg.observers {
		notifyObserver(cfg, changedKeys...)
	}
}

// reloadAsync reloads the config map asynchronous, interval based.
// Calling Close() will stop this goroutine.
func (cfg *defaultConfig) reloadAsync() {
	defer cfg.wg.Done()

	for {
		select {
		case <-cfg.closed:
			cfg.ticker.Stop()

			return
		case <-cfg.ticker.C:
			if err := cfg.setConfigMap(); err != nil && cfg.reloadErrorHandler != nil {
				cfg.reloadErrorHandler(err)
			}
		}
	}
}

// close stops the underlying ticker used to reload config, avoiding memory leaks.
func (cfg *defaultConfig) close() {
	if cfg != nil {
		close(cfg.closed)
		cfg.wg.Wait()
	}
}

// Close stops the underlying ticker used to reload config, avoiding memory leaks.
// It should be called at your application shutdown.
// It implements io.Closer interface, and the returned error can be disregarded (is nil all the time).
func (cfg *DefaultConfig) Close() error {
	if cfg != nil && cfg.reloadInterval > 0 {
		cfg.close()
		runtime.SetFinalizer(cfg, nil)
	}

	return nil
}

// castValueByDefault casts a key's value to provided default value's type.
// Only basic types (string, bool, int, uint, float, and their flavours),
// time.Duration, time.Time, []int, []string are covered.
// If a cast error occurs, the defaultValue is returned.
func castValueByDefault(value, defaultValue interface{}) interface{} {
	var (
		castValue interface{}
		castErr   error
	)
	switch defaultValue.(type) {
	case string:
		castValue, castErr = cast.ToStringE(value)
	case int:
		castValue, castErr = cast.ToIntE(value)
	case uint:
		castValue, castErr = cast.ToUintE(value)
	case float64:
		castValue, castErr = cast.ToFloat64E(value)
	case bool:
		castValue, castErr = cast.ToBoolE(value)
	case time.Duration:
		castValue, castErr = cast.ToDurationE(value)
	case int64:
		castValue, castErr = cast.ToInt64E(value)
	case int32:
		castValue, castErr = cast.ToInt32E(value)
	case int16:
		castValue, castErr = cast.ToInt16E(value)
	case int8:
		castValue, castErr = cast.ToInt8E(value)
	case uint64:
		castValue, castErr = cast.ToUint64E(value)
	case uint32:
		castValue, castErr = cast.ToUint32E(value)
	case uint16:
		castValue, castErr = cast.ToUint16E(value)
	case uint8:
		castValue, castErr = cast.ToUint8E(value)
	case float32:
		castValue, castErr = cast.ToFloat32E(value)
	case time.Time:
		castValue, castErr = cast.ToTimeE(value)
	case []string:
		castValue, castErr = cast.ToStringSliceE(value)
	case []int:
		castValue, castErr = cast.ToIntSliceE(value)
	default:
		castValue = value // not supported cast type, return directly the value
	}

	if castErr == nil {
		return castValue
	}

	return defaultValue
}

// toUppercaseConfigMap transforms all (first level) keys to uppercase.
func toUppercaseConfigMap(configMap map[string]interface{}) {
	for key, value := range configMap {
		delete(configMap, key)
		// Note: here if a duplicate key exists, it will get overwritten.
		configMap[strings.ToUpper(key)] = value
	}
}

// DefaultConfigOption defines optional function for configuring
// a DefaultConfig object.
type DefaultConfigOption func(*DefaultConfig)

// DefaultConfigWithReloadInterval sets interval to reload configuration.
// Passing a value <= 0 disables the config reload.
//
// By default, configuration reload is disabled.
//
// Usage example:
//
//	// enable config reload at an interval of 5 minutes:
//	cfg, err := xconf.NewDefaultConfig(loader, xconf.DefaultConfigWithReloadInterval(5 * time.Minute))
func DefaultConfigWithReloadInterval(reloadInterval time.Duration) DefaultConfigOption {
	return func(config *DefaultConfig) {
		config.reloadInterval = reloadInterval
	}
}

// DefaultConfigWithIgnoreCaseSensitivity disables case sensitivity for keys.
//
// For example, if the configuration map contains a key "Foo", calling Get() with "foo" / "FOO" / etc.
// will return Foo's value.
//
// Usage example:
//
//	cfg, err := xconf.NewDefaultConfig(loader, xconf.DefaultConfigWithIgnoreCaseSensitivity())
//	if err != nil {
//		panic(err)
//	}
//	value1 := cfg.Get("foo")
//	value2 := cfg.Get("FOO")
//	value3 := cfg.Get("foO")
//	// all values are equal
func DefaultConfigWithIgnoreCaseSensitivity() DefaultConfigOption {
	return func(config *DefaultConfig) {
		config.ignoreCaseSensitivity = true
	}
}

// DefaultConfigWithReloadErrorHandler sets the handler for errors that may occur
// during reloading configuration, if DefaultConfigWithReloadInterval was applied.
// If reload fails, "old"/previous configuration is active.
//
// You can choose to log the error, for example.
//
// By default, error is simply ignored.
func DefaultConfigWithReloadErrorHandler(errHandler func(error)) DefaultConfigOption {
	return func(config *DefaultConfig) {
		config.reloadErrorHandler = errHandler
	}
}

// ConfigObserver gets called to notify about changed keys on Config reload.
type ConfigObserver func(cfg Config, changedKeys ...string)
