package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
)

type Converter interface {
	Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error
}

type converterEntry struct {
	Converter     Converter
	TerraformName string
}

var converters = map[string]converterEntry{}

func GetConverter(name string) (Converter, bool) {
	e, ok := converters[name]
	return e.Converter, ok
}

func registerConverter(groupName, terraformName string, c Converter) {
	converters[groupName] = converterEntry{Converter: c, TerraformName: terraformName}
}

// BuildGroupToTerraformName dynamically builds the group-to-terraform-name
// map from registered converters. It also adds entries for "permissions" and
// "grants" which don't have converters.
func BuildGroupToTerraformName() map[string]string {
	m := make(map[string]string, len(converters)+2)
	for groupName, entry := range converters {
		m[groupName] = entry.TerraformName
	}
	m["permissions"] = "databricks_permissions"
	m["grants"] = "databricks_grants"
	return m
}
