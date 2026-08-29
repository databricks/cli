package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type validateLifecycleStarted struct {
	engine engine.EngineType
}

// ValidateLifecycleStarted returns a mutator that errors when lifecycle.started
// is used with the terraform deployment engine.
// lifecycle.started is only supported in direct deployment mode.
func ValidateLifecycleStarted(e engine.EngineType) bundle.Mutator {
	return &validateLifecycleStarted{engine: e}
}

func (m *validateLifecycleStarted) Name() string {
	return "ValidateLifecycleStarted"
}

func (m *validateLifecycleStarted) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics
	for _, group := range b.Config.Resources.AllResources() {
		for key, resource := range group.Resources {
			lws, ok := resource.GetLifecycle().(resources.LifecycleWithStarted)
			if !ok || lws.Started == nil {
				continue
			}
			path := "resources." + group.Description.PluralName + "." + key + ".lifecycle.started"
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "lifecycle.started is only supported in direct deployment mode",
				Locations: b.Config.GetLocations(path),
			})
		}
	}
	return diags
}
