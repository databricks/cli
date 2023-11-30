package handler

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/stretchr/testify/assert"
)

func TestFriendlyHandler(t *testing.T) {
	var out bytes.Buffer

	handler := NewFriendlyHandler(&out, &Options{
		Color: true,
		Level: log.LevelTrace,
	})

	logger := slog.New(handler)

	{
		// One line per level.
		for _, level := range []slog.Level{
			log.LevelTrace,
			log.LevelDebug,
			log.LevelInfo,
			log.LevelWarn,
			log.LevelError,
		} {
			out.Reset()
			logger.Log(context.Background(), level, "simple message")
			t.Log(out.String())
		}
	}

	{
		// Single key/value pair.
		out.Reset()
		logger.Info("simple message", "key", "value")
		t.Log(out.String())
	}

	{
		// Multiple key/value pairs.
		out.Reset()
		logger.Info("simple message", "key1", "value", "key2", "value")
		t.Log(out.String())
	}

	{
		// Multiple key/value pairs with duplicate keys.
		out.Reset()
		logger.Info("simple message", "key", "value", "key", "value")
		t.Log(out.String())
	}

	{
		// Log message with time.
		out.Reset()
		logger.Info("simple message", "time", time.Now())
		t.Log(out.String())
	}

	{
		// Log message with grouped key/value pairs.
		out.Reset()
		logger.Info("simple message", slog.Group("group", slog.String("key", "value")))
		t.Log(out.String())
	}

	{
		// Add key/value pairs to logger.
		out.Reset()
		logger.With("logger_key", "value").Info("simple message")
		t.Log(out.String())
	}

	{
		// Add group to logger.
		out.Reset()
		logger.WithGroup("logger_group").Info("simple message", "key", "value")
		t.Log(out.String())
	}

	{
		// Add group and key/value pairs to logger.
		out.Reset()
		logger.WithGroup("logger_group").With("logger_key", "value").Info("simple message")
		t.Log(out.String())
	}
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

	{
		// ReplaceAttr replaces attributes.
		out.Reset()
		logger.Info("simple message", "key", "value")
		assert.Contains(t, out.String(), `replaced="value"`)
	}
}
