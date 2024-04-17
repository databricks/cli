package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/require"
)

func TestSyncIncludeExcludeNoMatchesTest(t *testing.T) {
	b := loadTarget(t, "./override_sync", "development")

	diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), validate.ValidateSyncPatterns())
	require.Len(t, diags, 3)
	require.NoError(t, diags.Error())
	require.Equal(t, diags[0].Severity, diag.Warning)
	require.Equal(t, diags[0].Summary, "Exclude pattern dist does not match any files")
	require.Equal(t, diags[0].Location.File, "override_sync/databricks.yml")
	require.Equal(t, diags[0].Location.Line, 17)
	require.Equal(t, diags[0].Location.Column, 11)
	require.Equal(t, diags[0].Path.String(), "sync.exclude[0]")

	require.Equal(t, diags[1].Severity, diag.Warning)
	require.Equal(t, diags[1].Summary, "Include pattern src/* does not match any files")
	require.Equal(t, diags[1].Location.File, "override_sync/databricks.yml")
	require.Equal(t, diags[1].Location.Line, 9)
	require.Equal(t, diags[1].Location.Column, 7)
	require.Equal(t, diags[1].Path.String(), "sync.include[0]")

	require.Equal(t, diags[2].Severity, diag.Warning)
	require.Equal(t, diags[2].Summary, "Include pattern tests/* does not match any files")
	require.Equal(t, diags[2].Location.File, "override_sync/databricks.yml")
	require.Equal(t, diags[2].Location.Line, 15)
	require.Equal(t, diags[2].Location.Column, 11)
	require.Equal(t, diags[2].Path.String(), "sync.include[1]")
}
