// This is in a separate package to avoid import cycles because it is imported by both terraform and statemgmt.
package resourcestate

// ResourceState stores relevant from terraform/direct state for one resoruce
type ResourceState struct {
	ID string

	// For dashboards
	ETag string
}

// ExportedResourcesMap stores relevant attributes from terraform/direct state for all resources
// Maps resource key (e.g. "resources.jobs.foo", "resources.jobs.foo.permissions") -> ResourceState
type ExportedResourcesMap map[string]ResourceState
