package mutator

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/engine"
)

// lifecycleStartedReader is implemented by resource types that expose an
// optional `lifecycle.started` flag. UCM has no such resources today, but the
// interface keeps this mutator structurally identical to its DAB counterpart so
// future UC kinds (apps/cluster-shaped resources) can opt in by implementing it.
type lifecycleStartedReader interface {
	HasLifecycleStarted() bool
}

// lifecycleStartedScope describes one resource group to walk for lifecycle.started.
type lifecycleStartedScope struct {
	pluralName string
	getMap     func(*ucm.Ucm) map[string]any
}

// lifecycleStartedScopes enumerates the UC resource groups that may surface a
// `lifecycle.started` marker. None do today; the slice intentionally stays
// empty so adding a kind is a one-line opt-in.
var lifecycleStartedScopes = []lifecycleStartedScope{}

type validateLifecycleStarted struct {
	engine engine.EngineType
}

// ValidateLifecycleStarted returns a mutator that errors when lifecycle.started
// is used with the terraform deployment engine. lifecycle.started is only
// supported in direct deployment mode.
func ValidateLifecycleStarted(e engine.EngineType) ucm.Mutator {
	return &validateLifecycleStarted{engine: e}
}

func (m *validateLifecycleStarted) Name() string {
	return "ValidateLifecycleStarted"
}

func (m *validateLifecycleStarted) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics
	for _, scope := range lifecycleStartedScopes {
		for key, resource := range scope.getMap(u) {
			lws, ok := resource.(lifecycleStartedReader)
			if !ok || !lws.HasLifecycleStarted() {
				continue
			}
			path := "resources." + scope.pluralName + "." + key + ".lifecycle.started"
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "lifecycle.started is only supported in direct deployment mode",
				Locations: u.Config.GetLocations(path),
			})
		}
	}
	return diags
}
