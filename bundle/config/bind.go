package config

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/diag"
)

// BindResource represents a single resource to bind with its workspace ID.
type BindResource struct {
	ID string `json:"id"`
}

// Bind defines resources to bind at the target level.
// Resources listed here will be bound to the bundle at deploy time.
// This field is only valid for the direct deployment engine.
//
// The outer map key is the resource type (e.g., "jobs", "pipelines"),
// and the inner map key is the resource name in the bundle configuration.
type Bind map[string]map[string]BindResource

// GetBindID returns the bind ID for a given resource type and name.
// Returns empty string if no bind is defined for the resource.
func (i Bind) GetBindID(resourceType, resourceName string) string {
	if r, ok := i[resourceType][resourceName]; ok {
		return r.ID
	}
	return ""
}

// ForEach calls fn for each bind entry in the configuration.
func (i Bind) ForEach(fn func(resourceType, resourceName, bindID string)) {
	for resourceType, resources := range i {
		for name, r := range resources {
			fn(resourceType, name, r.ID)
		}
	}
}

// IsEmpty returns true if no binds are defined.
func (i Bind) IsEmpty() bool {
	for _, resources := range i {
		if len(resources) > 0 {
			return false
		}
	}
	return true
}

// Validate checks if the bind configuration contains any invalid resource types.
// Binding is only allowed for resources directly under the resources block,
// not for child resources like permissions or grants.
func (i Bind) Validate() diag.Diagnostics {
	var diags diag.Diagnostics

	for resourceType, resources := range i {
		// Check if this is a child resource (permissions or grants)
		if strings.Contains(resourceType, ".permissions") || strings.Contains(resourceType, ".grants") {
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
	}

	return diags
}
