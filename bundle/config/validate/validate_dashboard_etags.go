package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("dashboard %q has an etag set. Etags must not be set in bundle configuration", dashboard.DisplayName),
					Paths:     []dyn.Path{dyn.MustPathFromString("resources.dashboards." + k)},
					Locations: b.Config.GetLocations("resources.dashboards." + k),
				},
			}
		}
	}
	return nil
}
