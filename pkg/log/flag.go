/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2020 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package log

import (
	"fmt"
	"sync"
	"time"

	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
)

const (
	// SamplingFreqFlagName flag name
	SamplingFreqFlagName = "log-sampling-frequency"
	// LevelFlagName flag name
	LevelFlagName = "log-level"
	// FormatFlagName flag name
	FormatFlagName = "log-format"
	// WithColorFlagName flag name
	WithColorFlagName = "log-enable-color"
	// IgnoreCallerFlagName caller flag
	IgnoreCallerFlagName = "log-ignore-caller"
	// OutputPathsName path name
	OutputPathsName = "log-output-paths"
)

var (
	lock            = &sync.RWMutex{}
	logSamplingFreq = pflag.Duration(SamplingFreqFlagName, 100*time.Millisecond, "Log sampling `INTERVAL`")
	logLevel        = pflag.String(LevelFlagName, "info", "Minimum log output `LEVEL`")
	logFormat       = pflag.String(FormatFlagName, "plain", "Log output `FORMAT`")
	logWithColor    = pflag.Bool(WithColorFlagName, false, "Whether to output colored log")
	logIgnoreCaller = pflag.Bool(IgnoreCallerFlagName, false, "Ignore the output of caller information in the log")
	logOutputPaths  = pflag.StringSlice(OutputPathsName, []string{}, "Log output paths, comma separated.")
)

// AddFlags registers this package's flags on arbitrary FlagSets, such that they
// point to the same value as the global flags.
func AddFlags(fs *pflag.FlagSet) {
	fs.AddFlag(pflag.Lookup(LevelFlagName))
	fs.AddFlag(pflag.Lookup(FormatFlagName))
	fs.AddFlag(pflag.Lookup(WithColorFlagName))
	fs.AddFlag(pflag.Lookup(IgnoreCallerFlagName))
	fs.AddFlag(pflag.Lookup(SamplingFreqFlagName))
	fs.AddFlag(pflag.Lookup(OutputPathsName))
}

// SetLevel to change the log level flag
func SetLevel(level string) error {
	lock.Lock()
	defer lock.Unlock()
	if _, err := parseLevel(); err != nil {
		return err
	}
	logLevel = &level
	return nil
}

// Level returns the current log level
func Level() string {
	lock.RLock()
	defer lock.RUnlock()
	return *logLevel
}

// SetFormat to change the log format flag
func SetFormat(format string) error {
	lock.Lock()
	defer lock.Unlock()
	if _, err := parseFormat(); err != nil {
		return err
	}
	logFormat = &format
	return nil
}

// Format return the current log format
func Format() string {
	lock.RLock()
	defer lock.RUnlock()
	return *logFormat
}

func mustLevel() zapcore.Level {
	level := MustParseLevel()
	zapLevel := zapcore.InfoLevel
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		panic(err)
	}
	return zapLevel
}

func parseFormat() (string, error) {
	switch *logFormat {
	case "json", "JSON":
		return "json", nil
	case "console", "plain":
		return "console", nil
	default:
		return "", fmt.Errorf("unable to parse the special format")
	}
}

// MustParseFormat to parse the log format flag
func MustParseFormat() string {
	if format, err := parseFormat(); err == nil {
		return format
	}
	return "console"
}

func parseLevel() (string, error) {
	switch *logLevel {
	case "DEBUG", "debug", "dbg", "DBG":
		return "DEBUG", nil
	case "WARN", "warn", "warning", "WARNING":
		return "WARN", nil
	case "ERROR", "error", "ERR", "err":
		return "ERROR", nil
	case "FATAL", "fatal":
		return "FATAL", nil
	case "PANIC", "panic":
		return "PANIC", nil
	default:
		return "", fmt.Errorf("unable to parse the special level")
	}
}

// MustParseLevel to parse the log level flag
func MustParseLevel() string {
	if level, err := parseLevel(); err == nil {
		return level
	}
	return "INFO"
}
