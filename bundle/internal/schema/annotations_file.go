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

// fieldsKey nests a type's block of field nodes inside the node of a field
// that resolves to it.
//
// typeDocKey holds the documentation for the type a field's value resolves to
// (for map and sequence fields: each entry). It is applied to the type's
// shared $defs entry, so it shows up at every occurrence of the type; this is
// also where enum values live.
//
// Both are "$"-prefixed so they cannot be mistaken for — or collide with — a
// config field of the same name (e.g. artifacts.*.type), which always appears
// as a bare key inside "$fields".
const (
	fieldsKey  = "$fields"
	typeDocKey = "$type"
)

// lineTypeDoc and lineFields sort the "$type" and "$fields" keys after a
// node's inline descriptor keys, which the saver orders with small line
// numbers (see descriptorKeyOrder).
const (
	lineTypeDoc = 9999
	lineFields  = 10000
)

const annotationsFileHeader = `# This file contains the documentation the CLI owns for the bundle
# configuration JSON schema: docs for fields that do not exist in the upstream
# API spec (.codegen/cli.json), and overrides of upstream docs. Documentation
# for everything else is inherited from cli.json at generation time and must
# not be duplicated here.
#
# The structure mirrors the bundle configuration tree. The "$type" and
# "$fields" keys are structural; every other key is a config field name.
#   - A node documents one field: its inline keys (description,
#     markdown_description, ...) apply to the field itself.
#   - "$type" documents the type the field's value resolves to — for map and
#     sequence fields, each entry. These docs are shared by every occurrence
#     of the type; enum values also live here.
#   - "$fields" holds the nodes of that type's fields (map and sequence levels
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

// descriptorKeyOrder is the leading key order for a serialized descriptor,
// matching the formatting of the previous annotation files. Remaining keys
// follow alphabetically.
var descriptorKeyOrder = []string{"description", "markdown_description", "title", "default", "enum"}

// descriptorToMap serializes d into dst with its keys ordered. It returns the
// nil value (writing nothing) when d carries no content.
func descriptorToMap(d annotation.Descriptor, dst map[string]dyn.Value) (dyn.Value, error) {
	v, err := convert.FromTyped(d, dyn.NilValue)
	if err != nil || v.Kind() == dyn.KindNil {
		return dyn.NilValue, err
	}
	return yamlsaver.ConvertToMapValue(d, yamlsaver.NewOrder(descriptorKeyOrder), []string{}, dst)
}

// descriptorEmpty reports whether d carries no documentation.
func descriptorEmpty(d annotation.Descriptor) bool {
	v, err := convert.FromTyped(d, dyn.NilValue)
	return err == nil && v.Kind() == dyn.KindNil
}

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

// node loads one field's node: the inline descriptor for the field, the
// "$type" docs for the type it resolves to, and the "$fields" block of that
// type's fields.
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
			d, ok, err := l.descriptor(pair.Value, where+"."+typeDocKey)
			if err != nil {
				return err
			}
			if ok {
				l.data.SetSelf(edge.typ, d)
			}
		case descriptorKeys[key]:
			desc.SetLoc(key, nil, pair.Value)
		default:
			l.unknown = append(l.unknown, where+"."+key)
		}
	}

	if desc.Len() > 0 {
		d, err := toDescriptor(desc, where)
		if err != nil {
			return err
		}
		l.data.SetField(typeKey, edge.name, d)
	}
	return nil
}

// descriptor parses a mapping of descriptor keys (the value of a "$type" key).
// Non-descriptor keys are flagged as unknown. The second return is false when
// the mapping carries no descriptor keys.
func (l *fileLoader) descriptor(v dyn.Value, where string) (annotation.Descriptor, bool, error) {
	m, ok := v.AsMap()
	if !ok {
		return annotation.Descriptor{}, false, fmt.Errorf("%s: expected a mapping, got %s", where, v.Kind())
	}
	desc := dyn.NewMapping()
	for _, pair := range m.Pairs() {
		key := pair.Key.MustString()
		if descriptorKeys[key] {
			desc.SetLoc(key, nil, pair.Value)
		} else {
			l.unknown = append(l.unknown, where+"."+key)
		}
	}
	if desc.Len() == 0 {
		return annotation.Descriptor{}, false, nil
	}
	d, err := toDescriptor(desc, where)
	return d, err == nil, err
}

// toDescriptor converts a mapping of descriptor keys to a typed descriptor.
func toDescriptor(desc dyn.Mapping, where string) (annotation.Descriptor, error) {
	var d annotation.Descriptor
	err := convert.ToTyped(&d, dyn.V(desc))
	if err != nil {
		return annotation.Descriptor{}, fmt.Errorf("%s: %w", where, err)
	}
	return d, nil
}

// saveAnnotationsFile writes data to path in the canonical tree layout: a
// depth-first walk over the config type graph in struct declaration order
// expands every type at its first occurrence; keys are emitted alphabetically.
// Entries that no tree position consumed (fields or types that no longer
// exist) are returned as detached and are not written.
func saveAnnotationsFile(path string, data annotation.File, g *typeGraph) ([]string, error) {
	s := &fileSaver{
		graph:        g,
		data:         data,
		visited:      map[string]bool{g.root: true},
		expandAt:     map[edgeKey]bool{},
		consumed:     map[edgeKey]bool{},
		selfConsumed: map[string]bool{},
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
	graph        *typeGraph
	data         annotation.File
	visited      map[string]bool
	expandAt     map[edgeKey]bool
	consumed     map[edgeKey]bool
	selfConsumed map[string]bool
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

// node renders one field's node: the inline field descriptor plus, at the
// field's canonical position, the resolved type's "$type" docs and the
// "fields" block of its fields.
func (s *fileSaver) node(typeKey string, edge fieldEdge) (map[string]dyn.Value, error) {
	out := map[string]dyn.Value{}

	// The inline descriptor keys are written directly into the node, sharing
	// it with the "$type" and "$fields" keys added below.
	if d, ok := s.takeField(typeKey, edge.name); ok {
		if _, err := descriptorToMap(d, out); err != nil {
			return nil, err
		}
	}

	if s.expandAt[edgeKey{typeKey, edge.name}] {
		v, err := descriptorToMap(s.takeSelf(edge.typ), map[string]dyn.Value{})
		if err != nil {
			return nil, err
		}
		if v.Kind() != dyn.KindNil {
			out[typeDocKey] = v.WithLocations([]dyn.Location{{Line: lineTypeDoc}})
		}

		child, err := s.block(edge.typ)
		if err != nil {
			return nil, err
		}
		if len(child) > 0 {
			out[fieldsKey] = dyn.NewValue(child, []dyn.Location{{Line: lineFields}})
		}
	}
	return out, nil
}

// takeField returns the descriptor for a field and marks it consumed for the
// detached-entry report.
func (s *fileSaver) takeField(typeKey, name string) (annotation.Descriptor, bool) {
	d, ok := s.data[typeKey].Fields[name]
	if ok {
		s.consumed[edgeKey{typeKey, name}] = true
	}
	return d, ok
}

// takeSelf returns a type's own descriptor and marks it accounted for, whether
// or not it carries any docs (expanding the type is what consumes it).
func (s *fileSaver) takeSelf(typeKey string) annotation.Descriptor {
	s.selfConsumed[typeKey] = true
	return s.data[typeKey].Self
}

// detached returns the data entries no tree position consumed, sorted. These
// are fields or types that no longer exist in the config.
func (s *fileSaver) detached() []string {
	var out []string
	for typeKey, ta := range s.data {
		if !s.selfConsumed[typeKey] && !descriptorEmpty(ta.Self) {
			out = append(out, typeKey+": (type)")
		}
		for name := range ta.Fields {
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
