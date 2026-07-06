package main

import (
	"testing"

	"github.com/databricks/cli/internal/clijson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLaunchStage(t *testing.T) {
	tests := []struct {
		input string
		want  clijson.LaunchStage
	}{
		{"GA", ""},
		{"", ""},
		{"PUBLIC_PREVIEW", clijson.LaunchStagePublicPreview},
		{"PUBLIC_BETA", clijson.LaunchStagePublicBeta},
		{"PRIVATE_PREVIEW", clijson.LaunchStagePrivatePreview},
	}
	for _, tc := range tests {
		got, err := normalizeLaunchStage(tc.input)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}

func TestNormalizeLaunchStageUnknown(t *testing.T) {
	_, err := normalizeLaunchStage("SOMETHING_ELSE")
	assert.Error(t, err)
}

func TestNotableEnumLaunchStages(t *testing.T) {
	t.Run("drops GA, keeps preview values", func(t *testing.T) {
		got, err := notableEnumLaunchStages(map[string]string{
			"STORAGE_OPTIMIZED": "PUBLIC_PREVIEW",
			"STANDARD":          "GA",
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]clijson.LaunchStage{"STORAGE_OPTIMIZED": clijson.LaunchStagePublicPreview}, got)
	})

	t.Run("returns nil when every value is GA", func(t *testing.T) {
		got, err := notableEnumLaunchStages(map[string]string{"STANDARD": "GA"})
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		got, err := notableEnumLaunchStages(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("errors on unknown stage", func(t *testing.T) {
		_, err := notableEnumLaunchStages(map[string]string{"X": "SOMETHING_ELSE"})
		assert.Error(t, err)
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
