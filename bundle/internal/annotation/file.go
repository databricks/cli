package annotation

// File is the in-memory representation of the annotations, keyed by Go type
// path and field name, e.g.:
// github.com/databricks/cli/bundle/config.Bundle:
//
//	cluster_id:
//	   description: "Description"
//
// The key "_" holds the annotation for the type itself.
type File map[string]map[string]Descriptor
