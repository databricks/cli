package main

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/structs/structvar"
	"go.yaml.in/yaml/v3"
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

// descriptorNodes serializes d into the key and value nodes of a mapping, with its
// keys ordered. It returns nothing when d carries no content.
func descriptorNodes(d annotation.Descriptor, style yaml.Style) []*yaml.Node {
	var pairs [][2]*yaml.Node
	for key, v := range structvar.NewView(d, nil, nil).MapItems() {
		pairs = append(pairs, [2]*yaml.Node{{Kind: yaml.ScalarNode, Value: key, Style: style}, valueNode(v, style)})
	}
	// Keys in descriptorKeyOrder come first, the others follow alphabetically.
	rank := func(key string) int {
		if i := slices.Index(descriptorKeyOrder, key); i >= 0 {
			return i
		}
		return len(descriptorKeyOrder)
	}
	slices.SortFunc(pairs, func(a, b [2]*yaml.Node) int {
		return cmp.Or(cmp.Compare(rank(a[0].Value), rank(b[0].Value)), cmp.Compare(a[0].Value, b[0].Value))
	})
	var out []*yaml.Node
	for _, p := range pairs {
		out = append(out, p[:]...)
	}
	return out
}

// valueNode converts a value of a descriptor to a YAML node.
func valueNode(x structvar.View, style yaml.Style) *yaml.Node {
	switch x.Kind() {
	case structvar.KindMap:
		n := &yaml.Node{Kind: yaml.MappingNode, Style: style}
		for k, c := range x.MapItems() {
			n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: k, Style: style}, valueNode(c, style))
		}
		return n
	case structvar.KindSequence:
		n := &yaml.Node{Kind: yaml.SequenceNode, Style: style}
		for _, c := range x.Sequence() {
			n.Content = append(n.Content, valueNode(c, style))
		}
		return n
	case structvar.KindString:
		s, _ := x.AsString()
		// A string that reads as another scalar (bool, number) is quoted to stay a string.
		if isScalarValueInString(s) {
			style = yaml.DoubleQuotedStyle
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: s, Style: style}
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(x.AsAny()), Style: style}
	}
}

