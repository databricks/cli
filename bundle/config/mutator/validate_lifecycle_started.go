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

// ValidateLifecycleStarted returns a mutator that validates lifecycle.started
// is only used on supported resource types (apps, clusters, sql_warehouses).
// lifecycle.started is only supported in direct deployment mode.
func ValidateLifecycleStarted(e engine.EngineType) bundle.Mutator {
	return &validateLifecycleStarted{engine: e}
}

func (m *validateLifecycleStarted) Name() string {
	return "ValidateLifecycleStarted"
}

func (m *validateLifecycleStarted) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, group := range b.Config.Resources.AllResources() {
		for _, resource := range group.Resources {
			lws, ok := resource.GetLifecycle().(resources.LifecycleWithStarted)
			if !ok || lws.Started == nil {
				continue
			}

			// lifecycle.started is a direct-mode-only feature.
			if !m.engine.IsDirect() {
				diags = diags.Append(diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "lifecycle.started is only supported in direct deployment mode",
				})
			}
		}
	}

	return diags
}
