package config_tests

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncIncludeExcludeNoMatchesTest(t *testing.T) {
	b := loadTarget(t, "./sync/override", "development")

	diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), validate.ValidateSyncPatterns())
	require.Len(t, diags, 3)
	require.NoError(t, diags.Error())

	require.Equal(t, diags[0].Severity, diag.Warning)
	require.Equal(t, diags[0].Summary, "Pattern dist does not match any files")

	require.Len(t, diags[0].Paths, 1)
	require.Equal(t, diags[0].Paths[0].String(), "sync.exclude[0]")

	assert.Len(t, diags[0].Locations, 1)
	require.Equal(t, diags[0].Locations[0].File, filepath.Join("sync", "override", "databricks.yml"))
	require.Equal(t, diags[0].Locations[0].Line, 17)
	require.Equal(t, diags[0].Locations[0].Column, 11)

	summaries := []string{
		fmt.Sprintf("Pattern %s does not match any files", filepath.Join("src", "*")),
		fmt.Sprintf("Pattern %s does not match any files", filepath.Join("tests", "*")),
	}

	require.Equal(t, diags[1].Severity, diag.Warning)
	require.Contains(t, summaries, diags[1].Summary)

	require.Equal(t, diags[2].Severity, diag.Warning)
	require.Contains(t, summaries, diags[2].Summary)
}

func TestSyncIncludeWithNegate(t *testing.T) {
	b := loadTarget(t, "./sync/negate", "default")

	diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), validate.ValidateSyncPatterns())
	require.Len(t, diags, 0)
	require.NoError(t, diags.Error())
}

func TestSyncIncludeWithNegateNoMatches(t *testing.T) {
	b := loadTarget(t, "./sync/negate", "dev")

	diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), validate.ValidateSyncPatterns())
	require.Len(t, diags, 1)
	require.NoError(t, diags.Error())

	require.Equal(t, diags[0].Severity, diag.Warning)
	require.Equal(t, diags[0].Summary, "Pattern !*.txt2 does not match any files")
}
