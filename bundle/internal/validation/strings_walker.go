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
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structvar"
)

// stringsGen generates the walker visiting every string of the configuration
// (bundle/config/walk/strings_walk.go): one function per struct type that contains
// strings, with plain field access.
type stringsGen struct {
	gen
	hasStrings map[reflect.Type]bool
	names      map[reflect.Type]string
}

// containsStrings reports whether values of type t can contain strings.
func (g *stringsGen) containsStrings(t reflect.Type, visiting map[reflect.Type]bool) bool {
	if v, ok := g.hasStrings[t]; ok {
		return v
	}
	switch t.Kind() {
	case reflect.String, reflect.Interface:
		return true
	case reflect.Pointer, reflect.Slice, reflect.Map:
		return g.containsStrings(t.Elem(), visiting)
	case reflect.Struct:
		if structvar.IsSDKNativeType(t) {
			return false
		}
		if visiting[t] {
			return false
		}
		visiting[t] = true
		defer delete(visiting, t)
		found := false
		info := structvar.GetStructInfo(t)
		for _, f := range info.List {
			if g.containsStrings(t.FieldByIndex(f.Index).Type, visiting) {
				found = true
			}
		}
		if len(visiting) == 1 {
			g.hasStrings[t] = found
		}
		return found
	default:
		return false
	}
}

func (g *stringsGen) funcName(t reflect.Type) string {
	if name, ok := g.names[t]; ok {
		return name
	}
	base := "strings" + capitalize(path.Base(t.PkgPath())) + t.Name()
	g.gen.names[base]++
	name := base
	if n := g.gen.names[base]; n > 1 {
		name = fmt.Sprintf("%s%d", base, n)
	}
	g.names[t] = name
	g.pendingTypes = append(g.pendingTypes, t)
	return name
}

// emitStringsStruct writes the function visiting the strings of struct type t.
func (g *stringsGen) emitStringsStruct(t reflect.Type) {
	fmt.Fprintf(&g.out, "\nfunc %s(c *stringsWalk, n *Node, v *%s) {\n", g.funcName(t), g.typeName(t))
	info := structvar.GetStructInfo(t)
	for _, f := range info.List {
		ft := t.FieldByIndex(f.Index).Type
		if !g.containsStrings(ft, map[reflect.Type]bool{}) {
			continue
		}
		expr, guards := selector("v", t, f.Index)
		if t == reflect.TypeFor[config.Root]() && f.Key == "resources" {
			// See walk.StringsWithoutResources.
			guards = append(guards, "!c.skipResources")
		}
		if len(guards) > 0 {
			fmt.Fprintf(&g.out, "\tif %s {\n", strings.Join(guards, " && "))
		}
		g.emitValue("\t", expr, ft, fmt.Sprintf("n.Child(%q)", f.Key), true)
		if len(guards) > 0 {
			fmt.Fprintf(&g.out, "\t}\n")
		}
	}
	fmt.Fprintf(&g.out, "}\n")
}

