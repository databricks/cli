package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/interpolation"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/stretchr/testify/require"
)

func TestBundleEnvironmentStringInterpolation(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./string_interpolation")
	require.NoError(t, err)

	init := interpolation.Interpolate(
		interpolation.IncludeLookupsInPath("bundle"),
		interpolation.IncludeLookupsInPath("workspace"),
		interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
	)
	err = init.Apply(ctx, b)
	require.NoError(t, err)

	require.Equal(t, b.Config.Resources.Jobs["test_job_a"].Name, "[development] Test A")
	require.Equal(t, b.Config.Resources.Jobs["test_job_b"].Name, "[development] Test B")
}
