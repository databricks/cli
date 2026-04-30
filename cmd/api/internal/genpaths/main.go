// genpaths emits cmd/api/paths_generated.go: the deny-list of workspace-routed
// proxy paths that live under accounts/ in the Databricks SDK. The runtime
// classifier in cmd/api uses it to avoid mis-routing those proxies as
// account-scope calls.
//
// The generator parses every service/*/impl.go from the pinned SDK module,
// finds `path :=` assignments, and classifies each as account-routed or as a
// workspace proxy. See classify.go for the per-expression rules.
//
// Output goes to stdout; the Taskfile target redirects it to the checked-in
// file.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

const sdkModule = "github.com/databricks/databricks-sdk-go"

func main() {
	if err := run(os.Stdout); err != nil {
		log.Fatalf("genpaths: %v", err)
	}
}

func run(out *os.File) error {
	dir, err := resolveSDKDir()
	if err != nil {
		return err
	}
	prefixes, exacts, err := scanSDK(dir)
	if err != nil {
		return err
	}
	return render(out, prefixes, exacts)
}

func resolveSDKDir() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", sdkModule)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go list -m %s: %w", sdkModule, err)
	}
	dir := strings.TrimSpace(string(output))
	if dir == "" {
		return "", fmt.Errorf("go list -m %s returned empty directory", sdkModule)
	}
	return dir, nil
}

// scanSDK walks every service/*/impl.go under dir and returns the deny-list
// entries grouped by match flavor. Both slices are deduplicated and sorted.
func scanSDK(dir string) (prefixes, exacts []string, err error) {
	pattern := filepath.Join(dir, "service", "*", "impl.go")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, nil, fmt.Errorf("glob %s: %w", pattern, err)
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no impl.go files found under %s", pattern)
	}
	slices.Sort(files)

	prefixSet := map[string]struct{}{}
	exactSet := map[string]struct{}{}
	for _, f := range files {
		if err := scanFile(f, prefixSet, exactSet); err != nil {
			return nil, nil, err
		}
	}

	prefixes = setToSortedSlice(prefixSet)
	exacts = setToSortedSlice(exactSet)
	return prefixes, exacts, nil
}

func scanFile(path string, prefixSet, exactSet map[string]struct{}) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return scanAST(fset, f, prefixSet, exactSet)
}

// scanAST walks every function body in f and emits deny-list entries for any
// `path :=` or `var path = ...` assignment whose RHS is classified as a
// workspace proxy. Exposed separately from scanFile so tests can drive it
// with in-memory source.
func scanAST(fset *token.FileSet, f *ast.File, prefixSet, exactSet map[string]struct{}) error {
	var classifyErr error
	emit := func(expr ast.Expr, pos token.Pos) bool {
		res, err := classify(expr)
		if err != nil {
			classifyErr = fmt.Errorf("%s: %w", fset.Position(pos), err)
			return false
		}
		switch res.class {
		case classWorkspaceProxyExact:
			exactSet[res.value] = struct{}{}
		case classWorkspaceProxyPrefix:
			prefixSet[res.value] = struct{}{}
		}
		return true
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if classifyErr != nil {
				return false
			}
			switch s := n.(type) {
			case *ast.AssignStmt:
				if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
					return true
				}
				lhs, ok := s.Lhs[0].(*ast.Ident)
				if !ok || lhs.Name != "path" {
					return true
				}
				// Compound assignments (+=, -=, etc.) imply state across
				// statements that the per-expression classifier can't track.
				// Reject them so a future SDK that introduces this idiom
				// fails loudly instead of silently emitting a fragment as a
				// deny-list entry.
				if s.Tok != token.DEFINE && s.Tok != token.ASSIGN {
					classifyErr = fmt.Errorf("%s: compound assignment %v to `path` is not supported by the classifier; "+
						"the SDK has changed path-construction idioms — extend the classifier or split into a single assignment",
						fset.Position(s.Pos()), s.Tok)
					return false
				}
				return emit(s.Rhs[0], s.Pos())
			case *ast.DeclStmt:
				gen, ok := s.Decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.VAR {
					return true
				}
				for _, spec := range gen.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Values) != 1 || len(vs.Names) != 1 {
						continue
					}
					if vs.Names[0].Name != "path" {
						continue
					}
					if !emit(vs.Values[0], vs.Pos()) {
						return false
					}
				}
				return true
			}
			return true
		})
		if classifyErr != nil {
			return classifyErr
		}
	}
	return nil
}

func setToSortedSlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

const fileTemplate = `// Code generated by genpaths. DO NOT EDIT.

package api

// workspaceProxyPrefixes lists path prefixes (Sprintf-derived, ending before
// the first verb) for SDK endpoints that live under accounts/ but route to
// the workspace gateway. Matched with strings.HasPrefix.
var workspaceProxyPrefixes = []string{
{{range .Prefixes}}	{{printf "%q" .}},
{{end}}}

// workspaceProxyExact lists literal paths for SDK endpoints that live under
// accounts/ but route to the workspace gateway. Matched with map equality.
var workspaceProxyExact = map[string]struct{}{
{{range .Exacts}}	{{printf "%q" .}}: {},
{{end}}}
`

func render(out *os.File, prefixes, exacts []string) error {
	tmpl, err := template.New("paths").Parse(fileTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	var raw bytes.Buffer
	if err := tmpl.Execute(&raw, struct {
		Prefixes, Exacts []string
	}{prefixes, exacts}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	formatted, err := format.Source(raw.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt generated source: %w\n--- raw ---\n%s", err, raw.String())
	}
	_, err = out.Write(formatted)
	return err
}
