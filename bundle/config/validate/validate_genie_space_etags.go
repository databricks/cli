package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

func ValidateGenieSpaceEtags() bundle.ReadOnlyMutator {
	return &validateGenieSpaceEtags{}
}

type validateGenieSpaceEtags struct{ bundle.RO }

func (v *validateGenieSpaceEtags) Name() string {
	return "validate:validate_genie_space_etags"
}

func (v *validateGenieSpaceEtags) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// No genie spaces should have etags set. They are purely internal state
	// (persisted by the direct engine for drift detection), never authored by
	// the user. Mirrors ValidateDashboardEtags.
	for k, genieSpace := range b.Config.Resources.GenieSpaces {
		if genieSpace.Etag != "" {
			return diag.Diagnostics{
				{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("genie space %q has an etag set. Etags must not be set in bundle configuration", genieSpace.Title),
					Paths:     []dyn.Path{dyn.MustPathFromString("resources.genie_spaces." + k)},
					Locations: b.Config.GetLocations("resources.genie_spaces." + k),
				},
			}
		}
	}
	return nil
}
