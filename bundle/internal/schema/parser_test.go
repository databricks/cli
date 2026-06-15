package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLaunchStage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GA", ""},
		{"", ""},
		{"PUBLIC_PREVIEW", "PUBLIC_PREVIEW"},
		{"PUBLIC_BETA", "PUBLIC_BETA"},
		{"PRIVATE_PREVIEW", "PRIVATE_PREVIEW"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, normalizeLaunchStage(tc.input))
	}
}

func TestNotableEnumLaunchStages(t *testing.T) {
	t.Run("drops GA, keeps preview values", func(t *testing.T) {
		got := notableEnumLaunchStages(map[string]string{
			"STORAGE_OPTIMIZED": "PUBLIC_PREVIEW",
			"STANDARD":          "GA",
		})
		assert.Equal(t, map[string]string{"STORAGE_OPTIMIZED": "PUBLIC_PREVIEW"}, got)
	})

	t.Run("returns nil when every value is GA", func(t *testing.T) {
		assert.Nil(t, notableEnumLaunchStages(map[string]string{"STANDARD": "GA"}))
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		assert.Nil(t, notableEnumLaunchStages(nil))
	})
}

func TestNonEmptyEnumDescriptions(t *testing.T) {
	t.Run("keeps non-empty descriptions", func(t *testing.T) {
		got := nonEmptyEnumDescriptions(map[string]string{
			"STORAGE_OPTIMIZED": "Storage-optimized endpoint.",
			"STANDARD":          "Standard endpoint.",
		})
		assert.Equal(t, map[string]string{
			"STORAGE_OPTIMIZED": "Storage-optimized endpoint.",
			"STANDARD":          "Standard endpoint.",
		}, got)
	})

	t.Run("drops empty descriptions", func(t *testing.T) {
		got := nonEmptyEnumDescriptions(map[string]string{
			"STORAGE_OPTIMIZED": "Storage-optimized endpoint.",
			"STANDARD":          "",
		})
		assert.Equal(t, map[string]string{"STORAGE_OPTIMIZED": "Storage-optimized endpoint."}, got)
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		assert.Nil(t, nonEmptyEnumDescriptions(nil))
		assert.Nil(t, nonEmptyEnumDescriptions(map[string]string{"STANDARD": ""}))
	})
}
