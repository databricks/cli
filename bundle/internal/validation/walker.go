package main

import (
	"bytes"
	"cmp"
	"fmt"
	"go/format"
	"os"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

// The walkers generated here replace matching every node of the configuration against
// the required and enum field patterns: they descend only along the patterns' paths
// of the typed configuration, with plain field access, and check the fields of each
// node on the way.
//
// The typed configuration cannot tell an unset field from one set to its zero value,
// null or a pure variable reference, and it has no locations. The generated code
// asks the node (bundle/config/validate/walk_node.go) for those.

// tnode is a node of the tree of pattern paths, rooted at config.Root.
type tnode struct {
	typ      reflect.Type // the declared type of the value at this node
	required []string     // required fields (JSON names) of the struct at this node
	enum     []string     // valid values if the value at this node is an enum
	children map[string]*tnode
	sig      string
}

func newTnode(typ reflect.Type) *tnode {
	return &tnode{typ: typ, children: map[string]*tnode{}}
}

// insert adds the path of pattern p to the tree and returns its node.
func (t *tnode) insert(p string) (*tnode, error) {
	pattern, err := structpath.ParsePattern(p)
	if err != nil {
		return nil, err
	}
	n := t
	for _, c := range pattern.AsSlice() {
		var key string
		var next reflect.Type
		elem := deref(n.typ)
		switch {
		case c.DotStar():
			if elem.Kind() != reflect.Map {
				return nil, fmt.Errorf("%s: .* on %s", p, elem)
			}
			key = "*"
			next = elem.Elem()
		case c.BracketStar():
			if elem.Kind() != reflect.Slice {
				return nil, fmt.Errorf("%s: [*] on %s", p, elem)
			}
			key = "[*]"
			next = elem.Elem()
		default:
			k, ok := c.StringKey()
			if !ok || elem.Kind() != reflect.Struct {
				return nil, fmt.Errorf("%s: unsupported component %s on %s", p, c, elem)
			}
			index, ok := fieldIndex(elem, k)
			if !ok {
				return nil, fmt.Errorf("%s: no field %q in %s", p, k, elem)
			}
			key = k
			next = elem.FieldByIndex(index).Type
		}
		child, ok := n.children[key]
		if !ok {
			child = newTnode(next)
			n.children[key] = child
		}
		n = child
	}
	return n, nil
}

// fieldIndex returns the index path of the field with JSON name key in struct st, as
// [convert.ToTyped] resolves it: fields of embedded structs are promoted, and a field
// of an outer struct takes precedence.
func fieldIndex(st reflect.Type, key string) ([]int, bool) {
	queue := [][]int{{}}
	for i := 0; i < len(queue); i++ {
		prefix := queue[i]
		typ := st
		if len(prefix) > 0 {
			typ = deref(st.FieldByIndex(prefix).Type)
		}
		for j := range typ.NumField() {
			sf := typ.Field(j)
			index := append(slices.Clone(prefix), j)
			if sf.Anonymous {
				queue = append(queue, index)
				continue
			}
			if !sf.IsExported() {
				continue
			}
			if structtag.JSONTag(sf.Tag.Get("json")).Name() == key {
				return index, true
			}
		}
	}
	return nil, false
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// signature identifies the code generated for the subtree, so equal subtrees share it.
func (t *tnode) signature() string {
	if t.sig != "" {
		return t.sig
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s|r=%v|e=%v|", t.typ, t.required, t.enum)
	for _, k := range sortedKeys(t.children) {
		fmt.Fprintf(&b, "%s{%s}", k, t.children[k].signature())
	}
	t.sig = b.String()
	return t.sig
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// gen generates the walker for one tree.
type gen struct {
	prefix  string // function name prefix, e.g. "required"
	imports map[string]string
	aliases map[string]bool
	funcs   map[string]string // signature -> function name
	names   map[string]int
	enums   map[string]string // joined values -> variable name
	out     bytes.Buffer
	pending []*tnode
}

func newGen(prefix string) *gen {
	return &gen{
		prefix:  prefix,
		imports: map[string]string{},
		aliases: map[string]bool{"slices": true, "dyn": true},
		funcs:   map[string]string{},
		names:   map[string]int{},
		enums:   map[string]string{},
	}
}

// typeName returns the Go name of the named type t, importing its package.
func (g *gen) typeName(t reflect.Type) string {
	if t.PkgPath() == "" {
		return t.String()
	}
	alias, ok := g.imports[t.PkgPath()]
	if !ok {
		alias = path.Base(t.PkgPath())
		for i := 2; g.aliases[alias]; i++ {
			alias = path.Base(t.PkgPath()) + strconv.Itoa(i)
		}
		g.aliases[alias] = true
		g.imports[t.PkgPath()] = alias
	}
	return alias + "." + t.Name()
}

// funcFor returns the name of the function walking struct node t.
func (g *gen) funcFor(t *tnode) string {
	sig := t.signature()
	if name, ok := g.funcs[sig]; ok {
		return name
	}
	st := deref(t.typ)
	base := g.prefix + capitalize(path.Base(st.PkgPath())) + st.Name()
	g.names[base]++
	name := base
	if g.names[base] > 1 {
		name = base + strconv.Itoa(g.names[base])
	}
	g.funcs[sig] = name
	g.pending = append(g.pending, t)
	return name
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *gen) enumVar(values []string) string {
	key := strings.Join(values, "\x00")
	if name, ok := g.enums[key]; ok {
		return name
	}
	name := fmt.Sprintf("%sValues%d", g.prefix, len(g.enums))
	g.enums[key] = name
	return name
}

// field is the field with JSON name key of the struct v points to.
type field struct {
	key    string
	typ    reflect.Type
	expr   string   // selector of the field
	guards []string // nil checks of the embedded pointers on the way
}

func structField(st reflect.Type, key string) (field, bool) {
	index, ok := fieldIndex(st, key)
	if !ok {
		return field{}, false
	}
	f := field{key: key, expr: "v"}
	t := st
	for i, x := range index {
		sf := deref(t).Field(x)
		f.expr += "." + sf.Name
		if i < len(index)-1 && sf.Type.Kind() == reflect.Pointer {
			f.guards = append(f.guards, f.expr+" != nil")
		}
		t = sf.Type
	}
	f.typ = t
	return f, true
}

func (f field) guarded(cond string) string {
	if len(f.guards) == 0 {
		return cond
	}
	return strings.Join(f.guards, " && ") + " && " + cond
}

// typedSet returns the condition under which the typed value shows the field is set,
// or "" if it cannot (a struct is never nil). A field that is not set by this
// condition can still be set to its zero value, so the node is asked.
func (f field) typedSet() string {
	switch f.typ.Kind() {
	case reflect.String:
		return f.guarded(f.expr + ` != ""`)
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return f.guarded(f.expr + " != nil")
	case reflect.Bool:
		return f.guarded(f.expr)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return f.guarded(f.expr + " != 0")
	default:
		return ""
	}
}

// emitStruct writes the function walking struct node t.
func (g *gen) emitStruct(t *tnode) error {
	st := deref(t.typ)
	if st.Kind() != reflect.Struct {
		return fmt.Errorf("%s is not a struct", st)
	}
	name := g.typeName(st)
	fmt.Fprintf(&g.out, "\nfunc %s(w *walker, n *node, v *%s) {\n", g.funcFor(t), name)
	// The value is null or a pure variable reference.
	fmt.Fprintf(&g.out, "\tif v == nil {\n\t\tv = &%s{}\n\t}\n", name)
	for _, k := range t.required {
		cond := ""
		if f, ok := structField(st, k); ok {
			cond = f.typedSet()
		}
		if cond == "" {
			fmt.Fprintf(&g.out, "\tif !n.has(%q) {\n\t\tw.missing(n, %q)\n\t}\n", k, k)
		} else {
			fmt.Fprintf(&g.out, "\tif !(%s) && !n.has(%q) {\n\t\tw.missing(n, %q)\n\t}\n", cond, k, k)
		}
	}
	for _, k := range sortedKeys(t.children) {
		f, ok := structField(st, k)
		if !ok {
			return fmt.Errorf("no field %q in %s", k, st)
		}
		if err := g.emitField(f, t.children[k]); err != nil {
			return err
		}
	}
	fmt.Fprintf(&g.out, "}\n")
	return nil
}

// emitField writes the code visiting the field f of the struct v (node n).
func (g *gen) emitField(f field, child *tnode) error {
	if len(f.guards) > 0 {
		fmt.Fprintf(&g.out, "\tif %s {\n", strings.Join(f.guards, " && "))
	}
	switch {
	case child.enum != nil:
		if f.typ.Kind() != reflect.String {
			return fmt.Errorf("enum field %q of type %s", f.key, f.typ)
		}
		fmt.Fprintf(&g.out, "\tw.enumField(n, %q, string(%s), %s)\n", f.key, f.expr, g.enumVar(child.enum))
	case f.typ.Kind() == reflect.Pointer:
		fmt.Fprintf(&g.out, "\tif c := n.child(%q); c.exists() {\n\t\t%s(w, &c, %s)\n\t}\n", f.key, g.funcFor(child), f.expr)
	case f.typ.Kind() == reflect.Struct:
		fmt.Fprintf(&g.out, "\tif c := n.child(%q); c.exists() {\n\t\t%s(w, &c, &%s)\n\t}\n", f.key, g.funcFor(child), f.expr)
	case f.typ.Kind() == reflect.Slice || f.typ.Kind() == reflect.Map:
		fmt.Fprintf(&g.out, "\tif len(%s) > 0 {\n\t\tc := n.child(%q)\n", f.expr, f.key)
		if err := g.emitContainer("\t\t", f.expr, f.typ, child, 0); err != nil {
			return err
		}
		fmt.Fprintf(&g.out, "\t}\n")
	default:
		return fmt.Errorf("unsupported field %q of type %s", f.key, f.typ)
	}
	if len(f.guards) > 0 {
		fmt.Fprintf(&g.out, "\t}\n")
	}
	return nil
}

// emitContainer writes the loop over the slice or map expr (node c) and its elements.
func (g *gen) emitContainer(indent, expr string, typ reflect.Type, t *tnode, depth int) error {
	if len(t.children) != 1 {
		return fmt.Errorf("container %s needs exactly one wildcard child", typ)
	}
	var elem *tnode
	for _, c := range t.children {
		elem = c
	}
	if typ.Kind() == reflect.Slice {
		i := fmt.Sprintf("i%d", depth)
		fmt.Fprintf(&g.out, "%sfor %s := range %s {\n%s\te := c.elem(%s)\n", indent, i, expr, indent, i)
		if err := g.emitElem(indent+"\t", expr+"["+i+"]", typ.Elem(), elem, depth); err != nil {
			return err
		}
	} else {
		k, x := fmt.Sprintf("k%d", depth), fmt.Sprintf("x%d", depth)
		key := k
		if typ.Key() != reflect.TypeFor[string]() {
			key = "string(" + k + ")"
		}
		fmt.Fprintf(&g.out, "%sfor %s, %s := range %s {\n%s\te := c.child(%s)\n", indent, k, x, expr, indent, key)
		if err := g.emitElem(indent+"\t", x, typ.Elem(), elem, depth); err != nil {
			return err
		}
	}
	fmt.Fprintf(&g.out, "%s}\n", indent)
	return nil
}

// emitElem writes the code visiting the element expr (node e) of type typ.
func (g *gen) emitElem(indent, expr string, typ reflect.Type, t *tnode, depth int) error {
	switch {
	case t.enum != nil:
		if typ.Kind() != reflect.String {
			return fmt.Errorf("enum element of type %s", typ)
		}
		fmt.Fprintf(&g.out, "%sw.enumElem(&e, string(%s), %s)\n", indent, expr, g.enumVar(t.enum))
	case typ.Kind() == reflect.Pointer:
		fmt.Fprintf(&g.out, "%s%s(w, &e, %s)\n", indent, g.funcFor(t), expr)
	case typ.Kind() == reflect.Struct:
		fmt.Fprintf(&g.out, "%s%s(w, &e, &%s)\n", indent, g.funcFor(t), expr)
	case typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map:
		fmt.Fprintf(&g.out, "%s{\n%s\tc := e\n", indent, indent)
		if err := g.emitContainer(indent+"\t", expr, typ, t, depth+1); err != nil {
			return err
		}
		fmt.Fprintf(&g.out, "%s}\n", indent)
	default:
		return fmt.Errorf("unsupported element type %s", typ)
	}
	return nil
}

// generateWalker writes the walker of the required fields (by parent pattern) and the
// enum values (by field pattern) to file. The entry point is <prefix>ConfigRoot.
func generateWalker(file, prefix, doc string, required, enums map[string][]string) error {
	root := newTnode(reflect.TypeFor[config.Root]())
	for _, p := range sortedKeys(required) {
		n, err := root.insert(p)
		if err != nil {
			return err
		}
		n.required = required[p]
	}
	for _, p := range sortedKeys(enums) {
		n, err := root.insert(p)
		if err != nil {
			return err
		}
		n.enum = enums[p]
	}

	g := newGen(prefix)
	g.funcFor(root)
	for len(g.pending) > 0 {
		t := g.pending[0]
		g.pending = g.pending[1:]
		if err := g.emitStruct(t); err != nil {
			return err
		}
	}

	var src bytes.Buffer
	fmt.Fprintf(&src, "// Code generated by bundle/internal/validation. DO NOT EDIT.\n\n// %s\n\npackage validate\n\nimport (\n", doc)
	paths := make([]string, 0, len(g.imports))
	for p := range g.imports {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		if path.Base(p) == g.imports[p] {
			fmt.Fprintf(&src, "\t%q\n", p)
		} else {
			fmt.Fprintf(&src, "\t%s %q\n", g.imports[p], p)
		}
	}
	fmt.Fprintf(&src, ")\n")
	keys := make([]string, 0, len(g.enums))
	for k := range g.enums {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Or(cmp.Compare(len(g.enums[a]), len(g.enums[b])), cmp.Compare(g.enums[a], g.enums[b]))
	})
	for _, k := range keys {
		var quoted []string
		for value := range strings.SplitSeq(k, "\x00") {
			quoted = append(quoted, strconv.Quote(value))
		}
		fmt.Fprintf(&src, "\nvar %s = []string{%s}\n", g.enums[k], strings.Join(quoted, ", "))
	}
	src.Write(g.out.Bytes())

	formatted, err := format.Source(src.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format %s: %w", file, err)
	}
	return os.WriteFile(file, formatted, 0o644)
}

// withoutTargets drops the patterns under targets and environments: they are merged
// into the root before validation.
func withoutTargets(patterns map[string][]string) map[string][]string {
	out := map[string][]string{}
	for p, v := range patterns {
		top, _, _ := strings.Cut(p, ".")
		if top == "targets" || top == "environments" {
			continue
		}
		out[p] = v
	}
	return out
}

func generateWalkers(dir string) error {
	required, err := extractRequiredFields(reflect.TypeFor[config.Root]())
	if err != nil {
		return fmt.Errorf("failed to extract required fields: %w", err)
	}
	enums, err := extractEnumFields(reflect.TypeFor[config.Root]())
	if err != nil {
		return fmt.Errorf("failed to extract enum fields: %w", err)
	}
	err = generateWalker(dir+"/required_walk.go", "required", "requiredConfigRoot reports the required fields that are not set.", withoutTargets(required), nil)
	if err != nil {
		return err
	}
	return generateWalker(dir+"/enum_walk.go", "enum", "enumConfigRoot reports the enum fields with invalid values.", nil, withoutTargets(enums))
}
