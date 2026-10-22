// Package fix completes reviewed field-copy mappings after an SDK update.
package fix

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

const directive = "// sdk:copy-from "

// Run fills missing fields in annotated direct-resource literals. Unmatched
// fields are reported for review and remain exhaustruct errors; they do not
// prevent the rest of bump-sdk-post from regenerating artifacts. Invalid Go or
// directives fail before any file is written.
func Run(ctx context.Context, root string, out io.Writer) error {
	cfg := &packages.Config{Context: ctx, Dir: root, Mode: packages.LoadSyntax}
	pkgs, err := load(cfg)
	if err != nil {
		return err
	}
	changes := make(map[string][]byte)
	var report bytes.Buffer
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			name := pkg.Fset.PositionFor(file.Pos(), false).Filename
			original, err := os.ReadFile(name)
			if err != nil {
				return fmt.Errorf("read %s: %w", filepath.ToSlash(name), err)
			}
			updated, err := rewrite(pkg, file, original, &report)
			if err != nil {
				return fmt.Errorf("%s: %w", filepath.ToSlash(name), err)
			}
			if !bytes.Equal(updated, original) {
				changes[name] = updated
			}
		}
	}
	if len(changes) > 0 {
		// Validate the entire proposed package, including interactions between
		// nested literals, before touching any source file.
		cfg.Overlay = changes
		if _, err := load(cfg); err != nil {
			return fmt.Errorf("validate proposed field copies: %w", err)
		}
	}
	var names []string
	for name := range changes {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if err := os.WriteFile(name, changes[name], 0); err != nil {
			return fmt.Errorf("write %s: %w", filepath.ToSlash(name), err)
		}
	}
	_, err = report.WriteTo(out)
	return err
}

func load(cfg *packages.Config) ([]*packages.Package, error) {
	pkgs, err := packages.Load(cfg, "./bundle/direct/dresources")
	if err != nil {
		return nil, fmt.Errorf("load direct resources: %w", err)
	}
	var problems []string
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			problems = append(problems, err.Error())
		}
	})
	if len(problems) > 0 {
		slices.Sort(problems)
		return nil, fmt.Errorf("type-check direct resources:\n%s", strings.Join(problems, "\n"))
	}
	return pkgs, nil
}

type insertion struct {
	offset int
	text   string
}

func rewrite(pkg *packages.Package, file *ast.File, original []byte, report io.Writer) ([]byte, error) {
	markers := make(map[int]*ast.Comment)
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, strings.TrimSpace(directive)) {
				markers[pkg.Fset.PositionFor(comment.Pos(), false).Line] = comment
			}
		}
	}
	if len(markers) == 0 {
		return original, nil
	}
	var edits []insertion
	var rewriteErr error
	ast.Inspect(file, func(node ast.Node) bool {
		if rewriteErr != nil {
			return false
		}
		lit, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		line := pkg.Fset.PositionFor(lit.Lbrace, false).Line
		marker, ok := markers[line]
		if !ok || marker.Pos() < lit.Lbrace || marker.End() > lit.Rbrace {
			return true
		}
		brace := pkg.Fset.PositionFor(lit.Lbrace, false).Offset
		comment := pkg.Fset.PositionFor(marker.Pos(), false).Offset
		if len(bytes.TrimSpace(original[brace+1:comment])) != 0 {
			return true
		}
		delete(markers, line)
		lines, err := copies(pkg, lit, marker, report)
		if err != nil {
			rewriteErr = fmt.Errorf("line %d: %w", line, err)
			return false
		}
		if len(lines) > 0 {
			// Insert after the directive line. Existing comments retain their
			// position relative to the original fields, including zero-value groups.
			end := pkg.Fset.PositionFor(marker.End(), false).Offset
			offset := end + bytes.IndexByte(original[end:], '\n') + 1
			edits = append(edits, insertion{offset: offset, text: strings.Join(lines, "\n") + "\n"})
		}
		return true
	})
	if rewriteErr != nil {
		return nil, rewriteErr
	}
	if len(markers) != 0 {
		return nil, fmt.Errorf("%s directive must follow a struct literal's opening brace on the same line", strings.TrimSpace(directive))
	}
	if len(edits) == 0 {
		return original, nil
	}
	slices.SortFunc(edits, func(a, b insertion) int { return b.offset - a.offset })
	updated := bytes.Clone(original)
	for _, edit := range edits {
		updated = slices.Insert(updated, edit.offset, []byte(edit.text)...)
	}
	formatted, err := format.Source(updated)
	if err != nil {
		return nil, fmt.Errorf("format field copies: %w", err)
	}
	return formatted, nil
}

