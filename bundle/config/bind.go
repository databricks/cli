package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/diag"
)

// BindResource represents a single resource to bind with its workspace ID.
type BindResource struct {
	ID string `json:"id"`
}

// Bind defines existing workspace resources to bring under bundle management at deploy
// time. The outer map key is the resource type (e.g. "jobs", "pipelines"), the inner
// key is the resource name in the bundle config, and the value carries the workspace
// resource ID. Bind blocks are only valid for the direct deployment engine.
type Bind map[string]map[string]BindResource

// ForEach calls fn for each bind entry in the configuration. Iteration order is
// stable (sorted by resource type, then resource name) so callers that emit user-
// visible diagnostics get deterministic output across runs.
func (b Bind) ForEach(fn func(resourceType, resourceName, bindID string)) {
	resourceTypes := make([]string, 0, len(b))
	for resourceType := range b {
		resourceTypes = append(resourceTypes, resourceType)
	}
	slices.Sort(resourceTypes)
	for _, resourceType := range resourceTypes {
		resources := b[resourceType]
		names := make([]string, 0, len(resources))
		for name := range resources {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			fn(resourceType, name, resources[name].ID)
		}
	}
}

// IsEmpty returns true if no binds are defined.
func (b Bind) IsEmpty() bool {
	for _, resources := range b {
		if len(resources) > 0 {
			return false
		}
	}
	return true
}

// Validate rejects bind blocks that target child resources (e.g. "jobs.permissions").
// The direct engine exposes child resources as full keys like "jobs.permissions" in its
// resource registry, but they are not bindable on their own — bind the parent resource
// and let the bundle manage the child entries declaratively.
func (b Bind) Validate() diag.Diagnostics {
	var diags diag.Diagnostics

	for resourceType, resources := range b {
		if !strings.HasSuffix(resourceType, ".permissions") && !strings.HasSuffix(resourceType, ".grants") {
			continue
		}
		for resourceName := range resources {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("binding %s is not allowed", resourceType),
				Detail: fmt.Sprintf(
					"bind can only be used for resources directly under the resources block, not for child resources like permissions or grants.\n\n"+
						"To manage permissions or grants:\n"+
						"1. First bind the parent resource (without .permissions or .grants)\n"+
						"2. Then define permissions or grants in your bundle configuration\n\n"+
						"Invalid bind configuration:\n"+
						"  bind:\n"+
						"    %s:\n"+
						"      %s:\n"+
						"        id: ...\n\n"+
						"Instead, remove this bind entry and ensure the parent resource is bound.",
					resourceType, resourceName),
			})
		}
	}

	return diags
}
