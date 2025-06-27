package statemgmt

type ResourceState struct {
	ID string

	// For dashboards
	ETag string
}

// maps group (e.g. "jobs") -> name -> ExportedStateAttributes
type ExportedResourcesMap map[string]map[string]ResourceState
