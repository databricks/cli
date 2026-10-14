// sdktypesgen generates JSON-capable mirrors of the sdk-go model types.
//
// The sdk-go model types carry no JSON tags and model oneofs as unexported
// interfaces, so they cannot be used for bundle configuration, deployment state
// or the test server, all of which rely on encoding/json and struct tags. Every
// sdk-go package already contains the JSON representation in its unexported
// wire.go: one tagged struct per model type with oneofs flattened, plus
// converters in both directions. This tool copies wire.go into a CLI package,
// exporting the wire structs under the model type names and the converters as
// <Type>ToSDK / <Type>FromSDK.
//
// Usage: go run ./internal/sdktypesgen <out-dir> <sdk package>...
// e.g. go run ./internal/sdktypesgen libs/sdktypes github.com/databricks/sdk-go/jobs/v2
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const sdkModulePrefix = "github.com/databricks/sdk-go/"

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: sdktypesgen <out-dir> <sdk package>...")
		os.Exit(2)
	}
	outRoot := os.Args[1]
	for _, pkg := range os.Args[2:] {
		if err := generate(outRoot, pkg); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", pkg, err)
			os.Exit(1)
		}
	}
}

func packageDir(pkg string) (string, error) {
	out, err := exec.Command("go", "list", "-f", "{{.Dir}}", pkg).Output()
	if err != nil {
		return "", fmt.Errorf("go list %s: %w", pkg, err)
	}
	return strings.TrimSpace(string(out)), nil
}

type model struct {
	// All exported top-level names declared in model.go.
	decls map[string]bool
	// Struct types declared in model.go.
	structs map[string]*ast.StructType
	// Doc comments of model types and their fields.
	typeDocs  map[string]*ast.CommentGroup
	fieldDocs map[string]map[string]*ast.CommentGroup
	// Non-struct named types (enums, etc.) and constants, re-exported as aliases.
	aliasTypes []string
	consts     []string
	// Oneof interface name -> implementations.
	oneofs map[string][]oneofAlt
}

func parseModel(fset *token.FileSet, path string) (*model, string, error) {
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, "", err
	}
	m := &model{
		decls:     map[string]bool{},
		structs:   map[string]*ast.StructType{},
		typeDocs:  map[string]*ast.CommentGroup{},
		fieldDocs: map[string]map[string]*ast.CommentGroup{},
		oneofs:    map[string][]oneofAlt{},
	}
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			// func (*Parent_Field_Alt) isParent_Field() {}
			if fd.Recv != nil && strings.HasPrefix(fd.Name.Name, "is") && len(fd.Recv.List) == 1 {
				if recv := starIdent(fd.Recv.List[0].Type); recv != "" {
					m.oneofs[fd.Name.Name] = append(m.oneofs[fd.Name.Name], oneofAlt{wrapper: recv})
				}
			}
			continue
		}
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if !s.Name.IsExported() {
					continue
				}
				m.decls[s.Name.Name] = true
				doc := s.Doc
				if doc == nil {
					doc = gd.Doc
				}
				m.typeDocs[s.Name.Name] = doc
				switch t := s.Type.(type) {
				case *ast.StructType:
					m.structs[s.Name.Name] = t
					docs := map[string]*ast.CommentGroup{}
					for _, field := range t.Fields.List {
						for _, n := range field.Names {
							docs[n.Name] = field.Doc
						}
					}
					m.fieldDocs[s.Name.Name] = docs
				case *ast.InterfaceType:
				default:
					if s.TypeParams == nil {
						m.aliasTypes = append(m.aliasTypes, s.Name.Name)
					}
				}
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if n.IsExported() {
						m.decls[n.Name] = true
						if gd.Tok == token.CONST {
							m.consts = append(m.consts, n.Name)
						}
					}
				}
			}
		}
	}
	for iface, alts := range m.oneofs {
		for i, a := range alts {
			st := m.structs[a.wrapper]
			if st == nil || len(st.Fields.List) != 1 || len(st.Fields.List[0].Names) != 1 {
				return nil, "", fmt.Errorf("unexpected oneof wrapper %s", a.wrapper)
			}
			alts[i].field = st.Fields.List[0].Names[0].Name
			alts[i].typ = exprString(st.Fields.List[0].Type)
		}
		m.oneofs[iface] = alts
	}
	return m, f.Name.Name, nil
}

