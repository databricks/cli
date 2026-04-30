package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// pathClass labels how a `path := ...` expression in an SDK service impl.go
// should be treated by the deny-list generator.
type pathClass int

const (
	// classNotAccount is used for paths that do not contain "accounts/".
	classNotAccount pathClass = iota
	// classAccountAPI is for fmt.Sprintf paths whose template contains
	// "accounts/" and whose argument list includes a recognized account-ID
	// source. These are account-routed and contribute nothing to the deny list.
	classAccountAPI
	// classWorkspaceProxyExact is for plain string literal paths containing
	// "accounts/". They go into the exact-match map.
	classWorkspaceProxyExact
	// classWorkspaceProxyPrefix is for fmt.Sprintf paths whose template
	// contains "accounts/" but whose argument list contains no account-ID
	// source. The literal portion up to the first verb goes into the prefix
	// list.
	classWorkspaceProxyPrefix
)

type classification struct {
	class pathClass
	value string
}

// classify inspects the right-hand side of a `path := <expr>` assignment in an
// SDK service impl.go and reports whether the path is account-routed, a
// workspace proxy under accounts/ (and which match flavor), or unrelated.
//
// The classifier is intentionally strict: any expression that looks
// account-related but doesn't match a recognized idiom returns an error so the
// generator fails loudly rather than silently producing a wrong deny-list.
func classify(expr ast.Expr) (classification, error) {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return classification{}, fmt.Errorf("unquote string literal: %w", err)
		}
		if !strings.Contains(s, "accounts/") {
			return classification{class: classNotAccount}, nil
		}
		return classification{class: classWorkspaceProxyExact, value: s}, nil
	}

	if call, ok := expr.(*ast.CallExpr); ok && isFmtSprintf(call) {
		return classifySprintf(call)
	}

	// Fallback: an unrecognized idiom. If the subtree contains no "accounts/"
	// literal, it can't be a path we care about. If it does, the generator
	// won't be able to decide, so fail.
	if hasAccountsLiteral(expr) {
		return classification{}, errors.New(
			"path expression mentions \"accounts/\" but uses an unrecognized construction idiom; " +
				"either teach the classifier the new shape or extend the deny-list manually")
	}
	return classification{class: classNotAccount}, nil
}

func classifySprintf(call *ast.CallExpr) (classification, error) {
	if len(call.Args) == 0 {
		return classification{}, errors.New("fmt.Sprintf with no template")
	}
	tmplLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || tmplLit.Kind != token.STRING {
		// Sprintf with a non-literal template. We can't reason about it.
		return classification{}, errors.New("fmt.Sprintf with non-literal template")
	}
	template, err := strconv.Unquote(tmplLit.Value)
	if err != nil {
		return classification{}, fmt.Errorf("unquote Sprintf template: %w", err)
	}
	if !strings.Contains(template, "accounts/") {
		return classification{class: classNotAccount}, nil
	}

	for _, a := range call.Args[1:] {
		if isAccountIDSource(a) {
			return classification{class: classAccountAPI}, nil
		}
	}

	prefix := prefixUpToFirstVerb(template)
	if prefix == "" {
		return classification{}, fmt.Errorf("Sprintf template %q has no format verb", template)
	}
	// Guard: a prefix ending exactly at "accounts/" would match every account
	// API under that family if it leaked into the deny-list. Refuse to emit
	// rather than silently classify all of them as workspace-routed.
	if strings.HasSuffix(prefix, "/accounts/") {
		return classification{}, fmt.Errorf(
			"path template %q has a format verb immediately after \"accounts/\" but no recognized account-ID source; "+
				"either teach the classifier a new account-ID spelling or extend the deny-list manually",
			template)
	}
	// Guard: if the first format verb appears before the "accounts/" segment,
	// the extracted prefix would not contain "accounts/" and would over-match
	// unrelated APIs at runtime. Refuse to emit.
	if !strings.Contains(prefix, "/accounts/") {
		return classification{}, fmt.Errorf(
			"path template %q has a format verb before the \"accounts/\" segment; "+
				"a prefix derived from this template would over-match unrelated APIs",
			template)
	}
	return classification{class: classWorkspaceProxyPrefix, value: prefix}, nil
}

func isFmtSprintf(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return id.Name == "fmt" && sel.Sel.Name == "Sprintf"
}

// isAccountIDSource reports whether the expression resolves to one of the
// recognized spellings of "the account ID for the current call". The list is
// an explicit closed allowlist: a new SDK spelling we don't recognize triggers
// a generator error rather than being silently classified as a workspace
// proxy. Currently allowed shapes: a.client.ConfiguredAccountID(),
// cfg.AccountID, a.client.Config.AccountID.
func isAccountIDSource(e ast.Expr) bool {
	if call, ok := e.(*ast.CallExpr); ok {
		if isClientConfiguredAccountID(call) {
			return true
		}
	}
	if sel, ok := e.(*ast.SelectorExpr); ok {
		if isCfgAccountID(sel) || isClientConfigAccountID(sel) {
			return true
		}
	}
	return false
}

// isClientConfiguredAccountID matches a.client.ConfiguredAccountID() with no
// arguments. This is the only spelling used in the SDK today.
func isClientConfiguredAccountID(call *ast.CallExpr) bool {
	if len(call.Args) != 0 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "ConfiguredAccountID" {
		return false
	}
	receiver, ok := sel.X.(*ast.SelectorExpr)
	if !ok || receiver.Sel.Name != "client" {
		return false
	}
	id, ok := receiver.X.(*ast.Ident)
	return ok && id.Name == "a"
}

func isCfgAccountID(s *ast.SelectorExpr) bool {
	if s.Sel.Name != "AccountID" {
		return false
	}
	id, ok := s.X.(*ast.Ident)
	return ok && id.Name == "cfg"
}

func isClientConfigAccountID(s *ast.SelectorExpr) bool {
	if s.Sel.Name != "AccountID" {
		return false
	}
	inner, ok := s.X.(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != "Config" {
		return false
	}
	inner2, ok := inner.X.(*ast.SelectorExpr)
	if !ok || inner2.Sel.Name != "client" {
		return false
	}
	id, ok := inner2.X.(*ast.Ident)
	return ok && id.Name == "a"
}

// prefixUpToFirstVerb returns the literal portion an fmt.Sprintf template
// would render before the first format verb, with "%%" escapes resolved to
// "%". The result is what runtime strings.HasPrefix compares against, so the
// escapes must match the rendered URL rather than the template source.
// Returns "" if the template has no real verb.
func prefixUpToFirstVerb(template string) string {
	var b strings.Builder
	for i := 0; i < len(template); i++ {
		if template[i] != '%' {
			b.WriteByte(template[i])
			continue
		}
		if i+1 < len(template) && template[i+1] == '%' {
			b.WriteByte('%')
			i++
			continue
		}
		return b.String()
	}
	return ""
}

func hasAccountsLiteral(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		s, err := strconv.Unquote(lit.Value)
		if err == nil && strings.Contains(s, "accounts/") {
			found = true
			return false
		}
		return true
	})
	return found
}
