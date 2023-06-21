// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/actforgood/xconf"
	"github.com/actforgood/xlog"
)

//nolint:lll
func ExampleLogLevelProvider() {
	const logLevelKey = "APP_LOG_LEVEL"
	const defaultLogLevel = "WARN"

	// initialize Config object,
	// loader can be any other Loader, used this for the sake of simplicity and readability.
	loader := xconf.PlainLoader(map[string]any{
		logLevelKey: "INFO",
	})
	config, _ := xconf.NewDefaultConfig( // treat the error on live code!
		loader,
		xconf.DefaultConfigWithReloadInterval(time.Second),
	)
	defer config.Close()

	// initialize the logger with min level taken from Config.
	opts := xlog.NewCommonOpts()
	opts.MinLevel = xconf.LogLevelProvider(config, logLevelKey, defaultLogLevel, opts.LevelLabels)
	opts.Time = func() any { // mock time for output check
		return "2022-06-21T17:17:20Z"
	}
	opts.Source = xlog.SourceProvider(4, 1) // keep only filename for output check
	logger := xlog.NewSyncLogger(
		os.Stdout,
		xlog.SyncLoggerWithOptions(opts),
	)
	defer logger.Close()

	logger.Info(xlog.MessageKey, "log level is taken from xconf.Config")
	logger.Debug(xlog.MessageKey, "this message should not end up being logged as min level is INFO")

	// Output:
	// {"date":"2022-06-21T17:17:20Z","lvl":"INFO","msg":"log level is taken from xconf.Config","src":"/xlog_adapter_test.go:49"}
}

func ExampleLogErrorHandler() {
	// initialize the logger.
	logger := xlog.NewSyncLogger(os.Stdout)
	defer logger.Close()

	// initialize Config object,
	// loader can be any other Loader, used this for the sake of simplicity and readability.
	loader := xconf.PlainLoader(map[string]any{
		"foo": "bar",
	})
	loggerGetter := func() xlog.Logger { return logger }
	config, _ := xconf.NewDefaultConfig( // treat the error on live code!
		loader,
		xconf.DefaultConfigWithReloadInterval(time.Second),
		xconf.DefaultConfigWithReloadErrorHandler(xconf.LogErrorHandler(loggerGetter)),
	)
	defer config.Close()

	foo := config.Get("foo", "default foo").(string)
	fmt.Println(foo)

	// Output:
	// bar
}

func TestLogLevelProvider(t *testing.T) {
	t.Parallel()

	t.Run("level key config is found", testLogLevelProviderWithExistingKey)
	t.Run("level key config is not found - default", testLogLevelProviderWithDefaultLevel)
}

func testLogLevelProviderWithExistingKey(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]any{
			"APP_LOG_LEVEL": "DEBUG",
			"foo":           "bar",
		})
		config, _      = xconf.NewDefaultConfig(loader)
		loggerCommOpts = xlog.NewCommonOpts()
		subject        = xconf.LogLevelProvider(
			config,
			"APP_LOG_LEVEL",
			"INFO",
			loggerCommOpts.LevelLabels,
		)
		expectedResult = xlog.LevelDebug
	)

	for i := 0; i < 10; i++ {
		// act
		result := subject()

		// assert
		assertEqual(t, expectedResult, result)
	}
}

func testLogLevelProviderWithDefaultLevel(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		loader = xconf.PlainLoader(map[string]any{
			"foo": "bar",
		})
		config, _      = xconf.NewDefaultConfig(loader)
		loggerCommOpts = xlog.NewCommonOpts()
		subject        = xconf.LogLevelProvider(
			config,
			"APP_LOG_LEVEL",
			"INFO",
			loggerCommOpts.LevelLabels,
		)
		expectedResult = xlog.LevelInfo
	)

	for i := 0; i < 10; i++ {
		// act
		result := subject()

		// assert
		assertEqual(t, expectedResult, result)
	}
}

func TestLogErrorHandler(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		logger       = xlog.NewMockLogger()
		loggerGetter = func() xlog.Logger { return logger }
		subject      = xconf.LogErrorHandler(loggerGetter)
		err          = errors.New("reload test error")
	)
	defer logger.Close()
	logger.SetLogCallback(xlog.LevelError, func(keyValues ...any) {
		if assertEqual(t, 4, len(keyValues)) {
			assertEqual(t, xlog.MessageKey, keyValues[0])
			if msg, ok := keyValues[1].(string); assertTrue(t, ok) {
				assertTrue(t, strings.Contains(msg, "could not reload configuration"))
			}
			assertEqual(t, xlog.ErrorKey, keyValues[2])
			if errMsg, ok := keyValues[3].(string); assertTrue(t, ok) {
				assertTrue(t, strings.Contains(errMsg, err.Error()))
			}
		}
	})

	// act
	subject(err)

	// assert
	assertEqual(t, 1, logger.LogCallsCount(xlog.LevelError))
}
