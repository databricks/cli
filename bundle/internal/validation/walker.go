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
	"github.com/databricks/cli/bundle/config/resourcedefaults"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// The walkers generated here replace matching every configuration node against the
// required and enum field patterns: they descend only along the patterns' paths, with
// plain field access, and check each node's fields during a single walk.

// tnode is a node of the tree of pattern paths, rooted at config.Root.
type tnode struct {
	typ      reflect.Type // the declared type of the value at this node
	required []string     // required fields (JSON names) of the struct at this node
	enum     []string     // valid values if the value at this node is an enum
	defaults []defaultKey // fields of the struct at this node that have a default
	children map[string]*tnode
	sig      string
}

// defaultKey is a field (JSON name) with a default, the index of the default.
type defaultKey struct {
	key   string
	index int
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
		case c.DotStar() || c.BracketStar():
			key = "*"
			if c.BracketStar() {
				key = "[*]"
			}
			if elem.Kind() != reflect.Map && elem.Kind() != reflect.Slice {
				return nil, fmt.Errorf("%s: wildcard on %s", p, elem)
			}
			next = elem.Elem()
		default:
			k, ok := c.StringKey()
			if !ok || elem.Kind() != reflect.Struct {
				return nil, fmt.Errorf("%s: unsupported component %s on %s", p, c, elem)
			}
			key = k
			index, ok := structvar.GetStructInfo(elem).Fields[k]
			if !ok {
				return nil, fmt.Errorf("%s: no field %q in %s", p, k, elem)
			}
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
	fmt.Fprintf(&b, "%s|r=%v|e=%v|d=%v|", t.typ, t.required, t.enum, t.defaults)
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

	// pendingTypes are the struct types whose functions are still to be written
	// (strings walker).
	pendingTypes []reflect.Type

	// walkerType is the name of the type of the walker state (first parameter).
	walkerType string

	// refBranches visits nil pointer fields that are present (in ForceSendFields, or
	// written without omitempty) or hold a pure reference as nodes without fields, so
	// their required fields are reported.
	refBranches bool
}

func newGen(prefix string, refBranches bool) *gen {
	return &gen{
		prefix:      prefix,
		refBranches: refBranches,
		imports:     map[string]string{},
		aliases:     map[string]bool{"slices": true, "structvar": true, "config": false},
		funcs:       map[string]string{},
		names:       map[string]int{},
		enums:       map[string]string{},
	}
}

// typeName returns the Go name of t, importing its package.
func (g *gen) typeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + g.typeName(t.Elem())
	case reflect.Slice:
		return "[]" + g.typeName(t.Elem())
	case reflect.Map:
		return "map[" + g.typeName(t.Key()) + "]" + g.typeName(t.Elem())
	default:
	}
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

// selector returns the Go expression selecting the field with the given index path
// from v, and the nil checks of the embedded pointers on the way.
func selector(v string, t reflect.Type, index []int) (string, []string) {
	var guards []string
	var expr strings.Builder
	expr.WriteString(v)
	for i, x := range index {
		f := deref(t).Field(x)
		expr.WriteString("." + f.Name)
		if i < len(index)-1 && f.Type.Kind() == reflect.Pointer {
			guards = append(guards, expr.String()+" != nil")
		}
		t = f.Type
	}
	return expr.String(), guards
}

// field describes the field k of the struct type st as seen from variable v.
type field struct {
	key, goName string
	typ         reflect.Type
	expr        string
	guards      []string
	fsf         string // selector of the governing ForceSendFields, if any
	forceEmpty  bool   // the field has no omitempty
}

func structField(v string, st reflect.Type, k string) field {
	info := structvar.GetStructInfo(st)
	expr, guards := selector(v, st, info.Fields[k])
	f := field{key: k, goName: info.GolangNames[k], typ: st.FieldByIndex(info.Fields[k]).Type, expr: expr, guards: guards, forceEmpty: info.ForceEmpty[k]}
	if index, ok := info.ForceSendFieldsIndex[k]; ok {
		f.fsf, _ = selector(v, st, index)
	}
	return f
}

