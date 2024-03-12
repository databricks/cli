package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
)

func TestPathTranslationFallback(t *testing.T) {
	b := loadTarget(t, "./path_translation/fallback", "development")

	m := mutator.TranslatePaths()
	err := bundle.Apply(context.Background(), b, m)
	assert.NoError(t, err)

	// TODO: assert output
}

func TestPathTranslationFallbackError(t *testing.T) {
	// TODO: add target with a bad path to trigger the error message
}

func TestPathTranslationNative(t *testing.T) {
	b := loadTarget(t, "./path_translation/native", "development")

	m := mutator.TranslatePaths()
	err := bundle.Apply(context.Background(), b, m)
	assert.NoError(t, err)

	// TODO: assert output
}

func TestPathTranslationNativeError(t *testing.T) {
	// TODO: add target with a bad path to trigger the error message
}
