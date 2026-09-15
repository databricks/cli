package mutator

import (
	"context"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type validateRunAsGroup struct {
	engine engine.EngineType
}

// ValidateRunAsGroup rejects group identities unsupported by the Terraform provider.
// See https://github.com/databricks/terraform-provider-databricks/blob/v1.131.0/jobs/resource_job.go#L629.
func ValidateRunAsGroup(e engine.EngineType) bundle.Mutator {
	return &validateRunAsGroup{engine: e}
}

func (m *validateRunAsGroup) Name() string {
	return "ValidateRunAsGroup"
}

func (m *validateRunAsGroup) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics
	for _, key := range slices.Sorted(maps.Keys(b.Config.Resources.Jobs)) {
		runAs := b.Config.Resources.Jobs[key].RunAs
		if runAs == nil || runAs.GroupName == "" {
			continue
		}
		path := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key(key), dyn.Key("run_as"), dyn.Key("group_name"))
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "run_as.group_name is only supported in direct deployment mode",
			Detail: "Set 'bundle.engine: direct' in your databricks.yml or set DATABRICKS_BUNDLE_ENGINE=direct. " +
				"For an existing Terraform deployment, run 'databricks bundle migrate' first. " +
				"Alternatively, set run_as.user_name or run_as.service_principal_name on this job",
			Locations: b.Config.GetLocations(path.String()),
			Paths:     []dyn.Path{path},
		})
	}
	return diags
}
