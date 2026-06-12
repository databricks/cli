package main

import (
	"container/list"
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

func newTypeGraph(root reflect.Type) (*typeGraph, error) {
	g := &typeGraph{
		root:   getPath(root),
		fields: map[string][]fieldEdge{},
	}

	var ferr error
	_, err := jsonschema.FromType(root, []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
			refPath := getPath(typ)
			if !strings.HasPrefix(refPath, "github.com") {
				return s
			}

			var edges []fieldEdge
			if typ.Kind() == reflect.Struct {
				for _, name := range structFieldNames(typ) {
					prop, ok := s.Properties[name]
					if !ok {
						ferr = fmt.Errorf("field order for %s diverged from the generated schema: %s not in schema", refPath, name)
						return s
					}
					edges = append(edges, fieldEdge{name: name, typ: resolveEdgeType(prop)})
				}
				if len(edges) != len(s.Properties) {
					ferr = fmt.Errorf("field order for %s diverged from the generated schema: %d fields, %d properties", refPath, len(edges), len(s.Properties))
					return s
				}
			}

			g.fields[refPath] = edges
			return s
		},
	})
	if err != nil {
		return nil, err
	}
	return g, ferr
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

// structFieldNames returns the JSON property names of typ in struct
// declaration order, flattening embedded structs breadth-first with the same
// tag rules as jsonschema.FromType. newTypeGraph checks the result against the
// properties FromType actually emitted, so the two cannot silently diverge.
func structFieldNames(typ reflect.Type) []string {
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

		bundleTags := strings.Split(field.Tag.Get("bundle"), ",")
		if slices.Contains(bundleTags, "readonly") || slices.Contains(bundleTags, "internal") {
			continue
		}

		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" || !field.IsExported() || seen[name] {
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