func generate(outRoot, pkg string) error {
	if !strings.HasPrefix(pkg, sdkModulePrefix) {
		return fmt.Errorf("not an sdk-go package")
	}
	dir, err := packageDir(pkg)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	m, pkgName, err := parseModel(fset, filepath.Join(dir, "model.go"))
	if err != nil {
		return err
	}
	wf, err := parser.ParseFile(fset, filepath.Join(dir, "wire.go"), nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// Map wire struct names to model names using the converter signatures:
	//   func fooToWire(v *Foo) (*fooWire, error)
	//   func fooFromWire(w *fooWire) (*Foo, error)
	wireToModel := map[string]string{}
	funcRename := map[string]string{}
	for _, d := range wf.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Type.Params.NumFields() != 1 || fd.Type.Results.NumFields() != 2 {
			continue
		}
		in := starIdent(fd.Type.Params.List[0].Type)
		out := starIdent(fd.Type.Results.List[0].Type)
		name := fd.Name.Name
		switch {
		case strings.HasSuffix(name, "ToWire") && m.structs[in] != nil && strings.HasSuffix(out, "Wire"):
			wireToModel[out] = in
			funcRename[name] = in + "FromSDK"
		case strings.HasSuffix(name, "FromWire") && m.structs[out] != nil && strings.HasSuffix(in, "Wire"):
			wireToModel[in] = out
			funcRename[name] = out + "ToSDK"
		}
	}

	var decls []ast.Decl
	var mirrorNames []string
	for _, d := range wf.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			if d.Tok == token.TYPE {
				ts := d.Specs[0].(*ast.TypeSpec)
				if ts.Name.Name == "wireInt64" {
					continue
				}
				if mn, ok := wireToModel[ts.Name.Name]; ok {
					mirrorNames = append(mirrorNames, mn)
					attachDocs(d, ts, mn, m)
				}
			}
			decls = append(decls, d)
		case *ast.FuncDecl:
			// Converters are synthesized below; the remaining functions are
			// helpers for the wire converters or wireInt64 decoding.
		}
	}

	aliasSet := map[string]bool{}
	for _, a := range m.aliasTypes {
		aliasSet[a] = true
	}
	mirrorFields := map[string][][2]string{}
	for _, d := range decls {
		rewrite(d, wireToModel, funcRename, m.decls, aliasSet)
		gd := d.(*ast.GenDecl)
		if gd.Tok != token.TYPE {
			continue
		}
		ts := gd.Specs[0].(*ast.TypeSpec)
		st, ok := ts.Type.(*ast.StructType)
		if !ok || !m.decls[ts.Name.Name] {
			continue
		}
		var fields [][2]string
		for _, f := range st.Fields.List {
			for _, n := range f.Names {
				fields = append(fields, [2]string{n.Name, exprString(f.Type)})
			}
		}
		mirrorFields[ts.Name.Name] = fields
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by internal/sdktypesgen from %s. DO NOT EDIT.\n\n", pkg)
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)
	imports := map[string]string{"sdk": pkg, "fmt": "fmt", "strings": "strings", "types": "github.com/databricks/sdk-go/core/types"}
	for _, imp := range wf.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := filepath.Base(p)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		imports[name] = p
	}
	buf.WriteString("import (\n")
	var names []string
	for n := range imports {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(&buf, "\t%s %q\n", n, imports[n])
	}
	buf.WriteString(")\n\n")

	if len(m.aliasTypes) > 0 {
		buf.WriteString("type (\n")
		for _, a := range m.aliasTypes {
			fmt.Fprintf(&buf, "\t%s = sdk.%s\n", a, a)
		}
		buf.WriteString(")\n\n")
	}
	if len(m.consts) > 0 {
		buf.WriteString("const (\n")
		for _, c := range m.consts {
			fmt.Fprintf(&buf, "\t%s = sdk.%s\n", c, c)
		}
		buf.WriteString(")\n\n")
	}

	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	for _, d := range decls {
		if err := cfg.Fprint(&buf, fset, d); err != nil {
			return err
		}
		buf.WriteString("\n\n")
	}
	buf.WriteString(generateConverters(m, mirrorFields, m.oneofs))

	src, err := pruneImports(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format: %w\n%s", err, buf.String())
	}

	outDir := filepath.Join(outRoot, strings.TrimPrefix(pkg, sdkModulePrefix))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "types.generated.go"), src, 0o644)
}

