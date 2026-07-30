// Package logging provides structured logging utilities for subtitle-manager.
// It wraps logrus and supports per-component log levels, file output, and configuration.
//
// This package is used throughout subtitle-manager for consistent, configurable logging.

package logging

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/jdfalk/subtitle-manager/pkg/security"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	mu      sync.Mutex
	loggers           = map[string]*logrus.Logger{}
	output  io.Writer = os.Stdout
)

// GetLogger returns a logger for the given component. It reads the log level
// from the configuration key "log_levels.<component>". If not set, the global
// "log-level" is used.
// SetHook installs h as the global log hook and discards the logger cache so
// existing components pick it up.
//
// Setting the Hook variable directly is not enough, and the failure is silent.
// GetLogger memoises one logger per component and calls AddHook when it first
// builds that logger, so a hook installed afterwards is attached to nothing —
// any component whose logger already exists keeps logging past it. Tests that
// installed a memory hook to capture output therefore passed in isolation and
// failed once a sibling test had already caused the same component's logger to
// be created.
func SetHook(h *MemoryHook) {
	mu.Lock()
	defer mu.Unlock()
	Hook = h
	clear(loggers)
}

func GetLogger(component string) *logrus.Entry {
	mu.Lock()
	defer mu.Unlock()
	if l, ok := loggers[component]; ok {
		return l.WithField("component", component)
	}
	l := logrus.New()
	l.SetOutput(output)
	levelStr := viper.GetString("log_levels." + component)
	if levelStr == "" {
		levelStr = viper.GetString("log-level")
	}
	lvl, err := logrus.ParseLevel(levelStr)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	l.SetLevel(lvl)
	l.AddHook(Hook)
	loggers[component] = l
	return l.WithField("component", component)
}

// Configure sets the log output destination for all loggers.
// It creates the log directory if necessary and writes to both
// standard output and the configured log file.
func Configure() {
	path := viper.GetString("log_file")
	if path == "" {
		output = os.Stdout
		logrus.SetOutput(output)
		return
	}
	sanitizedPath, err := security.ValidateAndSanitizePath(path)
	if err != nil {
		output = os.Stdout
		logrus.SetOutput(output)
		return
	}
	if err := os.MkdirAll(filepath.Dir(sanitizedPath), 0o755); err != nil {
		output = os.Stdout
		logrus.SetOutput(output)
		return
	}
	f, err := os.OpenFile(sanitizedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		output = os.Stdout
		logrus.SetOutput(output)
		return
	}
	output = io.MultiWriter(os.Stdout, f)
	logrus.SetOutput(output)
}