// emitValue writes the code visiting the value expr of type t whose node is nodeExpr.
// settable reports whether expr can be assigned to.
func (g *stringsGen) emitValue(indent, expr string, t reflect.Type, nodeExpr string, settable bool) {
	switch t.Kind() {
	case reflect.String:
		conv := ""
		if t.PkgPath() != "" {
			conv = g.typeName(t)
		}
		fmt.Fprintf(&g.out, "%sif s := string(%s); c.match(s) {\n%s\tch := %s\n%s\tif r, ok := c.fn(&ch, s); ok {\n", indent, expr, indent, nodeExpr, indent)
		if settable {
			fmt.Fprintf(&g.out, "%s\t\t%s = %s(r)\n", indent, expr, conv)
		} else {
			fmt.Fprintf(&g.out, "%s\t\t_ = r\n", indent)
		}
		fmt.Fprintf(&g.out, "%s\t}\n%s}\n", indent, indent)
	case reflect.Interface:
		fmt.Fprintf(&g.out, "%sif %s != nil {\n%s\tch := %s\n%s\tc.any(&ch, &%s)\n%s}\n", indent, expr, indent, nodeExpr, indent, expr, indent)
	case reflect.Pointer:
		et := t.Elem()
		if et.Kind() == reflect.Struct {
			fmt.Fprintf(&g.out, "%sif x := %s; x != nil {\n%s\tch := %s\n%s\t%s(c, &ch, x)\n%s}\n", indent, expr, indent, nodeExpr, indent, g.funcName(et), indent)
		} else {
			fmt.Fprintf(&g.out, "%sif x := %s; x != nil {\n", indent, expr)
			g.emitValue(indent+"\t", "*x", et, nodeExpr, true)
			fmt.Fprintf(&g.out, "%s}\n", indent)
		}
	case reflect.Struct:
		fmt.Fprintf(&g.out, "%s{\n%s\tch := %s\n%s\t%s(c, &ch, &%s)\n%s}\n", indent, indent, nodeExpr, indent, g.funcName(t), expr, indent)
	case reflect.Slice:
		fmt.Fprintf(&g.out, "%sif len(%s) > 0 {\n%s\tcn := %s\n%s\tfor i := range %s {\n", indent, expr, indent, nodeExpr, indent, expr)
		g.emitValue(indent+"\t\t", expr+"[i]", t.Elem(), "cn.Elem(i)", true)
		fmt.Fprintf(&g.out, "%s\t}\n%s}\n", indent, indent)
	case reflect.Map:
		et := t.Elem()
		fmt.Fprintf(&g.out, "%sif len(%s) > 0 {\n%s\tcn := %s\n%s\tfor _, k := range sortedKeys(%s) {\n", indent, expr, indent, nodeExpr, indent, expr)
		switch {
		case et.Kind() == reflect.Pointer:
			g.emitValue(indent+"\t\t", expr+"[k]", et, "cn.Child(string(k))", false)
		case et.Kind() == reflect.String:
			// Map values are not addressable: write the replacement back.
			fmt.Fprintf(&g.out, "%s\t\tif s := string(%s[k]); c.match(s) {\n%s\t\t\tch := cn.Child(string(k))\n%s\t\t\tif r, ok := c.fn(&ch, s); ok {\n%s\t\t\t\t%s[k] = %s(r)\n%s\t\t\t}\n%s\t\t}\n",
				indent, expr, indent, indent, indent, expr, g.convName(et), indent, indent)
		default:
			fmt.Fprintf(&g.out, "%s\t\tx := %s[k]\n", indent, expr)
			g.emitValue(indent+"\t\t", "x", et, "cn.Child(string(k))", true)
			fmt.Fprintf(&g.out, "%s\t\t%s[k] = x\n", indent, expr)
		}
		fmt.Fprintf(&g.out, "%s\t}\n%s}\n", indent, indent)
	default:
		panic(fmt.Sprintf("unsupported type %s", t))
	}
}

func (g *stringsGen) convName(t reflect.Type) string {
	if t.PkgPath() == "" {
		return ""
	}
	return g.typeName(t)
}

func generateStringsWalker(file string) error {
	g := &stringsGen{gen: *newGen("strings", false), hasStrings: map[reflect.Type]bool{}, names: map[reflect.Type]string{}}
	g.aliases["strings"] = true
	g.aliases["stringsWalk"] = true
	root := reflect.TypeFor[config.Root]()
	g.funcName(root)
	for len(g.pendingTypes) > 0 {
		t := g.pendingTypes[0]
		g.pendingTypes = g.pendingTypes[1:]
		g.emitStringsStruct(t)
	}

	var src bytes.Buffer
	fmt.Fprintf(&src, "// Code generated by bundle/internal/validation. DO NOT EDIT.\n\npackage walk\n\nimport (\n")
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
	src.Write(g.out.Bytes())
	formatted, err := format.Source(src.Bytes())
	if err != nil {
		_ = os.WriteFile(file+".broken", src.Bytes(), 0o644)
		return fmt.Errorf("failed to format %s: %w", file, err)
	}
	return os.WriteFile(file, formatted, 0o644)
}
