// This is in a separate package to avoid import cycles because it is imported by both terraform and statemgmt.
package resourcestate

// ResourceState stores relevant from terraform/direct state for one resoruce
type ResourceState struct {
	ID string

	// For dashboards
	ETag string

	// For job_runs: the resolved job_id the run was launched against. A run's
	// URL needs its parent job id, but in config that comes from a
	// ${resources.jobs.*.id} reference that is only resolved at deploy, so at
	// read time (e.g. bundle summary) it reads 0. We carry the deployed value
	// here so it can be restored into config at load time.
	JobID int64

	// Size in bytes of the resource's serialized state blob. Populated by the
	// direct engine (len of the JSON stored in resources.json) for deploy
	// telemetry; left zero by the terraform path.
	StateSizeBytes int
}

// ExportedResourcesMap stores relevant attributes from terraform/direct state for all resources
// Maps resource key (e.g. "resources.jobs.foo", "resources.jobs.foo.permissions") -> ResourceState
type ExportedResourcesMap map[string]ResourceState
