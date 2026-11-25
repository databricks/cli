// This is in a separate package to avoid import cycles because it is imported by both terraform and statemgmt.
package resourcestate

// ResourceState stores relevant from terraform/direct state for one resoruce
type ResourceState struct {
	ID string

	// The etag the backend returned, for resource types that keep one in state
	// (etag_field in their direct engine config).
	ETag string

	// The resolved id of the job the resource was launched against, for resource
	// types that declare job_id_field in their direct engine config. In config it
	// is a reference resolved only at deploy, so we carry the deployed value to
	// restore it at load time.
	JobID int64

	// Size in bytes of the resource's serialized state blob. Populated by the
	// direct engine (len of the JSON stored in resources.json) for deploy
	// telemetry; left zero by the terraform path.
	StateSizeBytes int
}

// ExportedResourcesMap stores relevant attributes from terraform/direct state for all resources
// Maps resource key (e.g. "resources.jobs.foo", "resources.jobs.foo.permissions") -> ResourceState
type ExportedResourcesMap map[string]ResourceState
