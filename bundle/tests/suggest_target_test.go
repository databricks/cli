package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/require"
)

func TestSuggestTargetIfWrongPassed(t *testing.T) {
	b := load(t, "target_overrides/workspace")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.SelectTarget("incorrect"))
	err := diags.Error()
	require.Error(t, err)
	require.Contains(t, err.Error(), "Available targets:")
	require.Contains(t, err.Error(), "development")
	require.Contains(t, err.Error(), "staging")
}
