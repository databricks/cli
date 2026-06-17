package main

import (
	"container/list"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
)

// typeGraph captures, for every annotatable type reachable from the root
// config type, the properties the JSON schema generator emits for it and the
// type each property resolves to. The property set and the resolution targets
// come from jsonschema.FromType itself so the graph cannot drift from the
// generated schema; reflection is only used to recover struct declaration
// order, which the schema's Properties map loses.
type typeGraph struct {
	// root is the type path of the root type.
	root string
	// fields maps a type path to its properties in struct declaration order.
	// Non-struct types (enums) are present with no properties.
	fields map[string][]fieldEdge
}

// fieldEdge is one property of a type: the property name and the type path of
// the type it resolves to after unwrapping pointers, slices and maps. typ is
// empty for properties whose type cannot carry annotations (primitives,
// interface).
type fieldEdge struct {
	name string
	typ  string
}

// newTypeGraph builds the graph for root. The transforms are the schema
// generator's structural field-pruning transforms; they run before each type's
// fields are recorded so the graph mirrors the fields the schema actually
// emits (a field the generator deletes must not be documentable). Annotation-
// dependent prunes like output-only removal are not replicated: those fields
// carry upstream docs, so they never surface as undocumented placeholders.
func newTypeGraph(root reflect.Type, transforms ...func(reflect.Type, jsonschema.Schema) jsonschema.Schema) (*typeGraph, error) {
	g := &typeGraph{
		root:   getPath(root),
		fields: map[string][]fieldEdge{},
	}

	var errs []error
	capture := func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
		refPath := getPath(typ)
		if !strings.HasPrefix(refPath, "github.com") {
			return s
		}

		var edges []fieldEdge
		if typ.Kind() == reflect.Struct {
			for _, name := range structFieldOrder(typ, s.Properties) {
				edges = append(edges, fieldEdge{name: name, typ: resolveEdgeType(s.Properties[name])})
			}
			// structFieldOrder only orders names the schema emitted, so a
			// mismatch means its struct walk failed to reach a property —
			// i.e. it diverged from the generator's own field handling.
			if len(edges) != len(s.Properties) {
				errs = append(errs, fmt.Errorf("type graph for %s reached %d of %d schema properties", refPath, len(edges), len(s.Properties)))
			}
		}

		g.fields[refPath] = edges
		return s
	}

	_, err := jsonschema.FromType(root, append(slices.Clone(transforms), capture))
	if err != nil {
		return nil, err
	}
	return g, errors.Join(errs...)
}

// edge returns the property of the given type with the given name.
func (g *typeGraph) edge(typeKey, name string) (fieldEdge, bool) {
	for _, e := range g.fields[typeKey] {
		if e.name == name {
			return e, true
		}
	}
	return fieldEdge{}, false
}

// structFieldOrder returns the names in props ordered by where each field is
// declared in typ, flattening embedded structs breadth-first like
// jsonschema.FromType. Membership in props is authoritative — it already
// reflects every skip rule the generator applies — so reflection here only
// recovers the declaration order the schema's property map loses.
func structFieldOrder(typ reflect.Type, props map[string]*jsonschema.Schema) []string {
	var names []string
	seen := map[string]bool{}
	bfsQueue := list.New()

	for field := range typ.Fields() {
		bfsQueue.PushBack(field)
	}
	for bfsQueue.Len() > 0 {
		front := bfsQueue.Front()
		field := front.Value.(reflect.StructField)
		bfsQueue.Remove(front)

		if field.Anonymous {
			fieldType := field.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}
			for f := range fieldType.Fields() {
				bfsQueue.PushBack(f)
			}
			continue
		}

		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if seen[name] {
			continue
		}
		if _, ok := props[name]; !ok {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

// resolveEdgeType maps a property's $ref to the type path of its annotatable
// element type, unwrapping the slice/ and map/ levels that the schema's type
// paths insert for repeated and keyed fields.
func resolveEdgeType(prop *jsonschema.Schema) string {
	if prop.Reference == nil {
		return ""
	}
	ref := strings.TrimPrefix(*prop.Reference, "#/$defs/")
	for {
		switch {
		case strings.HasPrefix(ref, "slice/"):
			ref = strings.TrimPrefix(ref, "slice/")
		case strings.HasPrefix(ref, "map/"):
			ref = strings.TrimPrefix(ref, "map/")
		case strings.HasPrefix(ref, "github.com"):
			return ref
		default:
			return ""
		}
	}
}
