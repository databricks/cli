package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/stretchr/testify/require"
)

func TestSuggestTargetIfWrongPassed(t *testing.T) {
	b := load(t, "target_overrides/workspace")

	diags := bundletest.ApplyLoading(t, b, mutator.SelectTarget("incorrect"))
	err := diags.Error()
	require.Error(t, err)
	require.Contains(t, err.Error(), "Available targets:")
	require.Contains(t, err.Error(), "development")
	require.Contains(t, err.Error(), "staging")
}
