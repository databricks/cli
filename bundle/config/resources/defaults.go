package resources

// DefaultsContext holds the bundle settings that field defaults depend on.
type DefaultsContext struct {
	// ResourcePath is workspace.resource_path.
	ResourcePath string
}

// FieldDefault is the value that a field of a newly added resource gets if
// the configuration does not set it.
type FieldDefault struct {
	// Path is relative to the resource, for example "queue" or
	// "job_clusters[*].new_cluster.workload_type.clients.jobs".
	Path  string
	Value any
}

// FieldDefaulter is implemented by resource types with fields that have a default.
type FieldDefaulter interface {
	// FieldDefaults returns the defaults of the type, in the order they are applied.
	FieldDefaults(c DefaultsContext) []FieldDefault
}
