package tfdyn

import (
	"context"
	"fmt"
	"sort"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
)

// convertOrder controls the order in which resource kinds are walked. The
// ordering matters because downstream converters inspect earlier ones
// (schemas look at Resources.Catalog to decide whether to emit depends_on;
// grants look at Resources.Catalog and Resources.Schema).
var convertOrder = []string{"catalogs", "schemas", "grants"}

// Convert walks a ucm configuration and produces the Terraform JSON
// resource tree suitable for writing as a .tf.json file. The returned
// dyn.Value is shaped as:
//
//	{
//	  "resource": {
//	    "databricks_catalog": { "<key>": { ... } },
//	    "databricks_schema":  { "<key>": { ... } },
//	    "databricks_grants":  { "<key>": { ... } }
//	  }
//	}
//
// Empty resource kinds are omitted.
func Convert(ctx context.Context, u *ucm.Ucm) (dyn.Value, error) {
	out := NewResources()

	resourcesVal, err := dyn.GetByPath(u.Config.Value(), dyn.NewPath(dyn.Key("resources")))
	if err != nil {
		// No resources: emit an empty terraform file rather than failing.
		resourcesVal = dyn.V(map[string]dyn.Value{})
	}

	for _, kind := range convertOrder {
		conv, ok := GetConverter(kind)
		if !ok {
			continue
		}
		bucket := resourcesVal.Get(kind)
		if !bucket.IsValid() {
			continue
		}
		m, ok := bucket.AsMap()
		if !ok {
			continue
		}

		keys := make([]string, 0, m.Len())
		for _, p := range m.Pairs() {
			keys = append(keys, p.Key.MustString())
		}
		sort.Strings(keys)

		for _, key := range keys {
			vin, _ := m.GetByString(key)
			if err := conv.Convert(ctx, key, vin, out); err != nil {
				return dyn.InvalidValue, fmt.Errorf("convert %s.%s: %w", kind, key, err)
			}
		}
	}

	return buildResourceTree(out), nil
}

// buildResourceTree assembles the `{resource: {<tf_type>: {<key>: ...}}}`
// dyn.Value from the per-kind maps produced by the converters.
func buildResourceTree(out *Resources) dyn.Value {
	blocks := []struct {
		tfType string
		values map[string]dyn.Value
	}{
		{"databricks_catalog", out.Catalog},
		{"databricks_schema", out.Schema},
		{"databricks_grants", out.Grants},
	}

	var resourcePairs []dyn.Pair
	for _, b := range blocks {
		if len(b.values) == 0 {
			continue
		}
		keys := make([]string, 0, len(b.values))
		for k := range b.values {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		pairs := make([]dyn.Pair, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, dyn.Pair{
				Key:   dyn.V(k),
				Value: b.values[k],
			})
		}
		resourcePairs = append(resourcePairs, dyn.Pair{
			Key:   dyn.V(b.tfType),
			Value: dyn.V(dyn.NewMappingFromPairs(pairs)),
		})
	}

	return dyn.V(dyn.NewMappingFromPairs([]dyn.Pair{
		{
			Key:   dyn.V("resource"),
			Value: dyn.V(dyn.NewMappingFromPairs(resourcePairs)),
		},
	}))
}
