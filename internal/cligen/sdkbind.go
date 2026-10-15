package main

// This file binds cli.json methods (old SDK naming) to sdk-go client methods.
//
// cli.json describes commands in terms of the REST API: the path, the request
// fields by wire name and their flags. The generated commands call sdk-go
// clients and bind flags and --json to the JSON mirror types in libs/sdktypes,
// so for every method we need the sdk-go module, client method, request and
// response types, and the Go field name behind every wire field. All of this is
// read from the sources of the sdk-go modules required by go.mod and from the
// generated mirrors, matching methods by their normalized HTTP path.

import (
	"cmp"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	sdkModulePrefix   = "github.com/databricks/sdk-go/"
	mirrorPrefix      = "github.com/databricks/cli/libs/sdktypes/"
	sdkClientsFile    = "libs/sdkclients/services.go"
	sdkTypesDir       = "libs/sdktypes"
	mirrorTypesFile   = "types.generated.go"
	sdkClientFileName = "client.go"
)

// SdkPackage is one versioned sdk-go package (e.g. jobs/v2).
type SdkPackage struct {
	// Rel is the path relative to the sdk-go module root, e.g. "uc/tables/v1".
	Rel         string
	ImportPath  string
	MirrorPath  string
	MirrorAlias string
	SdkAlias    string
	ClientField string

	methods map[string]*SdkMethod
	structs map[string]*MirrorStruct
	// models are the sdk-go model structs, for requests without a mirror (no
	// JSON body); their JSON names are only known for query parameters.
	models map[string]*MirrorStruct
}

// MirrorStruct is a struct type of a libs/sdktypes mirror package.
type MirrorStruct struct {
	Fields []*MirrorField
	byJSON map[string]*MirrorField
}

type MirrorField struct {
	GoName string
	JSON   string
	// Type is the Go type expression as written in the mirror package.
	Type string
}

// SdkMethod is an exported method of an sdk-go client.
type SdkMethod struct {
	Pkg  *SdkPackage
	Name string
	// Request is the request type name, "" for methods without a request.
	Request string
	// Kind is "resp", "err", "iter", "waiter" or "op".
	Kind string
	// Result is the response type (resp), or what Wait returns (waiter, op).
	Result string
	// Iter is the name of the companion ...Iter method and ItemType its element type.
	Iter     string
	ItemType string
	// WaiterIDs name the getters of a waiter's identifiers, e.g. "GetRunId".
	WaiterIDs []string
	// pollRequest is the request type of the method a waiter polls.
	pollRequest string
	// baseResult is the response type of the call that starts a wait.
	baseResult string
	// hasBody is set when a JSON body is sent; bodyGo is the request field
	// sent as the body, or "" for the whole request.
	hasBody bool
	bodyGo  string

	verb       string
	path       string
	pathFields []string
	// queryFields maps query parameter names to request Go fields.
	queryFields map[string]string
}

type sdkIndex struct {
	pkgs   []*SdkPackage
	byPath map[string][]*SdkMethod
}

func goListDirs(pkgs []string) (map[string]string, error) {
	args := append([]string{"list", "-f", "{{.ImportPath}} {{.Dir}}"}, pkgs...)
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("go list: %w: %s", err, ee.Stderr)
		}
		return nil, fmt.Errorf("go list: %w", err)
	}
	dirs := map[string]string{}
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		p, d, ok := strings.Cut(line, " ")
		if ok {
			dirs[p] = d
		}
	}
	return dirs, nil
}

