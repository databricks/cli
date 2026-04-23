package validate

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/ucm"
)

// UniqueResourceKeys errors when the same key appears under multiple resource
// kinds (e.g. a catalog and a schema both named "bronze"). Uniqueness within
// a single kind is enforced by YAML map semantics at load time, so this
// validator only needs to catch cross-kind collisions.
//
// This complements FlattenNestedResources' same-kind check between flat and
// nested forms.
func UniqueResourceKeys() ucm.Mutator { return &uniqueResourceKeys{} }

type uniqueResourceKeys struct{}

func (m *uniqueResourceKeys) Name() string { return "validate:unique_resource_keys" }

func (m *uniqueResourceKeys) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	type occurrence struct {
		kind string
		path dyn.Path
		locs []dyn.Location
	}
	occurrences := map[string][]occurrence{}

	root := u.Config.Value()
	resources := root.Get("resources")
	if resources.Kind() == dyn.KindInvalid || resources.Kind() == dyn.KindNil {
		return nil
	}

	for _, kind := range resourceKinds {
		kv := resources.Get(kind)
		if kv.Kind() != dyn.KindMap {
			continue
		}
		for _, pair := range kv.MustMap().Pairs() {
			key := pair.Key.MustString()
			p := dyn.NewPath(dyn.Key("resources"), dyn.Key(kind), dyn.Key(key))
			occurrences[key] = append(occurrences[key], occurrence{
				kind: kind,
				path: p,
				locs: pair.Value.Locations(),
			})
		}
	}

	var diags diag.Diagnostics
	for _, key := range sortedKeys(occurrences) {
		occs := occurrences[key]
		if len(occs) <= 1 {
			continue
		}
		var paths []dyn.Path
		var locs []dyn.Location
		for _, o := range occs {
			paths = append(paths, o.path)
			locs = append(locs, o.locs...)
		}
		slices.SortFunc(locs, func(a, b dyn.Location) int {
			if n := cmp.Compare(a.File, b.File); n != 0 {
				return n
			}
			if n := cmp.Compare(a.Line, b.Line); n != 0 {
				return n
			}
			return cmp.Compare(a.Column, b.Column)
		})
		slices.SortFunc(paths, func(a, b dyn.Path) int {
			return cmp.Compare(a.String(), b.String())
		})
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "resource key " + key + " is declared under multiple resource kinds",
			Paths:     paths,
			Locations: locs,
		})
	}
	return diags
}
