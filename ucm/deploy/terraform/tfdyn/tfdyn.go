package tfdyn

import (
	"context"
	"fmt"
	"sort"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
)

// providerSource and providerVersion mirror the constants in the parent
// terraform package. Duplicated (rather than imported) to keep this
// subpackage import-cycle-free — the parent package imports tfdyn.
const (
	providerSource  = "databricks/databricks"
	providerVersion = "1.112.0"
)

// convertOrder controls the order in which resource kinds are walked. The
// ordering matters because downstream converters inspect earlier ones
// (schemas look at Resources.Catalog to decide whether to emit depends_on;
// grants look at Resources.Catalog and Resources.Schema).
var convertOrder = []string{"storage_credentials", "external_locations", "catalogs", "schemas", "volumes", "connections", "grants"}

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

// buildResourceTree assembles the top-level Terraform JSON tree. The output
// shape mirrors what bundle/deploy/terraform writes so `terraform init`
// resolves the databricks provider out of the databricks/databricks
// namespace instead of defaulting to hashicorp/databricks (which does not
// exist in the registry).
func buildResourceTree(out *Resources) dyn.Value {
	blocks := []struct {
		tfType string
		values map[string]dyn.Value
	}{
		{"databricks_storage_credential", out.StorageCredential},
		{"databricks_external_location", out.ExternalLocation},
		{"databricks_catalog", out.Catalog},
		{"databricks_schema", out.Schema},
		{"databricks_volume", out.Volume},
		{"databricks_connection", out.Connection},
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

	rootPairs := []dyn.Pair{
		{Key: dyn.V("terraform"), Value: buildTerraformBlock()},
		{Key: dyn.V("provider"), Value: buildProviderBlock()},
		{
			Key:   dyn.V("resource"),
			Value: dyn.V(dyn.NewMappingFromPairs(resourcePairs)),
		},
	}
	return dyn.V(dyn.NewMappingFromPairs(rootPairs))
}

// buildTerraformBlock returns the `terraform.required_providers.databricks`
// value that pins the provider source and version.
func buildTerraformBlock() dyn.Value {
	databricks := dyn.V(dyn.NewMappingFromPairs([]dyn.Pair{
		{Key: dyn.V("source"), Value: dyn.V(providerSource)},
		{Key: dyn.V("version"), Value: dyn.V(providerVersion)},
	}))
	required := dyn.V(dyn.NewMappingFromPairs([]dyn.Pair{
		{Key: dyn.V("databricks"), Value: databricks},
	}))
	return dyn.V(dyn.NewMappingFromPairs([]dyn.Pair{
		{Key: dyn.V("required_providers"), Value: required},
	}))
}

// buildProviderBlock returns an empty `provider.databricks` block. The
// databricks terraform provider picks up its auth from the DATABRICKS_*
// env vars that buildEnv passes through to terraform-exec, so no fields
// need to be set here — the block's presence is what matters.
func buildProviderBlock() dyn.Value {
	return dyn.V(dyn.NewMappingFromPairs([]dyn.Pair{
		{Key: dyn.V("databricks"), Value: dyn.V(dyn.NewMapping())},
	}))
}
