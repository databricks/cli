package config_tests

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncIncludeExcludeNoMatchesTest(t *testing.T) {
	b := loadTarget(t, "./sync/override", "development")

	diags := bundle.Apply(context.Background(), b, validate.ValidateSyncPatterns())
	require.Len(t, diags, 3)
	require.NoError(t, diags.Error())

	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "Pattern dist does not match any files", diags[0].Summary)

	require.Len(t, diags[0].Paths, 1)
	require.Equal(t, "sync.exclude[0]", diags[0].Paths[0].String())

	assert.Len(t, diags[0].Locations, 1)
	require.Equal(t, diags[0].Locations[0].File, path.Join("sync", "override", "databricks.yml"))
	require.Equal(t, 17, diags[0].Locations[0].Line)
	require.Equal(t, 11, diags[0].Locations[0].Column)

	summaries := []string{
		fmt.Sprintf("Pattern %s does not match any files", path.Join("src", "*")),
		fmt.Sprintf("Pattern %s does not match any files", path.Join("tests", "*")),
	}

	require.Equal(t, diag.Warning, diags[1].Severity)
	require.Contains(t, summaries, diags[1].Summary)

	require.Equal(t, diag.Warning, diags[2].Severity)
	require.Contains(t, summaries, diags[2].Summary)
}

func TestSyncIncludeWithNegate(t *testing.T) {
	b := loadTarget(t, "./sync/negate", "default")

	diags := bundle.Apply(context.Background(), b, validate.ValidateSyncPatterns())
	require.Empty(t, diags)
	require.NoError(t, diags.Error())
}

func TestSyncIncludeWithNegateNoMatches(t *testing.T) {
	b := loadTarget(t, "./sync/negate", "dev")

	diags := bundle.Apply(context.Background(), b, validate.ValidateSyncPatterns())
	require.Len(t, diags, 1)
	require.NoError(t, diags.Error())

	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "Pattern !*.txt2 does not match any files", diags[0].Summary)
}
