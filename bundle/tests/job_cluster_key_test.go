package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/require"
)

func TestJobClusterKeyNotDefinedTest(t *testing.T) {
	b := loadTarget(t, "./job_cluster_key", "default")

	diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), validate.JobClusterKeyDefined())
	require.Len(t, diags, 1)
	require.NoError(t, diags.Error())
	require.Equal(t, diags[0].Severity, diag.Warning)
	require.Equal(t, diags[0].Summary, "job_cluster_key key is not defined")
}

func TestJobClusterKeyDefinedTest(t *testing.T) {
	b := loadTarget(t, "./job_cluster_key", "development")

	diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), validate.JobClusterKeyDefined())
	require.Len(t, diags, 0)
}
