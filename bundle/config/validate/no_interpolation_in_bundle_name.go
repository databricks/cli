package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type noInterpolationInBundleName struct{}

func NoInterpolationInBundleName() bundle.Mutator {
	return &noInterpolationInBundleName{}
}

func (m *noInterpolationInBundleName) Name() string {
	return "validate:no_interpolation_in_bundle_name"
}

func (m *noInterpolationInBundleName) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if structvar.ContainsVariableReference(b.Config.Bundle.Name) {
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity: diag.Warning,
			Summary: `Please do not use variable interpolation in the name of your bundle. The name of your bundle
is a part of the path at which your bundle state is stored at by default. Parameterizing it at
runtime can have unexpected consequences like duplicate deployments or resources not being
cleaned up during bundle destroy.`,
			Paths: structpath.NewPathSlice("bundle", "name"),
		})
	}

	return nil
}
