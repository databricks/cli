// This is in a separate package to avoid import cycles because it is imported by both terraform and statemgmt.
package resourcestate

// ResourceState stores relevant from terraform/direct state for one resoruce
type ResourceState struct {
	ID string

	// For dashboards
	ETag string

	// Size in bytes of the resource's serialized state blob. Populated by the
	// direct engine (len of the JSON stored in resources.json) for deploy
	// telemetry; left zero by the terraform path.
	StateSizeBytes int

	// Size in bytes of the resource's serialized state blob after zstd
	// compression. Populated by the direct engine alongside StateSizeBytes for
	// deploy telemetry; left zero by the terraform path. Used to gauge how much
	// resource state shrinks under compression (the deployment metadata service
	// stores state zstd-compressed).
	StateCompressedSizeBytes int
}

// ExportedResourcesMap stores relevant attributes from terraform/direct state for all resources
// Maps resource key (e.g. "resources.jobs.foo", "resources.jobs.foo.permissions") -> ResourceState
type ExportedResourcesMap map[string]ResourceState
