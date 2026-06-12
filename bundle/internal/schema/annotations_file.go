package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"

	yaml3 "go.yaml.in/yaml/v3"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
)

// fieldsKey nests a type's block inside the node of a field that resolves to
// it. It cannot clash with descriptor keys, and config fields that happen to
// be named "fields" simply appear inside it like any other field.
const fieldsKey = "fields"

// typeDocKey holds the documentation for the type a field's value resolves to
// (for map and sequence fields: each entry). It is applied to the type's
// shared $defs entry, so it shows up at every occurrence of the type; this is
// also where enum values live. Config fields named "type" do not clash: they
// appear inside "fields" like any other field.
const typeDocKey = "type"

const annotationsFileHeader = `# This file contains the documentation the CLI owns for the bundle
# configuration JSON schema: docs for fields that do not exist in the upstream
# API spec (.codegen/cli.json), and overrides of upstream docs. Documentation
# for everything else is inherited from cli.json at generation time and must
# not be duplicated here.
#
# The structure mirrors the bundle configuration tree:
#   - A node documents one field: its inline keys (description,
#     markdown_description, ...) apply to the field itself.
#   - "type" documents the type the field's value resolves to — for map and
#     sequence fields, each entry. These docs are shared by every occurrence
#     of the type; enum values also live here.
#   - "fields" holds the nodes of that type's fields (map and sequence levels
#     are unwrapped implicitly).
#   - Each type is expanded exactly once, at its first occurrence; fields of
#     types that occur again later (for example everything under "targets")
#     are documented at that first occurrence.
#   - "description: PLACEHOLDER" marks fields that have no documentation yet.
#
# Running "task generate-schema" keeps this file in sync with the
# configuration structure: it adds placeholders for new undocumented fields
# and drops entries for fields that no longer exist.
`

// descriptorKeys is the set of YAML keys a node carries inline (the JSON tags
// of annotation.Descriptor).
var descriptorKeys = func() map[string]bool {
	keys := map[string]bool{}
	for field := range reflect.TypeFor[annotation.Descriptor]().Fields() {
		keys[strings.Split(field.Tag.Get("json"), ",")[0]] = true
	}
	return keys
}()

// loadAnnotationsFile reads the tree-format annotations file and flattens it
// into per-type annotations. Tree positions that do not resolve to a type or
// field in the config (stale entries, typos) are returned in unknown; they
// are not loaded, so the next save drops them.
func loadAnnotationsFile(path string, g *typeGraph) (annotation.File, []string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	v, err := yamlloader.LoadYAML(path, bytes.NewBuffer(b))
	if err != nil {
		return nil, nil, err
	}

	l := &fileLoader{graph: g, data: annotation.File{}}
	err = l.block(v, g.root, "")
	if err != nil {
		return nil, nil, err
	}
	return l.data, l.unknown, nil
}

type fileLoader struct {
	graph   *typeGraph
	data    annotation.File
	unknown []string
}

// block loads one type's block of field nodes.
func (l *fileLoader) block(v dyn.Value, typeKey, where string) error {
	if v.Kind() == dyn.KindNil {
		return nil
	}
	m, ok := v.AsMap()
	if !ok {
		return fmt.Errorf("%s: expected a mapping, got %s", where, v.Kind())
	}

	for _, pair := range m.Pairs() {
		key := pair.Key.MustString()
		child := where + "." + key
		if where == "" {
			child = key
		}

		edge, ok := l.graph.edge(typeKey, key)
		if !ok {
			l.unknown = append(l.unknown, child)
			continue
		}
		err := l.node(pair.Value, typeKey, edge, child)
		if err != nil {
			return err
		}
	}
	return nil
}

// node loads one field's node: inline descriptor keys, plus the optional
// "type" docs and "fields" block for the type the field resolves to.
func (l *fileLoader) node(v dyn.Value, typeKey string, edge fieldEdge, where string) error {
	if v.Kind() == dyn.KindNil {
		return nil
	}
	m, ok := v.AsMap()
	if !ok {
		return fmt.Errorf("%s: expected a mapping, got %s", where, v.Kind())
	}

	desc := dyn.NewMapping()
	for _, pair := range m.Pairs() {
		key := pair.Key.MustString()
		switch {
		case key == fieldsKey && edge.typ != "":
			err := l.block(pair.Value, edge.typ, where+"."+fieldsKey)
			if err != nil {
				return err
			}
		case key == typeDocKey && edge.typ != "":
			// Type docs are stored under the type's RootTypeKey entry. The
			// synthetic edge has no element type, so nested "type" or
			// "fields" keys inside the docs are flagged as unknown.
			err := l.node(pair.Value, edge.typ, fieldEdge{name: RootTypeKey}, where+"."+typeDocKey)
			if err != nil {
				return err
			}
		case descriptorKeys[key]:
			desc.SetLoc(key, nil, pair.Value)
		default:
			l.unknown = append(l.unknown, where+"."+key)
		}
	}

	if desc.Len() == 0 {
		return nil
	}
	var d annotation.Descriptor
	err := convert.ToTyped(&d, dyn.V(desc))
	if err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}
	if l.data[typeKey] == nil {
		l.data[typeKey] = map[string]annotation.Descriptor{}
	}
	l.data[typeKey][edge.name] = d
	return nil
}