func guarded(guards []string, cond string) string {
	if len(guards) == 0 {
		return cond
	}
	return "(" + strings.Join(guards, " && ") + " && (" + cond + "))"
}

// forcedCond returns the condition that the field is listed in its ForceSendFields.
func (f field) forcedCond() string {
	if f.fsf == "" {
		return ""
	}
	return fmt.Sprintf(" || walk.Forced(%s, %q)", f.fsf, f.goName)
}

// present returns the condition under which a required field counts as set. Presence
// comes from the value and ForceSendFields only, never from locations: a string is set
// if it is not empty (an empty string never satisfies a required field, even if
// forced); another scalar or a struct if it is not zero or is in ForceSendFields (zero
// is a meaningful bool or number); a pointer, slice, map or interface if it is not nil.
// A value that cannot hold a string can also be replaced by a pure reference.
func (f field) present() string {
	ref := fmt.Sprintf(" || n.IsRef(%q)", f.key)
	switch f.typ.Kind() {
	case reflect.String:
		return guarded(f.guards, f.expr+` != ""`)
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return guarded(f.guards, f.expr+" != nil") + ref
	case reflect.Bool:
		return guarded(f.guards, f.expr+f.forcedCond()) + ref
	case reflect.Struct:
		return guarded(f.guards, fmt.Sprintf("!walk.IsZeroStruct(&%s)", f.expr)+f.forcedCond()) + ref
	default:
		return guarded(f.guards, f.expr+" != 0"+f.forcedCond()) + ref
	}
}

// isSet returns the condition under which the field is set as [structvar.View.Get]
// finds it valid: like [field.present], and a forced empty string or a forced nil
// pointer, slice or map is also set.
func (f field) isSet() string {
	switch f.typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return guarded(f.guards, f.expr+" != nil"+f.forcedCond()) + fmt.Sprintf(" || n.IsRef(%q)", f.key)
	case reflect.String:
		// An explicit, forced "" is kept rather than replaced by a default.
		return guarded(f.guards, f.expr+` != ""`+f.forcedCond())
	default:
		return f.present()
	}
}

// emitStruct writes the function walking struct node t.
func (g *gen) emitStruct(t *tnode) {
	st := deref(t.typ)
	fmt.Fprintf(&g.out, "\nfunc %s(w *%s, n *walk.Node, v *%s) {\n", g.funcFor(t), g.walkerType, g.typeName(st))
	fmt.Fprintf(&g.out, "\tif v == nil {\n")
	for _, k := range t.required {
		fmt.Fprintf(&g.out, "\t\tw.missing(n, %q)\n", k)
	}
	fmt.Fprintf(&g.out, "\t\treturn\n\t}\n")
	for _, k := range t.required {
		f := structField("v", st, k)
		fmt.Fprintf(&g.out, "\tif !(%s) {\n\t\tw.missing(n, %q)\n\t}\n", f.present(), k)
	}
	for _, d := range t.defaults {
		f := structField("v", st, d.key)
		fmt.Fprintf(&g.out, "\tif !(%s) {\n\t\tc := n.Child(%q)\n\t\tw.unset(&c, %d)\n\t}\n", f.isSet(), d.key, d.index)
	}
	for _, k := range sortedKeys(t.children) {
		g.emitField(structField("v", st, k), t.children[k])
	}
	fmt.Fprintf(&g.out, "}\n")
}

