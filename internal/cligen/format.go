package main

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

// formatSource returns gofmt-formatted src with unused imports removed. It
// replaces the goimports+gofmt pass genkit ran over the generated files, which
// dominated generation time (goimports rescans the module for every fix). The
// templates emit a fixed import block covering every package any branch may
// reference, so formatting only ever needs to delete imports, never add them.
func formatSource(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// An import is used iff its package name appears as a selector qualifier.
	// Generated code never shadows an imported package name (locals are
	// suffixed: xxxReq, xxxJson, xxxParam, ...), so this scan is exact.
	used := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
		}
		return true
	})

	drop := make(map[int]bool)
	for _, imp := range f.Imports {
		if !used[importName(imp)] {
			for l := fset.Position(imp.Pos()).Line; l <= fset.Position(imp.End()).Line; l++ {
				drop[l] = true
			}
		}
	}

	lines := strings.Split(string(src), "\n")
	keep := lines[:0]
	for i, line := range lines {
		if !drop[i+1] {
			keep = append(keep, line)
		}
	}
	return format.Source([]byte(strings.Join(keep, "\n")))
}

// importName returns the identifier an import is referenced by: the explicit
// name when present, the import path base otherwise. Every template import
// either has base == package name or carries an explicit name.
func importName(imp *ast.ImportSpec) string {
	if imp.Name != nil {
		return imp.Name.Name
	}
	p, _ := strconv.Unquote(imp.Path.Value)
	return path.Base(p)
}
