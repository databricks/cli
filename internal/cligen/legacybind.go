package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// legacyPackages are hand-written CLI clients for APIs that sdk-go does not
// have. They keep the previous SDK's method and type names (value fields,
// enums implementing pflag.Value), so commands bind to them the way they
// bound to the previous SDK.
var legacyPackages = []struct {
	service, dir, alias, clientField string
}{
	{"Workspace", "libs/workspaceapi", "workspaceapi", "Workspace"},
	{"QualityMonitors", "libs/qualitymonitors", "qualitymonitors", "QualityMonitors"},
	{"BundleDeployments", "libs/bundledeployments", "bundledeployments", "BundleDeployments"},
}

type legacyPkg struct {
	alias, importPath, clientField string
	methods                        map[string]*legacyMethod
	structs                        map[string]*MirrorStruct
	enums                          map[string]bool
}

type legacyMethod struct {
	name, request string
	// kind is "resp", "err" or "listing".
	kind string
}

func loadLegacy(repoRoot string) (map[string]*legacyPkg, error) {
	out := map[string]*legacyPkg{}
	for _, lp := range legacyPackages {
		p := &legacyPkg{
			alias:       lp.alias,
			importPath:  "github.com/databricks/cli/" + lp.dir,
			clientField: lp.clientField,
			methods:     map[string]*legacyMethod{},
			structs:     map[string]*MirrorStruct{},
			enums:       map[string]bool{},
		}
		files, err := filepath.Glob(filepath.Join(repoRoot, lp.dir, "*.go"))
		if err != nil {
			return nil, err
		}
		for _, fn := range files {
			if strings.HasSuffix(fn, "_test.go") {
				continue
			}
			src, err := os.ReadFile(fn)
			if err != nil {
				return nil, err
			}
			f, err := parser.ParseFile(token.NewFileSet(), fn, src, 0)
			if err != nil {
				return nil, err
			}
			for _, d := range f.Decls {
				switch d := d.(type) {
				case *ast.FuncDecl:
					if d.Recv == nil || !d.Name.IsExported() {
						continue
					}
					if d.Name.Name == "Set" {
						p.enums[strings.TrimPrefix(exprString(d.Recv.List[0].Type), "*")] = true
						continue
					}
					params := d.Type.Params.List
					if len(params) < 1 || exprString(params[0].Type) != "context.Context" {
						continue
					}
					m := &legacyMethod{name: d.Name.Name, kind: "resp"}
					if len(params) == 2 && len(params[1].Names) <= 1 {
						if id, ok := params[1].Type.(*ast.Ident); ok {
							m.request = id.Name
						}
					}
					if len(params) > 2 || len(params) == 2 && m.request == "" {
						continue
					}
					res := d.Type.Results.List
					switch {
					case len(res) == 1 && exprString(res[0].Type) == "error":
						m.kind = "err"
					case len(res) == 1 && strings.HasPrefix(exprString(res[0].Type), "listing.Iterator["):
						m.kind = "listing"
					case len(res) != 2:
						continue
					}
					p.methods[m.name] = m
				case *ast.GenDecl:
					if d.Tok != token.TYPE {
						continue
					}
					for _, spec := range d.Specs {
						ts := spec.(*ast.TypeSpec)
						st, ok := ts.Type.(*ast.StructType)
						if !ok {
							continue
						}
						ms := &MirrorStruct{byJSON: map[string]*MirrorField{}}
						for _, fl := range st.Fields.List {
							if fl.Tag == nil || len(fl.Names) != 1 {
								continue
							}
							tag, _ := strconv.Unquote(fl.Tag.Value)
							name, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
							if name == "" || name == "-" {
								continue
							}
							mf := &MirrorField{GoName: fl.Names[0].Name, JSON: name, Type: exprString(fl.Type)}
							ms.Fields = append(ms.Fields, mf)
							ms.byJSON[name] = mf
						}
						p.structs[ts.Name.Name] = ms
					}
				}
			}
		}
		out[lp.service] = p
	}
	return out, nil
}

// bindLegacy binds m to a hand-written client method with the same name.
func (b *MethodBind) bindLegacy(p *legacyPkg) bool {
	lm := p.methods[b.method.Name]
	if lm == nil {
		return false
	}
	if b.method.Pagination != nil && lm.kind != "listing" {
		return false
	}
	b.Kind = "legacy"
	b.legacy = p
	b.legacyM = lm
	req := p.structs[lm.request]
	if req == nil {
		return true
	}
	var body *MirrorStruct
	if rb := b.method.RequestBodyField; rb != nil {
		if bf := req.byJSON[rb.Name]; bf != nil {
			b.bodyGo = bf.GoName
			b.bodyPtr = strings.HasPrefix(bf.Type, "*")
			b.bodyType = strings.TrimPrefix(bf.Type, "*")
			body = p.structs[b.bodyType]
		}
	}
	for _, f := range b.allFields() {
		if _, ok := b.fields[f.Name]; ok {
			continue
		}
		fb := &fieldBind{}
		b.fields[f.Name] = fb
		mf, prefix := req.byJSON[f.Name], ""
		if body != nil && !f.IsPath && !f.IsQuery {
			if bmf := body.byJSON[f.Name]; bmf != nil {
				mf, prefix = bmf, "."+b.bodyGo
			}
		}
		if mf == nil {
			continue
		}
		fb.expr = prefix + "." + mf.GoName
		fb.ptr = strings.HasPrefix(mf.Type, "*")
		base := strings.TrimPrefix(mf.Type, "*")
		switch {
		case base == "string" || base == "bool" || base == "int" || base == "int64" || base == "float64":
			fb.kind = base
		case p.enums[base]:
			fb.kind = "legacyenum"
			fb.enumType = p.alias + "." + base
		default:
			fb.kind = "complex"
		}
	}
	return true
}
