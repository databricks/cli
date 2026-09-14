package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// MaxStateSizeBytes is the largest serialized resource state the deployment
// metadata service accepts, matching MAX_STATE_BYTES on the service side. A
// resource above it is rejected there with InvalidArgument, so we check it up
// front to fail during validate rather than midway through a deploy.
const MaxStateSizeBytes = 64 * 1024

type validateStateSize struct {
	bundle.RO
	engine engine.EngineType
}

// ValidateStateSize reports resources whose serialized state exceeds
// MaxStateSizeBytes and so cannot be recorded as deployment history.
func ValidateStateSize(e engine.EngineType) bundle.ReadOnlyMutator {
	return &validateStateSize{engine: e}
}

func (v *validateStateSize) Name() string {
	return "validate:state_size"
}

func (v *validateStateSize) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Only the direct engine records state with the metadata service, and only
	// when the bundle opted in. Terraform deployments never upload this state.
	if !v.engine.IsDirect() {
		return nil
	}
	if b.Config.Experimental == nil || !b.Config.Experimental.RecordDeploymentHistory {
		return nil
	}

	// The adapters only hold the client, so a nil one is enough to reach
	// PrepareState. Nothing here issues a request, which keeps this check in
	// the fast, in-memory half of validation.
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	for _, key := range sortedResourceKeys(b) {
		resourceType := config.GetResourceTypeFromKey(key)
		adapter, ok := adapters[resourceType]
		if !ok {
			// Resource types the direct engine does not support are rejected
			// during planning, with a better message than we could give here.
			continue
		}

		size, err := stateSize(b, adapter, key)
		if err != nil {
			return diag.FromErr(err)
		}
		if size <= MaxStateSizeBytes {
			continue
		}

		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("%s has a serialized state of %d bytes, which exceeds the %d byte limit for recording deployment history", key, size, MaxStateSizeBytes),
			Detail:    "Deployment history records the state of each resource, and this resource is too large to record.\n" + sizeAdvice(resourceType),
			Locations: b.Config.GetLocations(key),
			Paths:     []dyn.Path{dyn.MustPathFromString(key)},
		})
	}

	return diags
}

// sizeAdvice returns the remediation hint for an oversized resource of the given
// type. The limit is per resource, so the way out is always to split the
// definition into smaller ones; the wording names the split that applies.
//
// Note that file_path on alerts, dashboards and genie spaces is deliberately not
// suggested: those mutators inline the file into the resource during initialize,
// so the recorded state is the same size either way.
func sizeAdvice(resourceType string) string {
	switch resourceType {
	case "jobs":
		return "The limit applies per resource, so split this job into multiple jobs, each with fewer tasks."
	case "alerts":
		return "The limit applies per resource, so split this alert into multiple alerts, each covering fewer conditions."
	case "pipelines":
		return "The limit applies per resource, so split this pipeline into multiple pipelines, each with fewer libraries."
	default:
		return "The limit applies per resource, so split this resource into multiple smaller resources."
	}
}

// stateSize returns the size of the state that would be recorded for the
// resource at key: the state prepared from config, serialized as JSON.
func stateSize(b *bundle.Bundle, adapter *dresources.Adapter, key string) (int, error) {
	inputConfig, err := b.Config.GetResourceConfig(key)
	if err != nil {
		return 0, fmt.Errorf("cannot read config for %s: %w", key, err)
	}

	state, err := adapter.PrepareState(inputConfig)
	if err != nil {
		return 0, fmt.Errorf("cannot prepare state for %s: %w", key, err)
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return 0, fmt.Errorf("cannot serialize state for %s: %w", key, err)
	}

	return len(raw), nil
}

// sortedResourceKeys returns the bundle's resource keys ("resources.jobs.foo")
// in a stable order, so diagnostics do not depend on map iteration order.
func sortedResourceKeys(b *bundle.Bundle) []string {
	var keys []string
	for _, group := range b.Config.Resources.AllResources() {
		for name := range group.Resources {
			keys = append(keys, "resources."+group.Description.PluralName+"."+name)
		}
	}
	slices.Sort(keys)
	return keys
}
