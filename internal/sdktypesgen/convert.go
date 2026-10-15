package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"sort"
	"strings"
)

// oneofAlt is one implementation of a oneof interface in the sdk package:
//
//	type Parent_Field_Alt struct{ Alt T }
//	func (*Parent_Field_Alt) isParent_Field() {}
type oneofAlt struct {
	wrapper  string
	field    string
	typ      string
	wireType string
}

func nillable(t string) bool {
	return strings.HasPrefix(t, "*") || strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[") || t == "json.RawMessage" || t == "any"
}

func (g *converterGen) isEnum(t string) bool {
	return slices.Contains(g.m.aliasTypes, t)
}

func (g *converterGen) isSet(expr, wireType string) string {
	if g.isEnum(wireType) {
		return expr + ` != ""`
	}
	return expr + " != nil"
}

// converterGen synthesizes <T>ToSDK and <T>FromSDK for every mirror type.
// sdk-go's wire.go only has the directions sdk-go itself needs (requests are
// only encoded, responses only decoded), while the CLI needs both for every
// type, so converters are derived from the model and mirror struct shapes.
type converterGen struct {
	m *model
	// mirror struct name -> ordered fields (name, type) of the mirror struct.
	mirrorFields map[string][][2]string
	// oneof interface name -> implementations.
	oneofs map[string][]oneofAlt
	// Whether a helper is needed in the output.
	// mirror struct name -> Go field name -> JSON name.
	jsonNames     map[string]map[string]string
	curJSON       map[string]string
	needSlice     bool
	needFieldMask bool
	b             strings.Builder
	tmp           int
}

func exprString(e ast.Expr) string { return types.ExprString(e) }

func (g *converterGen) isMirror(name string) bool {
	_, ok := g.mirrorFields[name]
	return ok
}

func (g *converterGen) p(format string, args ...any) {
	fmt.Fprintf(&g.b, format, args...)
	g.b.WriteString("\n")
}

func (g *converterGen) newVar(prefix string) string {
	g.tmp++
	return fmt.Sprintf("%s%d", prefix, g.tmp)
}

// sdkType renders a model type expression as seen from the mirror package.
func (g *converterGen) sdkType(t string) string {
	prefix := ""
	var prefixSb78 strings.Builder
	for strings.HasPrefix(t, "*") || strings.HasPrefix(t, "[]") {
		if t[0] == '*' {
			prefixSb78.WriteString("*")
			t = t[1:]
		} else {
			prefixSb78.WriteString("[]")
			t = t[2:]
		}
	}
	prefix += prefixSb78.String()
	if g.m.decls[t] {
		return prefix + "sdk." + t
	}
	return prefix + t
}

// conv emits statements converting src (of model type mt on the model side)
// in the given direction and returns the expression holding the result.
// errCtx names the field in error messages.
func (g *converterGen) conv(src, mt, dir, errCtx string) string {
	errCheck := func(v string) {
		g.p("if err != nil {\nreturn nil, fmt.Errorf(\"%%s: %%w\", %q, err)\n}", errCtx)
		_ = v
	}
	switch {
	case strings.HasPrefix(mt, "*types.FieldMask["):
		g.needFieldMask = true
		if dir == "FromSDK" {
			return fmt.Sprintf("fieldMaskToString(%s)", src)
		}
		elem := strings.TrimSuffix(strings.TrimPrefix(mt, "*types.FieldMask["), "]")
		v := g.newVar("mask")
		g.p("%s, err := fieldMaskFromString[%s](%s)", v, g.sdkType(elem), src)
		errCheck(v)
		return v
	case strings.HasPrefix(mt, "*") && g.isMirror(mt[1:]):
		v := g.newVar("conv")
		g.p("%s, err := %s%s(%s)", v, mt[1:], dir, src)
		errCheck(v)
		return v
	case g.isMirror(mt):
		v := g.newVar("conv")
		g.p("%s, err := %s%s(&%s)", v, mt, dir, src)
		errCheck(v)
		return "*" + v
	case strings.HasPrefix(mt, "[]") && g.isMirror(mt[2:]):
		g.needSlice = true
		v := g.newVar("conv")
		g.p("%s, err := convertSlice(%s, %s%s)", v, src, mt[2:], dir)
		errCheck(v)
		return v
	default:
		return src
	}
}

func (g *converterGen) generate(name string) {
	mst := g.m.structs[name]
	g.curJSON = g.jsonNames[name]
	wireFields := map[string]string{}
	for _, f := range g.mirrorFields[name] {
		wireFields[f[0]] = f[1]
	}
	for _, dir := range []string{"ToSDK", "FromSDK"} {
		in, out := name, "sdk."+name
		inVar := "w"
		if dir == "FromSDK" {
			in, out = out, in
			inVar = "v"
		}
		if dir == "ToSDK" {
			g.p("// %sToSDK converts the mirror to the sdk-go type.", name)
		} else {
			g.p("// %sFromSDK converts the sdk-go type to the mirror.", name)
		}
		g.p("func %s%s(%s *%s) (*%s, error) {", name, dir, inVar, in, out)
		g.p("if %s == nil {\nreturn nil, nil\n}", inVar)
		g.p("out := &%s{}", out)
		for _, field := range mst.Fields.List {
			if len(field.Names) != 1 {
				continue
			}
			fname := field.Names[0].Name
			ft := exprString(field.Type)
			ctx := name + "." + fname
			if alts, ok := g.oneofs[ft]; ok {
				g.oneof(dir, inVar, fname, ctx, alts, wireFields)
				continue
			}
			if _, ok := wireFields[fname]; !ok {
				continue
			}
			val := g.conv(inVar+"."+fname, ft, dir, ctx)
			g.p("out.%s = %s", fname, val)
		}
		g.p("return out, nil\n}\n")
	}
}

func (g *converterGen) oneof(dir, inVar, fname, ctx string, alts []oneofAlt, wireFields map[string]string) {
	var present []oneofAlt
	for _, a := range alts {
		wt, ok := wireFields[a.field]
		if !ok {
			continue
		}
		if !nillable(wt) && !g.isEnum(wt) {
			panic(fmt.Sprintf("%s: oneof member %s has non-nillable mirror type %s", ctx, a.field, wt))
		}
		a.wireType = wt
		present = append(present, a)
	}
	if len(present) == 0 {
		return
	}
	if dir == "ToSDK" {
		g.p("{\nvar members []string")
		for _, a := range present {
			g.p("if %s {\nmembers = append(members, %q)\n}", g.isSet("w."+a.field, a.wireType), g.curJSON[a.field])
		}
		g.p("if len(members) > 1 {\nreturn nil, fmt.Errorf(\"only one of %%s can be set\", strings.Join(members, \", \"))\n}\n}")
		g.p("switch {")
		for _, a := range present {
			g.p("case %s:", g.isSet("w."+a.field, a.wireType))
			var val string
			switch {
			case g.isEnum(a.wireType):
				val = "w." + a.field
			case nillable(a.typ) || g.isMirror(a.typ):
				src := "w." + a.field
				if g.isMirror(a.typ) {
					// Value-typed message member: the mirror holds a pointer.
					v := g.conv(src, "*"+a.typ, dir, ctx)
					val = "*" + v
				} else {
					val = g.conv(src, a.typ, dir, ctx)
				}
			default:
				val = "*w." + a.field
			}
			g.p("out.%s = &sdk.%s{%s: %s}", fname, a.wrapper, a.field, val)
		}
		g.p("}")
		return
	}
	g.p("switch value := v.%s.(type) {", fname)
	for _, a := range present {
		g.p("case *sdk.%s:", a.wrapper)
		g.p("if value != nil {")
		var val string
		switch {
		case g.isEnum(a.wireType):
			val = "value." + a.field
		case g.isMirror(a.typ):
			val = g.conv("&value."+a.field, "*"+a.typ, dir, ctx)
		case nillable(a.typ):
			val = g.conv("value."+a.field, a.typ, dir, ctx)
		default:
			val = "new(value." + a.field + ")"
		}
		g.p("out.%s = %s", a.field, val)
		g.p("}")
	}
	g.p("}")
}

func (g *converterGen) helpers() string {
	var b strings.Builder
	if g.needSlice {
		b.WriteString(`
func convertSlice[T, W any](s []T, conv func(*T) (*W, error)) ([]W, error) {
	if s == nil {
		return nil, nil
	}
	out := make([]W, len(s))
	for i := range s {
		converted, err := conv(&s[i])
		if err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
		out[i] = *converted
	}
	return out, nil
}
`)
	}
	if g.needFieldMask {
		b.WriteString(`
func fieldMaskToString[T any](mask *types.FieldMask[T]) *string {
	if mask == nil {
		return nil
	}
	value := mask.String()
	return &value
}

func fieldMaskFromString[T any](s *string) (*types.FieldMask[T], error) {
	if s == nil {
		return nil, nil
	}
	var paths []string
	if *s != "" {
		paths = strings.Split(*s, ",")
	}
	return types.NewFieldMask[T](paths...)
}
`)
	}
	return b.String()
}

func generateConverters(m *model, mirrorFields map[string][][2]string, jsonNames map[string]map[string]string, oneofs map[string][]oneofAlt) string {
	g := &converterGen{m: m, mirrorFields: mirrorFields, jsonNames: jsonNames, oneofs: oneofs}
	var names []string
	for n := range mirrorFields {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		g.generate(n)
	}
	return g.b.String() + g.helpers()
}
