// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"github.com/actforgood/xlog"
)

// LogLevelProvider provides a level read from a Config object.
// It can be used to configure log level for a xlog.Logger.
// If the level configuration key is not found, the default provided level is returned.
// If the reload option is present on the config object, you may change
// during application run the underlying key without restarting the app,
// and new configured value will be used in place, if suitable.
func LogLevelProvider(
	config Config,
	lvlKey string,
	defaultLvl string,
	levelLabels map[xlog.Level]string,
) xlog.LevelProvider {
	labeledLevels := flipLevelLabels(levelLabels)

	return func() xlog.Level {
		lvl := config.Get(lvlKey, defaultLvl).(string)

		return labeledLevels[lvl]
	}
}

// flipLevelLabels flips level labels map.
func flipLevelLabels(levelLabels map[xlog.Level]string) map[string]xlog.Level {
	flippedLevelLabels := make(map[string]xlog.Level, len(levelLabels))
	for lvl, label := range levelLabels {
		flippedLevelLabels[label] = lvl
	}

	return flippedLevelLabels
}

// LogErrorHandler is a handler which can be used in a xconf.DefaultConfig
// object as a reload error handler. It logs the error with a xlog.Logger.
// Passed parameter is a function that returns the logger (Logger and Config depend
// one of each other, this way we can instantiate them separately...)
func LogErrorHandler(loggerGetter func() xlog.Logger) func(error) {
	return func(err error) {
		loggerGetter().Error(
			xlog.MessageKey, "[xconf] could not reload configuration",
			xlog.ErrorKey, xlog.StackErr(err),
		)
	}
}
