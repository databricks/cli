// Package tfdyn converts a ucm configuration tree into the Terraform JSON
// resource map consumed by the terraform wrapper in ucm/deploy/terraform.
//
// Each converter mirrors the shape of bundle/deploy/terraform/tfdyn but
// emits dyn.Value trees directly rather than typed TF schema structs. This
// keeps source-location metadata intact so downstream diagnostics can point
// back to the originating ucm.yml span.
package tfdyn

import (
	"context"

	"github.com/databricks/cli/libs/dyn"
)

// Resources is the accumulated Terraform resource payload produced while
// walking a ucm configuration. Each map is keyed by the resource key used in
// ucm.yml. The wrapper in U5 combines these into a top-level `resource`
// block.
type Resources struct {
	Catalog           map[string]dyn.Value
	Schema            map[string]dyn.Value
	Grants            map[string]dyn.Value
	StorageCredential map[string]dyn.Value
	ExternalLocation  map[string]dyn.Value
}

// NewResources returns an empty Resources ready for converters to fill.
func NewResources() *Resources {
	return &Resources{
		Catalog:           map[string]dyn.Value{},
		Schema:            map[string]dyn.Value{},
		Grants:            map[string]dyn.Value{},
		StorageCredential: map[string]dyn.Value{},
		ExternalLocation:  map[string]dyn.Value{},
	}
}

// Converter converts a single resource entry from the ucm tree into one or
// more Terraform resource blocks on out.
type Converter interface {
	Convert(ctx context.Context, key string, vin dyn.Value, out *Resources) error
}

var converters = map[string]Converter{}

// GetConverter returns the converter registered for the given ucm resource
// kind ("catalogs", "schemas", "grants").
func GetConverter(name string) (Converter, bool) {
	c, ok := converters[name]
	return c, ok
}

func registerConverter(name string, c Converter) {
	converters[name] = c
}