// loadSdkIndex reads libs/sdkclients/services.go for the client field of every
// sdk-go package, the sdk-go client sources and the mirror types.
func loadSdkIndex(repoRoot string) (*sdkIndex, error) {
	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath.Join(repoRoot, sdkClientsFile), nil, 0)
	if err != nil {
		return nil, err
	}
	aliasToPath := map[string]string{}
	for _, imp := range sf.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if !strings.HasPrefix(p, sdkModulePrefix) || imp.Name == nil {
			continue
		}
		aliasToPath[imp.Name.Name] = p
	}
	idx := &sdkIndex{byPath: map[string][]*SdkMethod{}}
	var importPaths []string
	ast.Inspect(sf, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "services" {
			return true
		}
		for _, f := range ts.Type.(*ast.StructType).Fields.List {
			sel := f.Type.(*ast.StarExpr).X.(*ast.SelectorExpr)
			alias := sel.X.(*ast.Ident).Name
			p := aliasToPath[alias]
			rel := strings.TrimPrefix(p, sdkModulePrefix)
			idx.pkgs = append(idx.pkgs, &SdkPackage{
				Rel:         rel,
				ImportPath:  p,
				MirrorPath:  mirrorPrefix + rel,
				MirrorAlias: alias,
				SdkAlias:    "sdk" + alias,
				ClientField: f.Names[0].Name,
				methods:     map[string]*SdkMethod{},
				structs:     map[string]*MirrorStruct{},
				models:      map[string]*MirrorStruct{},
			})
			importPaths = append(importPaths, p)
		}
		return false
	})
	dirs, err := goListDirs(importPaths)
	if err != nil {
		return nil, err
	}
	for _, p := range idx.pkgs {
		if err := p.loadMethods(dirs[p.ImportPath]); err != nil {
			return nil, fmt.Errorf("%s: %w", p.Rel, err)
		}
		if err := p.loadModels(filepath.Join(dirs[p.ImportPath], "model.go")); err != nil {
			return nil, fmt.Errorf("%s: %w", p.Rel, err)
		}
		if err := p.loadMirror(filepath.Join(repoRoot, sdkTypesDir, p.Rel, mirrorTypesFile)); err != nil {
			return nil, fmt.Errorf("%s: %w", p.Rel, err)
		}
		for _, m := range p.methods {
			if m.path != "" && m.Kind != "iter" {
				idx.byPath[m.path] = append(idx.byPath[m.path], m)
			}
		}
	}
	for k := range idx.byPath {
		ms := idx.byPath[k]
		slices.SortFunc(ms, func(a, b *SdkMethod) int {
			if a.Pkg.Rel != b.Pkg.Rel {
				return cmp.Compare(a.Pkg.Rel, b.Pkg.Rel)
			}
			return cmp.Compare(a.Name, b.Name)
		})
	}
	return idx, nil
}

func exprString(e ast.Expr) string {
	var sb strings.Builder
	_ = printer.Fprint(&sb, token.NewFileSet(), e)
	return sb.String()
}

func (p *SdkPackage) loadMirror(path string) error {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return err
	}
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
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
	return nil
}

func (p *SdkPackage) loadModels(path string) error {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return err
	}
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts := spec.(*ast.TypeSpec)
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			ms := &MirrorStruct{byJSON: map[string]*MirrorField{}}
			for _, fl := range st.Fields.List {
				for _, n := range fl.Names {
					ms.Fields = append(ms.Fields, &MirrorField{GoName: n.Name, Type: exprString(fl.Type)})
				}
			}
			p.models[ts.Name.Name] = ms
		}
	}
	return nil
}

var reqIdentRe = regexp.MustCompile(`^\*?req\.([A-Z][\w.]*)$`)

func (p *SdkPackage) loadMethods(dir string) error {
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, sdkClientFileName), nil, 0)
	if err != nil {
		return err
	}
	all := map[string]*ast.FuncDecl{}
	waiters := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}
		recv := exprString(fd.Recv.List[0].Type)
		switch {
		case recv == "*internalClient":
			all[fd.Name.Name] = fd
		case strings.HasSuffix(recv, "Waiter") || strings.HasSuffix(recv, "Operation"):
			waiters[strings.TrimPrefix(recv, "*")+"."+fd.Name.Name] = fd
		}
	}
	var impl *ast.FuncDecl
	for name, fd := range all {
		if !ast.IsExported(name) {
			continue
		}
		m := &SdkMethod{Pkg: p, Name: name}
		params := fd.Type.Params.List
		if len(params) >= 2 {
			if id, ok := params[1].Type.(*ast.Ident); ok {
				m.Request = id.Name
			}
		}
		results := fd.Type.Results.List
		first := exprString(results[0].Type)
		switch {
		case len(results) == 1:
			if strings.HasPrefix(first, "iter.Seq2[") {
				m.Kind = "iter"
				m.ItemType = strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(first, "iter.Seq2["), ", error]"), "*")
			} else {
				m.Kind = "err"
			}
		case strings.HasSuffix(first, "Waiter"):
			m.Kind = "waiter"
			m.Result = waitResult(waiters, strings.TrimPrefix(first, "*"))
			m.pollRequest = waiterPollRequest(f, strings.TrimPrefix(first, "*"))
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok || fd.Recv == nil || exprString(fd.Recv.List[0].Type) != first {
					continue
				}
				if strings.HasPrefix(fd.Name.Name, "Get") && fd.Type.Results.NumFields() == 1 {
					m.WaiterIDs = append(m.WaiterIDs, fd.Name.Name)
				}
			}
		case strings.HasSuffix(first, "Operation"):
			m.Kind = "op"
			m.Result = waitResult(waiters, strings.TrimPrefix(first, "*"))
		default:
			m.Kind = "resp"
			m.Result = strings.TrimPrefix(first, "*")
		}
		impl = fd
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == "c" && strings.HasSuffix(sel.Sel.Name, "Base") {
					if b := all[sel.Sel.Name]; b != nil {
						impl = b
					}
				}
			}
			return true
		})
		if impl != fd && impl.Type.Results.NumFields() == 2 {
			m.baseResult = strings.TrimPrefix(exprString(impl.Type.Results.List[0].Type), "*")
		}
		m.verb, m.path, m.pathFields = httpRoute(impl)
		m.queryFields = queryRoute(impl)
		m.hasBody, m.bodyGo = bodyRoute(impl)
		p.methods[name] = m
	}
	for name, m := range p.methods {
		if it, ok := p.methods[name+"Iter"]; ok && it.Kind == "iter" {
			m.Iter = it.Name
			m.ItemType = it.ItemType
		}
	}
	return nil
}

