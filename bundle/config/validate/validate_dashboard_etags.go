package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

func ValidateDashboardEtags() bundle.ReadOnlyMutator {
	return &validateDashboardEtags{}
}

type validateDashboardEtags struct{ bundle.RO }

func (v *validateDashboardEtags) Name() string {
	return "validate:validate_dashboard_etags"
}

func (v *validateDashboardEtags) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// No dashboards should have etags set. They are purely internal state.
	for k, dashboard := range b.Config.Resources.Dashboards {
		if dashboard.Etag != "" {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("dashboard %q has an etag set. Etags must not be set in bundle configuration", dashboard.DisplayName),
					Paths:    structpath.NewPathSlice("resources", "dashboards", k),
				},
			}
		}
	}
	return nil
}
