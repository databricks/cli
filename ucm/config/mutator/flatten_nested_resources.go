package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
)

type flattenNestedResources struct{}

// FlattenNestedResources unrolls the nested forms of schemas and grants
// (declared under a catalog, or grants declared under a schema) into the flat
// Root.Resources.{Schemas,Grants} maps. Injects parent references so the
// post-flatten tree is indistinguishable from a purely flat declaration.
//
// Runs first in the load phase, before any other mutator reads the resources
// tree. After it runs, catalog.schemas, catalog.grants, and schema.grants are
// all nil.
//
// Collision handling: a nested entry whose key already exists as a flat entry
// emits a diag.Error pointing at the nested location. Same for a user-supplied
// field that conflicts with the injected parent reference (e.g. a nested
// schema that sets `catalog: other`).
func FlattenNestedResources() ucm.Mutator { return &flattenNestedResources{} }

func (m *flattenNestedResources) Name() string { return "FlattenNestedResources" }

func (m *flattenNestedResources) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	err := u.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		resourcesValue := root.Get("resources")
		resources, ok := resourcesValue.AsMap()
		if !ok {
			return root, nil
		}

		catalogsValue, _ := resources.GetByString("catalogs")
		catalogs, ok := catalogsValue.AsMap()
		if !ok {
			return root, nil
		}

		flatSchemas, _ := resources.GetByString("schemas")
		flatGrants, _ := resources.GetByString("grants")
		schemasMap := mapOrNew(flatSchemas)
		grantsMap := mapOrNew(flatGrants)

		newCatalogs := dyn.NewMapping()
		for _, cp := range catalogs.Pairs() {
			catalogName := cp.Key.MustString()
			catalogBody, ok := cp.Value.AsMap()
			if !ok {
				newCatalogs.SetLoc(catalogName, cp.Key.Locations(), cp.Value)
				continue
			}

			if nested, ok := catalogBody.GetByString("schemas"); ok {
				diags = append(diags, liftNestedSchemas(catalogName, nested, &schemasMap, &grantsMap)...)
			}
			if nested, ok := catalogBody.GetByString("grants"); ok {
				diags = append(diags, liftNestedCatalogGrants(catalogName, nested, &grantsMap)...)
			}

			stripped := removeKeys(catalogBody, "schemas", "grants")
			newCatalogs.SetLoc(catalogName, cp.Key.Locations(), dyn.NewValue(stripped, cp.Value.Locations()))
		}

		newResources := resources.Clone()
		newResources.SetLoc("catalogs", nil, dyn.NewValue(newCatalogs, catalogsValue.Locations()))
		if schemasMap.Len() > 0 {
			newResources.SetLoc("schemas", nil, dyn.NewValue(schemasMap, flatSchemas.Locations()))
		}
		if grantsMap.Len() > 0 {
			newResources.SetLoc("grants", nil, dyn.NewValue(grantsMap, flatGrants.Locations()))
		}

		return dyn.SetByPath(root, dyn.NewPath(dyn.Key("resources")),
			dyn.NewValue(newResources, resourcesValue.Locations()))
	})
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

// liftNestedSchemas moves catalog.schemas into the top-level schemas map,
// injecting `catalog` and processing any schema-nested grants.
func liftNestedSchemas(catalogName string, nested dyn.Value, schemas, grants *dyn.Mapping) diag.Diagnostics {
	var diags diag.Diagnostics
	nestedMap, ok := nested.AsMap()
	if !ok {
		return diags
	}
	for _, sp := range nestedMap.Pairs() {
		schemaName := sp.Key.MustString()
		schemaValue, ok := sp.Value.AsMap()
		if !ok {
			continue
		}
		updated := schemaValue.Clone()

		if d := injectStringField(&updated, "catalog_name", catalogName,
			fmt.Sprintf("schema %q nested under catalog %q", schemaName, catalogName),
			sp.Key.Locations()); d != nil {
			diags = append(diags, d...)
		}

		if nestedGrants, ok := updated.GetByString("grants"); ok {
			diags = append(diags, liftNestedSchemaGrants(schemaName, nestedGrants, grants)...)
		}

		flatValue := dyn.NewValue(removeKeys(updated, "grants"), sp.Value.Locations())

		if existing, ok := schemas.GetPairByString(schemaName); ok {
			diags = append(diags, collisionDiag("schema", schemaName, sp.Key.Locations(), existing.Key.Locations())...)
			continue
		}
		schemas.SetLoc(schemaName, sp.Key.Locations(), flatValue)
	}
	return diags
}

