// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

import (
	"os"
	"sync"
	"time"
)

// FileCacheLoader decorates another "file" loader to load configuration only
// if the file was modified. If the file was not modified since the previous load,
// the file won't be read and parsed again. You can improve performance this way,
// if you plan to load configuration multiple times (like using it in DefaultConfig with reload enabled).
type FileCacheLoader struct {
	loader   Loader     // a "file" loader, like JSONFileLoader, YAMLFileLoader, etc...
	filePath string     // file's path.
	cache    *fileCache // cache storage.
}

// NewFileCacheLoader instantiates a new FileCacheLoader object that loads
// that caches the configuration from the original "file" loader.
// The second parameter should be the same file as the original loader's one.
func NewFileCacheLoader(loader Loader, filePath string) FileCacheLoader {
	return FileCacheLoader{
		loader:   loader,
		filePath: filePath,
		cache:    new(fileCache),
	}
}

// Load returns decorated loader's key-value configuration map.
// If the file was modified since last load, that file will be read and parsed again,
// if not, the previous, already processed, configuration map will be returned.
func (decorator FileCacheLoader) Load() (map[string]interface{}, error) {
	fInfo, err := os.Stat(decorator.filePath)
	if err != nil {
		return nil, err
	}
	fModifiedAt := fInfo.ModTime()

	if configMap := decorator.cache.load(fModifiedAt); configMap != nil {
		return configMap, nil
	}

	configMap, err := decorator.loader.Load()
	if err != nil {
		return configMap, err
	}

	decorator.cache.save(configMap, fModifiedAt)

	return configMap, nil
}

// fileCache holds caching info.
type fileCache struct {
	configMap    map[string]interface{} // cached config map.
	lastModified time.Time              // file's last modified time.
	mu           sync.RWMutex           // concurrency semaphore
}

// save stores configuration key-value map and file's last modified time.
func (cache *fileCache) save(configMap map[string]interface{}, lastModified time.Time) {
	cache.mu.Lock()
	cache.configMap = DeepCopyConfigMap(configMap)
	cache.lastModified = lastModified
	cache.mu.Unlock()
}

// load retrieves configuration key-value map comparing file's modified time.
func (cache *fileCache) load(currentLastModified time.Time) map[string]interface{} {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if !currentLastModified.After(cache.lastModified) {
		// return a copy not to modify this state from outside (for example from a decorator,
		// which usually modifies directly the original returned configuration map reference
		// - for performance reasons, so we ensure from this stateful loader that we return a
		// new configuration map each time)
		return DeepCopyConfigMap(cache.configMap)
	}

	return nil
}
