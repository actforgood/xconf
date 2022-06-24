// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

import (
	"fmt"
	"strings"
	"sync"

	"github.com/actforgood/xerr"
)

// KeyConflictError is an error returned by MultiLoader
// in case of a duplicate key.
// If key overwrite is allowed, this error will not be returned.
type KeyConflictError struct {
	key string // the duplicate key
}

// NewKeyConflictError instantiates a new KeyConflictError.
// The duplicate key must be provided.
func NewKeyConflictError(key string) KeyConflictError {
	return KeyConflictError{key: key}
}

// Error returns string representation of the KeyConflictError.
// It implements standard go error interface.
func (e KeyConflictError) Error() string {
	return fmt.Sprintf(`key "%s" already exists`, e.key)
}

// MultiLoader is a composite loader that returns
// configurations from multiple loaders.
type MultiLoader struct {
	// loaders to load configuration from.
	loaders []Loader
	// allowKeyOverwrite is a flag that indicates whether a duplicate key
	// is allowed to be overwritten.
	allowKeyOverwrite bool
}

// NewMultiLoader instantiates a new MultiLoader object that loads
// and merges configuration from multiple loaders.
// The first parameter is a flag indicating whether a key is allowed to be overwritten,
// if found more than once.
// If not, a KeyConflictError will be returned.
// If yes, the order of loaders matters, meaning a later provided loader,
// will overwrite a previous provided loader's same found key.
// The rest of the parameters consist of the list of loaders configuration should be
// retrieved from.
func NewMultiLoader(allowKeyOverwrite bool, loaders ...Loader) MultiLoader {
	return MultiLoader{
		loaders:           loaders,
		allowKeyOverwrite: allowKeyOverwrite,
	}
}

// Load returns a merged configuration key-value map of all encapsulated loaders,
// or an error if something bad happens along the process.
func (loader MultiLoader) Load() (map[string]interface{}, error) {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		results   = make([]loadResult, len(loader.loaders))
		configMap map[string]interface{}
		unqKeys   = make(map[string]struct{})
		mErr      *xerr.MultiError
		startIdx  int
	)

	// load async each loader.
	for idx, loader := range loader.loaders {
		wg.Add(1)
		go loadAsync(loader, idx, &wg, &mu, results)
	}
	wg.Wait()

	// micro-optimization not to make extra allocation(s) (see benchmarks):
	// when allowKeyOverwrite is true we can append directly to first loader's config map
	// the rest of loaders' config maps.
	if loader.allowKeyOverwrite && results[0].err == nil {
		configMap = results[0].configMap
		startIdx = 1
	} else {
		configMap = make(map[string]interface{})
		startIdx = 0
	}

	// merge the results in the order loaders were provided.
	// Last loader will override previous loaders key in case of
	// a key conflict if allowKeyOverwrite option is set on MultiLoader.
	for idx := startIdx; idx < len(results); idx++ {
		loadResult := results[idx]
		if loadResult.err != nil {
			mErr = mErr.Add(loadResult.err)

			continue
		}
		for key, value := range loadResult.configMap {
			if !loader.allowKeyOverwrite {
				unqKey := strings.ToLower(key)
				if _, found := unqKeys[unqKey]; found {
					mErr = mErr.Add(NewKeyConflictError(key))

					continue
				}
				unqKeys[unqKey] = struct{}{}
			}

			configMap[key] = value
		}
	}

	if err := mErr.ErrOrNil(); err != nil {
		return nil, err
	}

	return configMap, nil
}

// loadResult encapsulates the result from a Loader.
type loadResult struct {
	configMap map[string]interface{} // configMap is the loaded key-value configuration.
	err       error                  // err is the error returned from Loader, if any.
}

// loadAsync calls a Loader asynchronous.
// Result is put in a results slice.
func loadAsync(
	loader Loader,
	idx int,
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	results []loadResult,
) {
	configMap, err := loader.Load()
	result := loadResult{
		configMap: configMap,
		err:       err,
	}
	mu.Lock()
	results[idx] = result
	mu.Unlock()
	wg.Done()
}