// liftNestedCatalogGrants injects securable={catalog, <name>} into nested
// catalog grants and moves them into the flat grants map.
func liftNestedCatalogGrants(catalogName string, nested dyn.Value, grants *dyn.Mapping) diag.Diagnostics {
	return liftNestedGrants(nested, grants, "catalog", catalogName,
		fmt.Sprintf("grant nested under catalog %q", catalogName))
}

// liftNestedSchemaGrants injects securable={schema, <name>} into nested
// schema grants and moves them into the flat grants map.
func liftNestedSchemaGrants(schemaName string, nested dyn.Value, grants *dyn.Mapping) diag.Diagnostics {
	return liftNestedGrants(nested, grants, "schema", schemaName,
		fmt.Sprintf("grant nested under schema %q", schemaName))
}

func liftNestedGrants(nested dyn.Value, grants *dyn.Mapping, kind, parentName, ctxDesc string) diag.Diagnostics {
	var diags diag.Diagnostics
	nestedMap, ok := nested.AsMap()
	if !ok {
		return diags
	}
	for _, gp := range nestedMap.Pairs() {
		grantName := gp.Key.MustString()
		grantValue, ok := gp.Value.AsMap()
		if !ok {
			continue
		}
		updated := grantValue.Clone()

		if d := injectSecurable(&updated, kind, parentName, ctxDesc, gp.Key.Locations()); d != nil {
			diags = append(diags, d...)
		}

		flatValue := dyn.NewValue(updated, gp.Value.Locations())
		if existing, ok := grants.GetPairByString(grantName); ok {
			diags = append(diags, collisionDiag("grant", grantName, gp.Key.Locations(), existing.Key.Locations())...)
			continue
		}
		grants.SetLoc(grantName, gp.Key.Locations(), flatValue)
	}
	return diags
}

// injectStringField writes key=want into m. If m already has key with a
// different value, emits an error diagnostic.
func injectStringField(m *dyn.Mapping, key, want, ctxDesc string, locs []dyn.Location) diag.Diagnostics {
	existing, ok := m.GetByString(key)
	if ok {
		if got, _ := existing.AsString(); got != "" && got != want {
			return diag.Diagnostics{{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("%s: %s=%q conflicts with parent %q", ctxDesc, key, got, want),
				Locations: locs,
			}}
		}
	}
	m.SetLoc(key, locs, dyn.V(want))
	return nil
}

// injectSecurable writes securable={type: kind, name: parentName} into m. If
// m already has a securable that differs, emits an error.
func injectSecurable(m *dyn.Mapping, kind, parentName, ctxDesc string, locs []dyn.Location) diag.Diagnostics {
	want := dyn.V(map[string]dyn.Value{
		"type": dyn.V(kind),
		"name": dyn.V(parentName),
	})
	existing, ok := m.GetByString("securable")
	if ok {
		existingMap, _ := existing.AsMap()
		gotType, _ := existingMap.GetByString("type")
		gotName, _ := existingMap.GetByString("name")
		gt, _ := gotType.AsString()
		gn, _ := gotName.AsString()
		if (gt != "" && gt != kind) || (gn != "" && gn != parentName) {
			return diag.Diagnostics{{
				Severity: diag.Error,
				Summary: fmt.Sprintf(
					"%s: securable=%s/%s conflicts with parent %s/%s",
					ctxDesc, gt, gn, kind, parentName,
				),
				Locations: locs,
			}}
		}
	}
	m.SetLoc("securable", locs, want)
	return nil
}

func collisionDiag(kind, name string, nestedLocs, flatLocs []dyn.Location) diag.Diagnostics {
	locs := append([]dyn.Location{}, nestedLocs...)
	locs = append(locs, flatLocs...)
	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s %q is declared both as a flat entry and nested under its parent", kind, name),
		Locations: locs,
	}}
}

func mapOrNew(v dyn.Value) dyn.Mapping {
	m, ok := v.AsMap()
	if !ok {
		return dyn.NewMapping()
	}
	return m.Clone()
}

// removeKeys returns a new Mapping with the named keys dropped.
func removeKeys(m dyn.Mapping, keys ...string) dyn.Mapping {
	drop := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		drop[k] = struct{}{}
	}
	out := dyn.NewMapping()
	for _, p := range m.Pairs() {
		if _, skip := drop[p.Key.MustString()]; skip {
			continue
		}
		out.SetLoc(p.Key.MustString(), p.Key.Locations(), p.Value)
	}
	return out
}
