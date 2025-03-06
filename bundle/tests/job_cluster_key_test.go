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

	diags := bundle.Apply(context.Background(), b, validate.JobClusterKeyDefined())
	require.Len(t, diags, 1)
	require.NoError(t, diags.Error())
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "job_cluster_key key is not defined", diags[0].Summary)
}

func TestJobClusterKeyDefinedTest(t *testing.T) {
	b := loadTarget(t, "./job_cluster_key", "development")

	diags := bundle.Apply(context.Background(), b, validate.JobClusterKeyDefined())
	require.Empty(t, diags)
}