// emitField writes the code visiting the field f of the struct v.
func (g *gen) emitField(f field, child *tnode) {
	if len(f.guards) > 0 {
		fmt.Fprintf(&g.out, "\tif %s {\n", strings.Join(f.guards, " && "))
	}
	switch {
	case child.enum != nil:
		fmt.Fprintf(&g.out, "\tif s := string(%s); s != \"\"%s {\n\t\tc := n.Child(%q)\n\t\tw.enumValue(&c, s, %s)\n\t}\n", f.expr, f.forcedCond(), f.key, g.enumVar(child.enum))
	case f.typ.Kind() == reflect.Pointer:
		fn := g.funcFor(child)
		fmt.Fprintf(&g.out, "\tif x := %s; x != nil {\n\t\tc := n.Child(%q)\n\t\t%s(w, &c, x)\n\t}", f.expr, f.key, fn)
		if g.refBranches {
			// A forced nil value (e.g. "run_job_task:") or one replaced by a pure
			// reference is a node without fields.
			cond := strings.TrimPrefix(f.forcedCond()+fmt.Sprintf(" || n.IsRef(%q)", f.key), " || ")
			fmt.Fprintf(&g.out, " else if %s {\n\t\tc := n.Child(%q)\n\t\t%s(w, &c, nil)\n\t}", cond, f.key, fn)
		}
		fmt.Fprintf(&g.out, "\n")
	case f.typ.Kind() == reflect.Struct:
		fmt.Fprintf(&g.out, "\tif x := &%s; !walk.IsZeroStruct(x)%s {\n\t\tc := n.Child(%q)\n\t\t%s(w, &c, x)\n\t}\n", f.expr, f.forcedCond(), f.key, g.funcFor(child))
	case f.typ.Kind() == reflect.Slice || f.typ.Kind() == reflect.Map:
		fmt.Fprintf(&g.out, "\tif len(%s) > 0 {\n\t\tc := n.Child(%q)\n", f.expr, f.key)
		g.emitContainer("\t\t", f.expr, f.typ, child)
		fmt.Fprintf(&g.out, "\t}\n")
	default:
		panic(fmt.Sprintf("unsupported field %s of type %s", f.key, f.typ))
	}
	if len(f.guards) > 0 {
		fmt.Fprintf(&g.out, "\t}\n")
	}
}

// emitContainer writes the loop over the slice or map expr (node c) and its elements.
func (g *gen) emitContainer(indent, expr string, typ reflect.Type, t *tnode) {
	var elem *tnode
	for _, c := range t.children {
		elem = c
	}
	if elem == nil || len(t.children) != 1 {
		panic(fmt.Sprintf("container %s needs exactly one wildcard child", typ))
	}
	if typ.Kind() == reflect.Slice {
		fmt.Fprintf(&g.out, "%sfor i := range %s {\n%s\te := c.Elem(i)\n", indent, expr, indent)
		g.emitElem(indent+"\t", expr+"[i]", typ.Elem(), elem)
	} else {
		fmt.Fprintf(&g.out, "%sfor k, x := range %s {\n%s\te := c.Child(k)\n", indent, expr, indent)
		g.emitElem(indent+"\t", "x", typ.Elem(), elem)
	}
	fmt.Fprintf(&g.out, "%s}\n", indent)
}

// emitElem writes the code visiting the element expr (node e) of type typ.
func (g *gen) emitElem(indent, expr string, typ reflect.Type, t *tnode) {
	switch {
	case t.enum != nil:
		fmt.Fprintf(&g.out, "%sw.enumValue(&e, string(%s), %s)\n", indent, expr, g.enumVar(t.enum))
	case typ.Kind() == reflect.Pointer:
		fmt.Fprintf(&g.out, "%s%s(w, &e, %s)\n", indent, g.funcFor(t), expr)
	case typ.Kind() == reflect.Struct:
		fmt.Fprintf(&g.out, "%s%s(w, &e, &%s)\n", indent, g.funcFor(t), expr)
	case typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map:
		fmt.Fprintf(&g.out, "%s{\n%s\tc := e\n", indent, indent)
		g.emitContainer(indent+"\t", expr, typ, t)
		fmt.Fprintf(&g.out, "%s}\n", indent)
	default:
		panic(fmt.Sprintf("unsupported element type %s", typ))
	}
}

// generateWalker writes the walker for the patterns to file; fn is the name of the
// entry point.
// walkerSpec describes a generated walker.
type walkerSpec struct {
	file, pkg, prefix, doc string
	walkerType             string
	required, enums        map[string][]string
	defaults               []string // patterns of fields with a default, in order
}

