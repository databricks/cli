package handler

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/log"
)

func TestFriendlyHandler(t *testing.T) {
	var out bytes.Buffer

	handler := NewFriendlyHandler(&out, &Options{
		Color: true,
		Level: log.LevelTrace,
	})

	logger := slog.New(handler)

	// Helper function to run a test case and print the output.
	run := func(fn func()) {
		out.Reset()
		fn()
		t.Log(strings.TrimSpace(out.String()))
	}

	// One line per level.
	for _, level := range []slog.Level{
		log.LevelTrace,
		log.LevelDebug,
		log.LevelInfo,
		log.LevelWarn,
		log.LevelError,
	} {
		run(func() {
			logger.Log(context.Background(), level, "simple message")
		})
	}

	// Single key/value pair.
	run(func() {
		logger.Info("simple message", "key", "value")
	})

	// Multiple key/value pairs.
	run(func() {
		logger.Info("simple message", "key1", "value", "key2", "value")
	})

	// Multiple key/value pairs with duplicate keys.
	run(func() {
		logger.Info("simple message", "key", "value", "key", "value")
	})

	// Log message with time.
	run(func() {
		logger.Info("simple message", "time", time.Now())
	})

	// Log message with grouped key/value pairs.
	run(func() {
		logger.Info("simple message", slog.Group("group", slog.String("key", "value")))
	})

	// Add key/value pairs to logger.
	run(func() {
		logger.With("logger_key", "value").Info("simple message")
	})

	// Add group to logger.
	run(func() {
		logger.WithGroup("logger_group").Info("simple message", "key", "value")
	})

	// Add group and key/value pairs to logger.
	run(func() {
		logger.WithGroup("logger_group").With("logger_key", "value").Info("simple message")
	})
}

func TestFriendlyHandlerReplaceAttr(t *testing.T) {
	var out bytes.Buffer

	handler := NewFriendlyHandler(&out, &Options{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "key" {
				a.Key = "replaced"
			}
			return a
		},
	})

	logger := slog.New(handler)

	// Helper function to run a test case and print the output.
	run := func(fn func()) {
		out.Reset()
		fn()
		t.Log(strings.TrimSpace(out.String()))
	}

	// ReplaceAttr replaces attributes.
	run(func() {
		logger.Info("simple message", "key", "value")
	})
}