// waiterPollRequest returns the request type of the waiter's poll function,
// from its `poll func(context.Context, GetXRequest, ...call.Option) (...)` field.
func waiterPollRequest(f *ast.File, waiter string) string {
	req := ""
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != waiter {
			return true
		}
		for _, fl := range ts.Type.(*ast.StructType).Fields.List {
			ft, ok := fl.Type.(*ast.FuncType)
			if ok && len(fl.Names) == 1 && fl.Names[0].Name == "poll" && len(ft.Params.List) >= 2 {
				req = exprString(ft.Params.List[1].Type)
			}
		}
		return false
	})
	return req
}

func waitResult(waiters map[string]*ast.FuncDecl, typ string) string {
	if wf := waiters[typ+".Wait"]; wf != nil && wf.Type.Results.NumFields() == 2 {
		return strings.TrimPrefix(exprString(wf.Type.Results.List[0].Type), "*")
	}
	return ""
}

// httpRoute extracts the HTTP verb, the normalized path ("{}" per path
// parameter) and the request field behind every path parameter ("" for the
// implicit account ID) from a generated sdk-go method body.
func httpRoute(fd *ast.FuncDecl) (string, string, []string) {
	var verb, path string
	var fields []string
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.KeyValueExpr:
			if k, ok := x.Key.(*ast.Ident); ok && k.Name == "Method" {
				if lit, ok := x.Value.(*ast.BasicLit); ok && verb == "" {
					verb, _ = strconv.Unquote(lit.Value)
				}
			}
		case *ast.AssignStmt:
			if len(x.Lhs) == 1 && exprString(x.Lhs[0]) == "baseURL.Path" {
				if lit, ok := x.Rhs[0].(*ast.BasicLit); ok {
					path, _ = strconv.Unquote(lit.Value)
				}
			}
		}
		return true
	})
	if path != "" {
		return verb, path, nil
	}
	var sb strings.Builder
	var walk func(stmts []ast.Stmt)
	walk = func(stmts []ast.Stmt) {
		for _, s := range stmts {
			switch x := s.(type) {
			case *ast.ExprStmt:
				call, ok := x.X.(*ast.CallExpr)
				if !ok {
					continue
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || exprString(sel.X) != "pb" {
					continue
				}
				switch sel.Sel.Name {
				case "literal":
					v, _ := strconv.Unquote(call.Args[0].(*ast.BasicLit).Value)
					sb.WriteString(v)
				case "singleSegment", "multiSegments":
					sb.WriteString("{}")
					field := ""
					if m := reqIdentRe.FindStringSubmatch(exprString(call.Args[0])); m != nil {
						field = m[1]
					}
					fields = append(fields, field)
				}
			case *ast.IfStmt:
				// `if req.X == nil { pb.singleSegment("") } else { pb.singleSegment(*req.X) }`
				if x.Else != nil {
					if eb, ok := x.Else.(*ast.BlockStmt); ok {
						walk(eb.List)
						continue
					}
				}
				walk(x.Body.List)
			}
		}
	}
	walk(fd.Body.List)
	return verb, sb.String(), fields
}

// bodyRoute finds the `json.Marshal(wireReq...)` call of a method body.
func bodyRoute(fd *ast.FuncDecl) (bool, string) {
	found, field := false, ""
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || exprString(call.Fun) != "json.Marshal" || len(call.Args) != 1 {
			return true
		}
		arg := exprString(call.Args[0])
		if arg == "wireReq" || strings.HasPrefix(arg, "wireReq.") {
			found, field = true, strings.TrimPrefix(strings.TrimPrefix(arg, "wireReq"), ".")
		}
		return true
	})
	return found, field
}

var queryFieldRe = regexp.MustCompile(`^(?:req|wireReq)\.([A-Z][\w.]*)$`)

