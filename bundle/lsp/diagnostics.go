package lsp

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/dyn"
)

const diagnosticSource = "databricks-bundle-lsp"

// DiagnoseInterpolations checks all ${...} interpolation references in the document
// and returns diagnostics for references that cannot be resolved in the merged tree.
func DiagnoseInterpolations(lines []string, tree dyn.Value) []Diagnostic {
	var diags []Diagnostic
	for lineIdx, line := range lines {
		matches := InterpolationRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			// m[0]:m[1] is the full "${...}" match.
			// m[2]:m[3] is the captured path inside ${}.
			path := line[m[2]:m[3]]

			if isComputedPath(path) {
				continue
			}

			_, found := ResolveDefinition(tree, path)
			if found {
				continue
			}

			diags = append(diags, Diagnostic{
				Range: Range{
					Start: Position{Line: lineIdx, Character: m[0]},
					End:   Position{Line: lineIdx, Character: m[1]},
				},
				Severity: DiagnosticSeverityWarning,
				Source:   diagnosticSource,
				Message:  fmt.Sprintf("Cannot resolve reference %q", path),
			})
		}
	}
	return diags
}

// isComputedPath returns true if the path is known to be populated at deploy
// time and won't appear in the static merged tree. Derived from the
// computedKeys list in completion.go so both features share a single source
// of truth.
func isComputedPath(path string) bool {
	for _, key := range computedKeys {
		if path == key || strings.HasPrefix(path, key+".") || strings.HasPrefix(path, key+"[") {
			return true
		}
	}
	return false
}