func generateWalker(spec walkerSpec) error {
	file, prefix, doc := spec.file, spec.prefix, spec.doc
	root := newTnode(reflect.TypeFor[config.Root]())
	for _, p := range sortedKeys(spec.required) {
		n, err := root.insert(p)
		if err != nil {
			return err
		}
		n.required = spec.required[p]
	}
	for _, p := range sortedKeys(spec.enums) {
		n, err := root.insert(p)
		if err != nil {
			return err
		}
		n.enum = spec.enums[p]
	}
	for i, p := range spec.defaults {
		pattern, err := structpath.ParsePattern(p)
		if err != nil {
			return err
		}
		key, ok := pattern.StringKey()
		if !ok {
			return fmt.Errorf("default %s: the last component must be a key", p)
		}
		n, err := root.insert(pattern.Parent().String())
		if err != nil {
			return err
		}
		n.defaults = append(n.defaults, defaultKey{key, i})
	}

	g := newGen(prefix, spec.required != nil)
	g.walkerType = spec.walkerType
	g.funcFor(root)
	for len(g.pending) > 0 {
		t := g.pending[0]
		g.pending = g.pending[1:]
		g.emitStruct(t)
	}

	var src bytes.Buffer
	fmt.Fprintf(&src, "// Code generated by bundle/internal/validation. DO NOT EDIT.\n\n// %s\n\npackage %s\n\nimport (\n\t\"github.com/databricks/cli/bundle/config/walk\"\n", doc, spec.pkg)
	paths := make([]string, 0, len(g.imports))
	for p := range g.imports {
		paths = append(paths, p)
	}
	slices.SortFunc(paths, cmp.Compare)
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
	slices.SortFunc(keys, func(a, b string) int { return cmp.Compare(g.enums[a], g.enums[b]) })
	for _, k := range keys {
		values := strings.Split(k, "\x00")
		fmt.Fprintf(&src, "\nvar %s = %s\n", g.enums[k], "[]string"+formatSliceToString(values))
	}
	src.Write(g.out.Bytes())

	formatted, err := format.Source(src.Bytes())
	if err != nil {
		_ = os.WriteFile(file+".broken", src.Bytes(), 0o644)
		return fmt.Errorf("failed to format %s: %w", file, err)
	}
	return os.WriteFile(file, formatted, 0o644)
}

// patternMaps returns the required fields by parent pattern and the enum values by
// field pattern, without targets and environments (they are merged into the root).
func patternMaps() (map[string][]string, map[string][]string, error) {
	required, err := extractRequiredFields(reflect.TypeFor[config.Root]())
	if err != nil {
		return nil, nil, err
	}
	enums, err := extractEnumFields(reflect.TypeFor[config.Root]())
	if err != nil {
		return nil, nil, err
	}
	skip := func(p string) bool {
		k := getGroupingKey(p)
		return k == "targets" || k == "environments"
	}
	r := map[string][]string{}
	for p, fields := range required {
		if !skip(p) {
			r[p] = fields
		}
	}
	e := map[string][]string{}
	for p, values := range enums {
		if !skip(p) {
			e[p] = values
		}
	}
	return r, e, nil
}

func generateWalkers(dir string) error {
	required, enums, err := patternMaps()
	if err != nil {
		return err
	}
	err = generateWalker(walkerSpec{
		file: dir + "/required_walk.go", pkg: "validate", prefix: "required", walkerType: "walker",
		doc: "requiredWalk reports required fields that are not set.", required: required,
	})
	if err != nil {
		return err
	}
	err = generateWalker(walkerSpec{
		file: dir + "/enum_walk.go", pkg: "validate", prefix: "enum", walkerType: "walker",
		doc: "enumWalk reports enum fields with invalid values.", enums: enums,
	})
	if err != nil {
		return err
	}
	err = generateWalker(walkerSpec{
		file: "bundle/config/mutator/resourcemutator/defaults_walk.go", pkg: "resourcemutator", prefix: "defaults", walkerType: "defaultsWalker",
		doc: "defaultsWalk calls unset for the fields with a default (resourcedefaults.Defaults) that are not set.", defaults: resourcedefaults.Patterns(),
	})
	if err != nil {
		return err
	}
	err = generateStringsWalker("bundle/config/walk/strings_walk.go")
	if err != nil {
		return err
	}
	return generateResourcesIterate("bundle/config/resources_iterate.go")
}