// saveAnnotationsFile writes data to path in the canonical tree layout: a
// depth-first walk over the config type graph in struct declaration order
// expands every type at its first occurrence; keys are emitted alphabetically.
// Entries that no tree position consumed (fields or types that no longer
// exist) are returned as detached and are not written.
func saveAnnotationsFile(path string, data annotation.File, g *typeGraph) ([]string, error) {
	s := &fileSaver{
		graph:    g,
		data:     data,
		visited:  map[string]bool{g.root: true},
		expandAt: map[edgeKey]bool{},
		consumed: map[edgeKey]bool{},
	}
	s.assignCanonical(g.root)

	root, err := s.block(g.root)
	if err != nil {
		return nil, err
	}

	// Style every top-level key so all nested scalars render in literal block
	// style, matching the formatting of the previous annotation files.
	style := map[string]yaml3.Style{}
	for k := range root {
		style[k] = yaml3.LiteralStyle
	}
	err = yamlsaver.NewSaverWithStyle(style).SaveAsYAML(root, path, true)
	if err != nil {
		return nil, err
	}
	err = prependCommentToFile(path, annotationsFileHeader)
	if err != nil {
		return nil, err
	}
	return s.detached(), nil
}

type edgeKey struct {
	typ  string
	name string
}

type fileSaver struct {
	graph    *typeGraph
	data     annotation.File
	visited  map[string]bool
	expandAt map[edgeKey]bool
	consumed map[edgeKey]bool
}

// assignCanonical walks the type graph depth-first in struct declaration
// order and records, for every type, the field edge at which it is expanded.
func (s *fileSaver) assignCanonical(typeKey string) {
	for _, edge := range s.graph.fields[typeKey] {
		if edge.typ == "" || s.visited[edge.typ] {
			continue
		}
		s.visited[edge.typ] = true
		s.expandAt[edgeKey{typeKey, edge.name}] = true
		s.assignCanonical(edge.typ)
	}
}

// block renders one type's block of field nodes, emitted alphabetically.
// Lines in the value locations encode the output order for the YAML saver.
func (s *fileSaver) block(typeKey string) (map[string]dyn.Value, error) {
	out := map[string]dyn.Value{}
	line := 0

	edges := slices.Clone(s.graph.fields[typeKey])
	slices.SortFunc(edges, func(a, b fieldEdge) int {
		return strings.Compare(a.name, b.name)
	})
	for _, edge := range edges {
		node, err := s.node(typeKey, edge)
		if err != nil {
			return nil, err
		}
		if len(node) > 0 {
			out[edge.name] = dyn.NewValue(node, []dyn.Location{{Line: line}})
			line++
		}
	}
	return out, nil
}

// node renders one field's node: the inline descriptor plus, at the field's
// canonical position, the "fields" block of the type it resolves to.
func (s *fileSaver) node(typeKey string, edge fieldEdge) (map[string]dyn.Value, error) {
	out := map[string]dyn.Value{}

	if d, ok := s.take(typeKey, edge.name); ok {
		v, err := convert.FromTyped(d, dyn.NilValue)
		if err != nil {
			return nil, err
		}
		if v.Kind() != dyn.KindNil {
			// Order the descriptor keys with description first, like the
			// previous annotation files.
			order := yamlsaver.NewOrder([]string{"description", "markdown_description", "title", "default", "enum"})
			_, err = yamlsaver.ConvertToMapValue(d, order, []string{}, out)
			if err != nil {
				return nil, err
			}
		}
	}

	if s.expandAt[edgeKey{typeKey, edge.name}] {
		// High line numbers sort the type docs and the fields block after
		// the inline descriptor keys.
		if d, ok := s.take(edge.typ, RootTypeKey); ok {
			v, err := descriptorValue(d, 9999)
			if err != nil {
				return nil, err
			}
			if v.Kind() != dyn.KindNil {
				out[typeDocKey] = v
			}
		}
		child, err := s.block(edge.typ)
		if err != nil {
			return nil, err
		}
		if len(child) > 0 {
			out[fieldsKey] = dyn.NewValue(child, []dyn.Location{{Line: 10000}})
		}
	}
	return out, nil
}

// descriptorValue converts a type docs descriptor, ordering its keys like the
// inline descriptors and placing it at the given line in its node.
func descriptorValue(d annotation.Descriptor, line int) (dyn.Value, error) {
	v, err := convert.FromTyped(d, dyn.NilValue)
	if err != nil || v.Kind() == dyn.KindNil {
		return dyn.NilValue, err
	}
	order := yamlsaver.NewOrder([]string{"description", "markdown_description", "title", "default", "enum"})
	v, err = yamlsaver.ConvertToMapValue(d, order, []string{}, map[string]dyn.Value{})
	if err != nil {
		return dyn.NilValue, err
	}
	return v.WithLocations([]dyn.Location{{Line: line}}), nil
}

// take returns the descriptor for the given type and field and marks it
// consumed for the detached-entry report.
func (s *fileSaver) take(typeKey, name string) (annotation.Descriptor, bool) {
	d, ok := s.data[typeKey][name]
	if ok {
		s.consumed[edgeKey{typeKey, name}] = true
	}
	return d, ok
}

// detached returns the data entries no tree position consumed, sorted.
func (s *fileSaver) detached() []string {
	var out []string
	for typeKey, fields := range s.data {
		for name := range fields {
			if !s.consumed[edgeKey{typeKey, name}] {
				out = append(out, typeKey+": "+name)
			}
		}
	}
	slices.Sort(out)
	return out
}

func prependCommentToFile(outputPath, comment string) error {
	b, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(comment)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}