func isScalarValueInString(s string) bool {
	if s == "true" || s == "false" || s == "" {
		return true
	}
	if _, err := strconv.ParseInt(s, 0, 64); err == nil {
		return true
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// descriptorEmpty reports whether d carries no documentation.
func descriptorEmpty(d annotation.Descriptor) bool {
	return structvar.NewView(d, nil, nil).Kind() == structvar.KindNil
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
	v, err := structvar.ParseYAML(bytes.NewReader(b))
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
func (l *fileLoader) block(v *yaml.Node, typeKey, where string) error {
	pairs, err := mappingPairs(v, where)
	if err != nil {
		return err
	}

	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i].Value
		child := where + "." + key
		if where == "" {
			child = key
		}

		edge, ok := l.graph.edge(typeKey, key)
		if !ok {
			l.unknown = append(l.unknown, child)
			continue
		}
		err := l.node(pairs[i+1], typeKey, edge, child)
		if err != nil {
			return err
		}
	}
	return nil
}

// node loads one field's node: the inline descriptor for the field, the
// "$type" docs for the type it resolves to, and the "$fields" block of that
// type's fields.
func (l *fileLoader) node(v *yaml.Node, typeKey string, edge fieldEdge, where string) error {
	pairs, err := mappingPairs(v, where)
	if err != nil {
		return err
	}

	var desc []*yaml.Node
	for i := 0; i < len(pairs); i += 2 {
		key, value := pairs[i].Value, pairs[i+1]
		switch {
		case key == fieldsKey && edge.typ != "":
			err := l.block(value, edge.typ, where+"."+fieldsKey)
			if err != nil {
				return err
			}
		case key == typeDocKey && edge.typ != "":
			d, ok, err := l.descriptor(value, where+"."+typeDocKey)
			if err != nil {
				return err
			}
			if ok {
				l.data.SetSelf(edge.typ, d)
			}
		case descriptorKeys[key]:
			desc = append(desc, pairs[i], value)
		default:
			l.unknown = append(l.unknown, where+"."+key)
		}
	}

	if len(desc) > 0 {
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
func (l *fileLoader) descriptor(v *yaml.Node, where string) (annotation.Descriptor, bool, error) {
	if v.Kind != yaml.MappingNode {
		return annotation.Descriptor{}, false, fmt.Errorf("%s: expected a mapping, got %s", where, nodeKind(v))
	}
	var desc []*yaml.Node
	for i := 0; i < len(v.Content); i += 2 {
		key := v.Content[i].Value
		if descriptorKeys[key] {
			desc = append(desc, v.Content[i], v.Content[i+1])
		} else {
			l.unknown = append(l.unknown, where+"."+key)
		}
	}
	if len(desc) == 0 {
		return annotation.Descriptor{}, false, nil
	}
	d, err := toDescriptor(desc, where)
	return d, err == nil, err
}

// toDescriptor converts a mapping of descriptor keys to a typed descriptor.
func toDescriptor(desc []*yaml.Node, where string) (annotation.Descriptor, error) {
	var d annotation.Descriptor
	_, diags, err := structvar.DecodeYAMLNode(where, &yaml.Node{Kind: yaml.MappingNode, Content: desc}, &d, nil)
	if err == nil {
		err = diags.Error()
	}
	if err != nil {
		return annotation.Descriptor{}, fmt.Errorf("%s: %w", where, err)
	}
	return d, nil
}

// mappingPairs returns the alternating key and value nodes of a mapping. A null node has none.
func mappingPairs(v *yaml.Node, where string) ([]*yaml.Node, error) {
	switch {
	case v.ShortTag() == "!!null":
		return nil, nil
	case v.Kind != yaml.MappingNode:
		return nil, fmt.Errorf("%s: expected a mapping, got %s", where, nodeKind(v))
	}
	return v.Content, nil
}

func nodeKind(v *yaml.Node) string {
	switch v.Kind {
	case yaml.MappingNode:
		return "map"
	case yaml.SequenceNode:
		return "sequence"
	default:
		return strings.TrimPrefix(v.ShortTag(), "!!")
	}
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

	// Everything below the top-level keys renders in literal block style, matching
	// the formatting of the previous annotation files.
	root := &yaml.Node{Kind: yaml.MappingNode, Content: s.block(g.root, 0)}
	err := writeYAML(path, root)
	if err != nil {
		return nil, err
	}
	err = prependCommentToFile(path, annotationsFileHeader)
	if err != nil {
		return nil, err
	}
	return s.detached(), nil
}

func writeYAML(path string, node *yaml.Node) error {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	return errors.Join(enc.Encode(node), f.Close())
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

// block renders one type's block of field nodes as the content of a mapping, emitted
// alphabetically, with the keys in keyStyle.
func (s *fileSaver) block(typeKey string, keyStyle yaml.Style) []*yaml.Node {
	var out []*yaml.Node

	edges := slices.Clone(s.graph.fields[typeKey])
	slices.SortFunc(edges, func(a, b fieldEdge) int {
		return strings.Compare(a.name, b.name)
	})
	for _, edge := range edges {
		node := s.node(typeKey, edge)
		if len(node) > 0 {
			out = append(out, &yaml.Node{Kind: yaml.ScalarNode, Value: edge.name, Style: keyStyle}, mapping(node))
		}
	}
	return out
}

func mapping(content []*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Content: content, Style: yaml.LiteralStyle}
}

func literalKey(name string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: name, Style: yaml.LiteralStyle}
}

// node renders one field's node: the inline field descriptor plus, at the
// field's canonical position, the resolved type's "$type" docs and the
// "fields" block of its fields.
func (s *fileSaver) node(typeKey string, edge fieldEdge) []*yaml.Node {
	var out []*yaml.Node

	if d, ok := s.takeField(typeKey, edge.name); ok {
		out = descriptorNodes(d, yaml.LiteralStyle)
	}

	if s.expandAt[edgeKey{typeKey, edge.name}] {
		if doc := descriptorNodes(s.takeSelf(edge.typ), yaml.LiteralStyle); len(doc) > 0 {
			out = append(out, literalKey(typeDocKey), mapping(doc))
		}

		if child := s.block(edge.typ, yaml.LiteralStyle); len(child) > 0 {
			out = append(out, literalKey(fieldsKey), mapping(child))
		}
	}
	return out
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
