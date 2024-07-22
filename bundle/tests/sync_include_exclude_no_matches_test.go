package config_tests

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/require"
)

func TestSyncIncludeExcludeNoMatchesTest(t *testing.T) {
	b := loadTarget(t, "./sync/override", "development")

	diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), validate.ValidateSyncPatterns())
	require.Len(t, diags, 3)
	require.NoError(t, diags.Error())

	require.Equal(t, diags[0].Severity, diag.Warning)
	require.Equal(t, diags[0].Summary, "Pattern dist does not match any files")
	require.Equal(t, diags[0].Location.File, filepath.Join("sync", "override", "databricks.yml"))
	require.Equal(t, diags[0].Location.Line, 17)
	require.Equal(t, diags[0].Location.Column, 11)
	require.Equal(t, diags[0].Paths[0].String(), "sync.exclude[0]")

	summaries := []string{
		fmt.Sprintf("Pattern %s does not match any files", filepath.Join("src", "*")),
		fmt.Sprintf("Pattern %s does not match any files", filepath.Join("tests", "*")),
	}

	require.Equal(t, diags[1].Severity, diag.Warning)
	require.Contains(t, summaries, diags[1].Summary)

	require.Equal(t, diags[2].Severity, diag.Warning)
	require.Contains(t, summaries, diags[2].Summary)
}
