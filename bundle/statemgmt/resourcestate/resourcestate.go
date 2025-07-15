// This is in a separate package to avoid import cycles because it is imported by both terraform and statemgmt.
package resourcestate

// ResourceState stores relevant from terraform/terranova state for one resoruce
type ResourceState struct {
	ID string

	// For dashboards
	ETag string
}

// ExportedResourcesMap stores relevant attributes from terraform/terranova state for all resources
// Maps group (e.g. "jobs") -> resource name -> ResourceState
type ExportedResourcesMap map[string]map[string]ResourceState
