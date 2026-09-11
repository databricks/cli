package config_test

// TestNoSameDepthJSONShadows asserts that no resource config type has two
// anonymous embedded structs that both declare the same json name at the same
// embedding depth.
//
// Same-depth collisions are the only shadow that causes real problems:
// encoding/json calls the name ambiguous and serializes neither field, so the
// field silently disappears from the wire format and cannot be read or written
// by structaccess. A depth-mismatch shadow (a direct named field overriding an
// embedded one) is intentional and handled correctly — the shallower field wins.
//
// The test is a guard against accidentally introducing a new same-depth
// collision when adding fields to BaseResource, adding a new SDK embed, or
// creating a new resource type.

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/stretchr/testify/assert"
)

// knownSameDepthCollisions lists the existing same-depth json name collisions.
// These are bugs — encoding/json calls the name ambiguous and neither field is
// reachable — but fixing them requires a breaking change to the bundle YAML
// format, so they are listed here rather than silently tolerated. Every entry
// must describe where the collision comes from; fixing one removes its entry.
//
// Fix: add an explicit depth-0 field on the resource type that shadows both
// embedded declarations (e.g. `ID string \`json:"id,omitempty" bundle:"readonly"\“
// directly on the resource struct).
var knownSameDepthCollisions = map[string][]string{
	// BaseResource.ID and pipelines.CreatePipeline.Id both carry json:"id".
	"pipelines": {"id"},
	// BaseResource.ID and apps.App.Id carry json:"id";
	// BaseResource.URL and apps.App.Url carry json:"url".
	"apps": {"id", "url"},
	// BaseResource.ID and sql.AlertV2.Id carry json:"id".
	"alerts": {"id"},
}

func TestNoSameDepthJSONShadows(t *testing.T) {
	rt := reflect.TypeFor[config.Resources]()
	var newCollisions []string

	for i := range rt.NumField() {
		f := rt.Field(i)
		et := f.Type.Elem()
		for et.Kind() == reflect.Pointer {
			et = et.Elem()
		}
		if et.Kind() != reflect.Struct {
			continue
		}
		group := structtag.JSONTag(f.Tag.Get("json")).Name()

		for _, c := range sameDepthCollisions(et) {
			known := false
			for _, k := range knownSameDepthCollisions[group] {
				if k == c.name {
					known = true
					break
				}
			}
			if !known {
				newCollisions = append(newCollisions,
					fmt.Sprintf("%s (%s): json name %q declared by %s and %s at the same embedding depth",
						group, et, c.name, c.typeA, c.typeB))
			}
		}
	}

	assert.Empty(t, newCollisions,
		"NEW same-depth json name collisions found — encoding/json calls these ambiguous "+
			"and serializes neither; structaccess cannot read or write them either. "+
			"Fix by adding an explicit depth-0 field on the resource struct, or add to knownSameDepthCollisions.")
}

type collision struct {
	name, typeA, typeB string
}

// sameDepthCollisions returns the json names declared at the same embedding
// depth by two or more anonymous embedded structs inside t.
func sameDepthCollisions(t reflect.Type) []collision {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	var result []collision
	seen := map[reflect.Type]bool{t: true}
	level := embeddedTypes(t)

	for len(level) > 0 {
		nameToTypes := map[string][]string{}
		for _, ft := range level {
			for i := range ft.NumField() {
				sf := ft.Field(i)
				if sf.PkgPath != "" || sf.Anonymous {
					continue
				}
				name := structtag.JSONTag(sf.Tag.Get("json")).Name()
				if name == "" {
					name = sf.Name
				}
				if name == "-" || sf.Name == "ForceSendFields" {
					continue
				}
				nameToTypes[name] = append(nameToTypes[name], ft.String())
			}
		}
		for name, types := range nameToTypes {
			if len(types) > 1 {
				result = append(result, collision{name: name, typeA: types[0], typeB: types[1]})
			}
		}

		var next []reflect.Type
		for _, ft := range level {
			for _, embedded := range embeddedTypes(ft) {
				if !seen[embedded] {
					seen[embedded] = true
					next = append(next, embedded)
				}
			}
		}
		level = next
	}
	return result
}

func embeddedTypes(t reflect.Type) []reflect.Type {
	var out []reflect.Type
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.Anonymous {
			continue
		}
		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			out = append(out, ft)
		}
	}
	return out
}
