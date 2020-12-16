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
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	once   sync.Once
)

// InitLogger initializes logger the way we want for tke.
func InitLogger() {
	once.Do(func() {
		logger = newLogger()
	})
}

// FlushLogger calls the underlying Core's Sync method, flushing any buffered
// log entries. Applications should take care to call Sync before exiting.
func FlushLogger() {
	if logger != nil {
		// #nosec
		// nolint: errcheck
		logger.Sync()
	}

}

// ZapLogger returns zap logger instance.
func ZapLogger() *zap.Logger {
	return getLogger()
}

// Reset to recreate the logger by changed flag params
func Reset() {
	lock.Lock()
	defer lock.Unlock()
	logger = newLogger()
}

// Check return if logging a message at the specified level is enabled.
func Check(level int32) bool {
	var lvl zapcore.Level
	if level < 5 {
		lvl = zapcore.InfoLevel
	} else {
		lvl = zapcore.DebugLevel
	}
	checkEntry := getLogger().Check(lvl, "")
	return checkEntry != nil
}

// StdErrLogger returns logger of standard library which writes to supplied zap
// logger at error level
func StdErrLogger() *log.Logger {
	if l, err := zap.NewStdLogAt(getLogger(), zapcore.ErrorLevel); err == nil {
		return l
	}
	return nil
}

// StdInfoLogger returns logger of standard library which writes to supplied zap
// logger at info level
func StdInfoLogger() *log.Logger {
	if l, err := zap.NewStdLogAt(getLogger(), zapcore.InfoLevel); err == nil {
		return l
	}
	return nil
}

// Debug method output debug level log.
func Debug(msg string, fields ...Field) {
	getLogger().Debug(msg, fields...)
}

// Info method output info level log.
func Info(msg string, fields ...Field) {
	getLogger().Info(msg, fields...)
}

// Warn method output warning level log.
func Warn(msg string, fields ...Field) {
	getLogger().Warn(msg, fields...)
}

// Error method output error level log.
func Error(msg string, fields ...Field) {
	getLogger().Error(msg, fields...)
}

// Panic method output panic level log and shutdown application.
func Panic(msg string, fields ...Field) {
	getLogger().Panic(msg, fields...)
}

// Fatal method output fatal level log.
func Fatal(msg string, fields ...Field) {
	getLogger().Fatal(msg, fields...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	Debug(fmt.Sprintf(template, args...))
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	Info(fmt.Sprintf(template, args...))
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	Warn(fmt.Sprintf(template, args...))
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	Error(fmt.Sprintf(template, args...))
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...interface{}) {
	Panic(fmt.Sprintf(template, args...))
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	Fatal(fmt.Sprintf(template, args...))
}

func getLogger() *zap.Logger {
	once.Do(func() {
		logger = newLogger()
	})
	return logger
}

func newLogger() *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     timeEncoder,
		EncodeDuration: milliSecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	// when output to local path, with color is forbidden
	if *logWithColor && ((logOutputPaths == nil) || (len(*logOutputPaths) == 0)) {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	loggerConfig := &zap.Config{
		Level:             zap.NewAtomicLevelAt(mustLevel()),
		Development:       false,
		DisableCaller:     *logIgnoreCaller,
		DisableStacktrace: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: int(*logSamplingFreq / time.Millisecond),
		},
		Encoding:         MustParseFormat(),
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	if logOutputPaths != nil {
		loggerConfig.OutputPaths = append(loggerConfig.OutputPaths, *logOutputPaths...)
	}

	//log rolling
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   strings.Join(*logOutputPaths, ""),
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     30, // days
	})
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout),
			w),
		zap.NewAtomicLevelAt(mustLevel()),
	)

	l := zap.New(core, zap.AddStacktrace(zapcore.PanicLevel),
		zap.AddCaller(), zap.Development(), zap.AddCallerSkip(2))

	/*l, err := loggerConfig.Build(zap.AddStacktrace(zapcore.PanicLevel),
		zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}*/
	initRestfulLogger(l)
	return l
}
