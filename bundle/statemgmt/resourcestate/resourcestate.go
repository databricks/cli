// This is in a separate package to avoid import cycles because it is imported by both terraform and statemgmt.
package resourcestate

type ResourceState struct {
	ID string

	// For dashboards
	ETag string
}

// maps group (e.g. "jobs") -> name -> ExportedStateAttributes
type ExportedResourcesMap map[string]map[string]ResourceState