func starIdent(e ast.Expr) string {
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// attachDocs copies the model's type and field doc comments onto the mirror.
// Comments are rendered as text so the printer does not need positions from
// model.go.
func attachDocs(gd *ast.GenDecl, ts *ast.TypeSpec, modelName string, m *model) {
	gd.Doc = cloneComment(m.typeDocs[modelName], gd.Pos()-1)
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return
	}
	docs := m.fieldDocs[modelName]
	for _, f := range st.Fields.List {
		if len(f.Names) == 1 {
			f.Doc = cloneComment(docs[f.Names[0].Name], f.Pos()-1)
		}
	}
}

func cloneComment(cg *ast.CommentGroup, pos token.Pos) *ast.CommentGroup {
	if cg == nil {
		return nil
	}
	var list []*ast.Comment
	for _, c := range cg.List {
		list = append(list, &ast.Comment{Slash: pos, Text: c.Text})
	}
	return &ast.CommentGroup{List: list}
}

// rewrite renames wire types and converters to their exported mirror names and
// qualifies references to model declarations with the sdk package.
func rewrite(n ast.Node, wireToModel, funcRename map[string]string, modelDecls, aliasTypes map[string]bool) {
	skip := map[*ast.Ident]bool{}
	inTypeDecl := false
	if gd, ok := n.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
		inTypeDecl = true
	}
	ast.Inspect(n, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.SelectorExpr:
			skip[x.Sel] = true
		case *ast.KeyValueExpr:
			if id, ok := x.Key.(*ast.Ident); ok {
				skip[id] = true
			}
		case *ast.Field:
			for _, id := range x.Names {
				skip[id] = true
			}
		}
		return true
	})
	ast.Inspect(n, func(node ast.Node) bool {
		// Qualify via SelectorExpr replacement on parent nodes is awkward with
		// ast.Inspect, so encode the qualifier into the identifier name; the
		// printer emits it verbatim.
		id, ok := node.(*ast.Ident)
		if !ok || skip[id] {
			return true
		}
		switch {
		case id.Name == "wireInt64":
			id.Name = "int64"
		case wireToModel[id.Name] != "":
			id.Name = wireToModel[id.Name]
		case funcRename[id.Name] != "":
			id.Name = funcRename[id.Name]
		case modelDecls[id.Name]:
			if inTypeDecl && aliasTypes[id.Name] {
				return true
			}
			id.Name = "sdk." + id.Name
		}
		return true
	})
}

func pruneImports(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
		}
		return true
	})
	var specs []ast.Spec
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		for _, s := range gd.Specs {
			is := s.(*ast.ImportSpec)
			name := filepath.Base(strings.Trim(is.Path.Value, `"`))
			if is.Name != nil {
				name = is.Name.Name
			}
			if used[name] {
				specs = append(specs, s)
			}
		}
		gd.Specs = specs
	}
	var out bytes.Buffer
	if err := format.Node(&out, fset, f); err != nil {
		return nil, err
	}
	return format.Source(out.Bytes())
}


