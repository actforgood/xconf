// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/actforgood/xconf"
)

func TestFileCacheLoader(t *testing.T) {
	t.Parallel()

	t.Run("success - config is loaded from cache", testFileCacheLoaderSuccess)
	t.Run("error - fstat", testFileCacheLoaderReturnsFstatError)
	t.Run("error - original, decorated loader", testFileCacheLoaderReturnsErrFromDecoratedLoader)
	t.Run("success - safe-mutable config map", testFileCacheLoaderReturnsSafeMutableConfigMap)
}

func testFileCacheLoaderSuccess(t *testing.T) {
	t.Parallel()

	// arrange
	// setup a file for which we will play with its modification time.
	filePath, err := setUpTmpFile("xconf-filecacheloader-*.json", `{"foo":"bar"}`+"\n")
	if err != nil {
		t.Fatal("prerequisite failed:", err)
	}
	defer tearDownTmpFile(filePath)

	fileLoaderCallsCnt := 0
	fileLoader := xconf.LoaderFunc(func() (map[string]interface{}, error) {
		fileLoaderCallsCnt++
		if fileLoaderCallsCnt == 1 {
			return map[string]interface{}{"foo": "bar"}, nil
		}

		return map[string]interface{}{"foo": "baz", "year": 2022}, nil
	})
	subject := xconf.NewFileCacheLoader(fileLoader, filePath)

	// act & assert - first time content should be loaded from file loader
	config, err := subject.Load()
	requireNil(t, err)
	assertEqual(t, 1, fileLoaderCallsCnt)
	assertEqual(t, map[string]interface{}{"foo": "bar"}, config)

	// act & assert - second time result should be taken from cache
	config, err = subject.Load()
	requireNil(t, err)
	assertEqual(t, 1, fileLoaderCallsCnt) // still 1
	assertEqual(t, map[string]interface{}{"foo": "bar"}, config)

	// modify the file
	time.Sleep(time.Second)
	if err := writeToFile(filePath, `{"foo":"baz","year":2022}`+"\n"); err != nil {
		t.Error(err)
		t.FailNow()
	}

	// act & assert - third time result should be reloaded
	config, err = subject.Load()
	requireNil(t, err)
	assertEqual(t, 2, fileLoaderCallsCnt)
	assertEqual(t, map[string]interface{}{"foo": "baz", "year": 2022}, config)
}

func testFileCacheLoaderReturnsFstatError(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = os.ErrNotExist
		fileLoader  = xconf.PlainLoader(map[string]interface{}{
			"foo": "bar",
		})
		subject = xconf.NewFileCacheLoader(
			fileLoader,
			"/this/path/does/not/exist/1234.json",
		)
	)

	// act
	config, err := subject.Load()

	// assert
	assertTrue(t, errors.Is(err, expectedErr))
	assertNil(t, config)
}

func testFileCacheLoaderReturnsErrFromDecoratedLoader(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		expectedErr = errors.New("intentionally triggered decorated loader error")
		fileLoader  = xconf.LoaderFunc(func() (map[string]interface{}, error) {
			return nil, expectedErr
		})
		subject = xconf.NewFileCacheLoader(fileLoader, jsonFilePath)
	)

	// act
	config, err := subject.Load()

	// assert
	assertTrue(t, errors.Is(err, expectedErr))
	assertNil(t, config)
}

func testFileCacheLoaderReturnsSafeMutableConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		fileLoader = xconf.PlainLoader(map[string]interface{}{
			"filecache_string": "some string",
			"filecache_slice":  []interface{}{"foo", "bar", "baz"},
			"filecache_map":    map[string]interface{}{"foo": "bar"},
		})
		subject        = xconf.NewFileCacheLoader(fileLoader, jsonFilePath)
		expectedConfig = map[string]interface{}{
			"filecache_string": "some string",
			"filecache_slice":  []interface{}{"foo", "bar", "baz"},
			"filecache_map":    map[string]interface{}{"foo": "bar"},
		}
	)

	// act
	config1, err := subject.Load()

	// assert
	assertNil(t, err)
	assertEqual(t, expectedConfig, config1)

	// modify first returned value, expect second returned value to be initial one.
	config1["filecache_int"] = 4444
	config1["filecache_string"] = "test filecache string"
	config1["filecache_slice"].([]interface{})[0] = "test filecache slice"
	config1["filecache_map"].(map[string]interface{})["foo"] = "test filecache map"

	// act
	config2, err2 := subject.Load()

	// assert
	assertNil(t, err2)
	assertEqual(t, expectedConfig, config2)

	assertEqual(
		t,
		map[string]interface{}{
			"filecache_string": "some string",
			"filecache_slice":  []interface{}{"foo", "bar", "baz"},
			"filecache_map":    map[string]interface{}{"foo": "bar"},
		},
		expectedConfig,
	)
}