func copies(pkg *packages.Package, lit *ast.CompositeLit, marker *ast.Comment, report io.Writer) ([]string, error) {
	source := strings.TrimSpace(strings.TrimPrefix(marker.Text, directive))
	expr, err := parser.ParseExpr(source)
	if err != nil || !isSelector(expr) {
		return nil, fmt.Errorf("copy source must be an identifier or selector chain, got %q", source)
	}
	value, err := types.Eval(pkg.Fset, pkg.Types, lit.Pos(), source)
	if err != nil {
		return nil, fmt.Errorf("resolve copy source %q: %w", source, err)
	}
	if !value.IsValue() || asStruct(value.Type) == nil {
		return nil, fmt.Errorf("copy source %q must be a struct or pointer to a struct", source)
	}
	destination := pkg.TypesInfo.TypeOf(lit)
	st, ok := destination.Underlying().(*types.Struct)
	if !ok {
		return nil, errors.New("copy directive requires a struct literal")
	}
	var present [][]int
	for _, element := range lit.Elts {
		kv, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return nil, errors.New("copy directive requires a keyed struct literal")
		}
		key := kv.Key.(*ast.Ident)
		_, index, _ := types.LookupFieldOrMethod(destination, false, pkg.Types, key.Name)
		present = append(present, index)
	}
	fields := missingFields(st, nil, present)
	slices.SortFunc(fields, func(a, b field) int { return strings.Compare(a.variable.Name(), b.variable.Name()) })
	var lines []string
	for _, field := range fields {
		name := field.variable.Name()
		obj, _, _ := types.LookupFieldOrMethod(destination, false, pkg.Types, name)
		reason := "destination field is shadowed or ambiguous"
		if obj == field.variable {
			reason = copyReason(value.Type, field, pkg.Types)
		}
		position := pkg.Fset.PositionFor(lit.Lbrace, false)
		if reason != "" {
			fmt.Fprintf(report, "%s:%d: leave %s for review: %s\n", filepath.Base(position.Filename), position.Line, name, reason)
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s.%s,", name, source, name))
		fmt.Fprintf(report, "%s:%d: add %s: %s.%s\n", filepath.Base(position.Filename), position.Line, name, source, name)
	}
	return lines, nil
}

// isSelector excludes calls, indexes and other expressions with evaluation
// behavior beyond the reviewed source variable or selector chain.
func isSelector(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return isSelector(expr.X)
	default:
		return false
	}
}

type field struct {
	variable *types.Var
	tag      string
}

// missingFields follows only value embeddings already partially populated by
// promoted keys. Assigning the embedded field itself initializes all its fields.
func missingFields(st *types.Struct, prefix []int, present [][]int) []field {
	var missing []field
	for i := range st.NumFields() {
		path := append(slices.Clone(prefix), i)
		initialized, partial := false, false
		for _, key := range present {
			if slices.Equal(key, path) {
				initialized = true
			}
			if len(key) > len(path) && slices.Equal(key[:len(path)], path) {
				partial = true
			}
		}
		if initialized {
			continue
		}
		f := st.Field(i)
		if partial {
			if embedded, ok := f.Type().Underlying().(*types.Struct); ok {
				missing = append(missing, missingFields(embedded, path, present)...)
				continue
			}
		}
		missing = append(missing, field{variable: f, tag: st.Tag(i)})
	}
	return missing
}

func copyReason(source types.Type, destination field, pkg *types.Package) string {
	name := destination.variable.Name()
	if !destination.variable.Exported() || name == "ForceSendFields" || name == "UpdateMask" {
		return "metadata or unexported field requires an explicit value"
	}
	wireName := jsonName(destination.tag)
	if wireName == "" || wireName == "-" {
		return "destination has no JSON payload field name"
	}
	obj, index, _ := types.LookupFieldOrMethod(source, true, pkg, name)
	f, ok := obj.(*types.Var)
	if !ok || !f.IsField() {
		return "source field is absent or ambiguous"
	}
	if !types.Identical(f.Type(), destination.variable.Type()) {
		return "source and destination types differ"
	}
	st := asStruct(source)
	for _, i := range index[:len(index)-1] {
		embedded, ok := st.Field(i).Type().Underlying().(*types.Struct)
		if !ok {
			return "source field crosses a pointer embedding"
		}
		st = embedded
	}
	if jsonName(st.Tag(index[len(index)-1])) != wireName {
		return "source and destination JSON field names differ"
	}
	return ""
}

func asStruct(t types.Type) *types.Struct {
	if p, ok := t.Underlying().(*types.Pointer); ok {
		t = p.Elem()
	}
	st, _ := t.Underlying().(*types.Struct)
	return st
}

func jsonName(tag string) string {
	name, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
	return name
}