// queryRoute maps query parameter names to request fields from the
// `addQueryValue(queryParams, "name", req.Field)` calls of a method body.
func queryRoute(fd *ast.FuncDecl) map[string]string {
	out := map[string]string{}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || exprString(call.Fun) != "addQueryValue" || len(call.Args) != 3 {
			return true
		}
		lit, ok := call.Args[1].(*ast.BasicLit)
		if !ok {
			return true
		}
		name, _ := strconv.Unquote(lit.Value)
		if m := queryFieldRe.FindStringSubmatch(exprString(call.Args[2])); m != nil {
			out[name] = m[1]
		}
		return true
	})
	return out
}

var pathParamRe = regexp.MustCompile(`\{[^}]*\}`)

func normPath(p string) string {
	return pathParamRe.ReplaceAllString(p, "{}")
}

func pathParams(p string) []string {
	var out []string
	for _, m := range pathParamRe.FindAllString(p, -1) {
		name := strings.Trim(m, "{}")
		name = strings.TrimSuffix(name, "...")
		name, _, _ = strings.Cut(name, "=")
		out = append(out, name)
	}
	return out
}

//go:embed http_verbs.json
var httpVerbsJSON []byte

// httpVerbs maps "<package>.<lowercase service>.<method>" to the HTTP verb of
// every method of the previous SDK. cli.json carries paths but no verbs; this
// table was extracted once from databricks-sdk-go v0.188.0 (service/*/impl.go)
// and is frozen: methods added to cli.json later need an entry here.
var httpVerbs = func() map[string]string {
	m := map[string]string{}
	if err := json.Unmarshal(httpVerbsJSON, &m); err != nil {
		panic(err)
	}
	return m
}()

func methodVerb(svc *ServiceJSON, m *MethodJSON) string {
	return httpVerbs[svc.Package.Name+"."+strings.ToLower(svc.Name)+"."+m.Name]
}

func leadingWord(s string) string {
	for i := 1; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			return s[:i]
		}
	}
	return s
}

// pathMatch reports whether an sdk-go path matches a cli.json path. Exact
// matches return no constants. Otherwise an sdk-go parameter may stand for a
// literal segment of the cli.json path (the typed permission endpoints, e.g.
// /permissions/jobs/{job_id} served by /permissions/{type}/{id}); the literal
// is returned for that parameter position.
func pathMatch(sdkPath, cliPath string) (bool, map[int]string) {
	if sdkPath == cliPath {
		return true, nil
	}
	ss := strings.Split(sdkPath, "/")
	cs := strings.Split(cliPath, "/")
	if len(ss) != len(cs) {
		return false, nil
	}
	consts := map[int]string{}
	param := 0
	for i := range ss {
		switch {
		case ss[i] == "{}" && cs[i] == "{}":
			param++
		case ss[i] == "{}":
			consts[param] = cs[i]
			param++
		case ss[i] != cs[i]:
			return false, nil
		}
	}
	return len(consts) > 0, consts
}

// match finds the sdk-go method for a cli.json method by HTTP verb and path.
// It also returns literal values for sdk-go path parameters that cli.json has
// as fixed path segments.
func (idx *sdkIndex) match(svc *ServiceJSON, m *MethodJSON) (*SdkMethod, map[int]string) {
	verb := methodVerb(svc, m)
	np := normPath(m.Path)
	type cand struct {
		m      *SdkMethod
		consts map[int]string
	}
	var cands []cand
	for _, c := range idx.byPath[np] {
		cands = append(cands, cand{c, nil})
	}
	if len(cands) == 0 {
		for p, ms := range idx.byPath {
			if ok, consts := pathMatch(p, np); ok && consts != nil {
				for _, c := range ms {
					cands = append(cands, cand{c, consts})
				}
			}
		}
	}
	var best cand
	bestScore := -1
	for _, c := range cands {
		if verb != "" && c.m.verb != verb {
			continue
		}
		score := 0
		switch {
		case c.m.Name == m.Name:
			score += 100
		case strings.HasPrefix(c.m.Name, m.Name):
			score += 60
		case leadingWord(c.m.Name) == leadingWord(m.Name):
			score += 40
		}
		if m.Pagination != nil && c.m.Iter != "" {
			score += 10
		}
		// Prefer a stable choice when several modules serve the same route.
		if score > bestScore || score == bestScore && c.m.Pkg.Rel+c.m.Name < best.m.Pkg.Rel+best.m.Name {
			best, bestScore = c, score
		}
	}
	if best.m == nil {
		return nil, nil
	}
	return best.m, best.consts
}

// debugBindings prints the binding of every method to stderr when
// CLIGEN_DEBUG is set.
func debugf(format string, args ...any) {
	//nolint:forbidigo
	if os.Getenv("CLIGEN_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