func TestFileCacheLoader_concurrency(t *testing.T) {
	t.Parallel()

	// arrange
	// setup a file for which we will play with its modification time.
	filePath, err := setUpTmpFile("xconf-filecacheloader-concurrency-*.json", `{"foo":"bar"}`+"\n")
	if err != nil {
		t.Fatal("prerequisite failed:", err)
	}
	defer tearDownTmpFile(filePath)

	// have a goroutine that constantly modifies the file
	stopModify, stoppedModify := make(chan struct{}, 1), make(chan struct{}, 1)
	var modifiedCnt uint32
	go func(fPath string, stop <-chan struct{}, stopped chan<- struct{}) {
		for {
			select {
			case <-stop: // stop this goroutine
				close(stopped)

				return
			default:
				content := `{"foo":"bar_` + strconv.FormatInt(time.Now().UnixNano(), 10) + `"}` + "\n"
				_ = writeToFile(fPath, content)

				// make a pause, let Load() also take from cache.
				time.Sleep(time.Millisecond)
				atomic.AddUint32(&modifiedCnt, 1)
			}
		}
	}(filePath, stopModify, stoppedModify)

	subject := xconf.NewFileCacheLoader(xconf.JSONFileLoader(filePath), filePath)
	goroutinesNo := 500
	var wg sync.WaitGroup

	// act & assert
	for i := 0; i < goroutinesNo; i++ {
		wg.Add(1)
		go func(loader xconf.Loader, waitGr *sync.WaitGroup) {
			defer waitGr.Done()

			// trigger load while another goroutine may modify the underlying file
			for i := 0; i < 50; i++ {
				config, err := loader.Load()
				if assertNil(t, err) {
					assertEqual(t, 1, len(config))
				}
			}
		}(subject, &wg)
	}

	wg.Wait()         // wait for Loading goroutines to finish
	close(stopModify) // trigger file modification goroutine to stop
	<-stoppedModify   // wait for file modification goroutine to stop
	// print some stats
	t.Logf(
		"%d goroutines loaded for 50 times each a file that was modified for %d times",
		goroutinesNo,
		atomic.LoadUint32(&modifiedCnt),
	)
}

// setUpTmpFile creates a file in the tmp directory.
func setUpTmpFile(filePattern, content string) (string, error) {
	f, err := os.CreateTemp("", filePattern)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err = f.WriteString(content); err != nil {
		return "", err
	}

	return f.Name(), nil
}

// tearDownTmpFile deletes the file specified.
func tearDownTmpFile(filePath string) {
	_ = os.Remove(filePath)
}

// writeToFile writes given content to the specified file.
func writeToFile(filePath, content string) error {
	f, err := os.OpenFile(filePath, os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)

	return err
}

func benchmarkFileCacheLoader(loader xconf.Loader) func(b *testing.B) {
	return func(b *testing.B) {
		b.Helper()
		b.ReportAllocs()
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			_, err := loader.Load()
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkFileCacheLoader_withJSON(b *testing.B) {
	subject := xconf.NewFileCacheLoader(
		xconf.JSONFileLoader(jsonFilePath),
		jsonFilePath,
	)

	benchmarkFileCacheLoader(subject)(b)
}

func BenchmarkFileCacheLoader_withYAML(b *testing.B) {
	subject := xconf.NewFileCacheLoader(
		xconf.YAMLFileLoader(yamlFilePath),
		yamlFilePath,
	)

	benchmarkFileCacheLoader(subject)(b)
}

func BenchmarkFileCacheLoader_withIni(b *testing.B) {
	subject := xconf.NewFileCacheLoader(
		xconf.NewIniFileLoader(iniFilePath),
		iniFilePath,
	)

	benchmarkFileCacheLoader(subject)(b)
}

func BenchmarkFileCacheLoader_withProperties(b *testing.B) {
	subject := xconf.NewFileCacheLoader(
		xconf.PropertiesFileLoader(propertiesFilePath),
		propertiesFilePath,
	)

	benchmarkFileCacheLoader(subject)(b)
}

func ExampleFileCacheLoader() {
	var (
		filePath = "testdata/config.json"
		loader   = xconf.NewFileCacheLoader(
			xconf.JSONFileLoader(filePath),
			filePath,
		)
		configMap map[string]interface{}
		err       error
	)

	for i := 0; i < 3; i++ {
		// 1st time original loader will be called,
		// 2nd and 3rd time, config will be retrieved from cache.
		configMap, err = loader.Load()
		if err != nil {
			panic(err)
		}
	}

	for key, value := range configMap {
		fmt.Println(key+":", value)
	}

	// Unordered output:
	// json_foo: bar
	// json_year: 2022
	// json_temperature: 37.5
	// json_shopping_list: [bread milk eggs]
}
