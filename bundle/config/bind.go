package config

import (
	"fmt"
	"slices"

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

// Validate rejects bind blocks whose resource type is not a supported bundle
// resource. Only the resource types that can appear under the resources block are
// bindable; anything else (a typo, a child key like "jobs.permissions", etc.) is
// reported here rather than surfacing later as a confusing planning error.
func (b Bind) Validate() diag.Diagnostics {
	var diags diag.Diagnostics

	supported := SupportedResources()
	b.ForEach(func(resourceType, resourceName, bindID string) {
		if _, ok := supported[resourceType]; ok {
			return
		}
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("binding %q is not supported", resourceType),
			Detail:   fmt.Sprintf("%q is not a supported resource type; bind is only supported for resources that can be defined under the resources block.", resourceType),
		})
	})

	return diags
}
