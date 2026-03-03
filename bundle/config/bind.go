package config

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
