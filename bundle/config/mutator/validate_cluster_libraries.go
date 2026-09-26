package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
)

type validateClusterLibraries struct {
	engine engine.EngineType
}

// ValidateClusterLibraries returns a mutator that errors when cluster libraries are used with
// the terraform deployment engine. Cluster libraries are only supported in direct deployment mode.
func ValidateClusterLibraries(e engine.EngineType) bundle.Mutator {
	return &validateClusterLibraries{engine: e}
}

func (m *validateClusterLibraries) Name() string {
	return "ValidateClusterLibraries"
}

func (m *validateClusterLibraries) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics
	for key, cluster := range b.Config.Resources.Clusters {
		if cluster == nil || len(cluster.Libraries) == 0 {
			continue
		}
		path := "resources.clusters." + key + ".libraries"
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "cluster libraries are only supported in direct deployment mode",
			Locations: b.Config.GetLocations(path),
		})
	}
	return diags
}
