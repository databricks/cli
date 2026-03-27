package lsp

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/dyn"
)

const diagnosticSource = "databricks-bundle-lsp"

// computedPrefixes are path prefixes that are populated at deploy time and
// should not be flagged as unresolved references. Prefixes ending in "."
// match any sub-path; others require an exact match.
var computedPrefixes = []string{
	"bundle.target",
	"bundle.environment",
	"bundle.git.",
	"workspace.current_user.",
	"workspace.root_path",
	"workspace.file_path",
	"workspace.resource_path",
	"workspace.artifact_path",
	"workspace.state_path",
}

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
// time and won't appear in the static merged tree.
func isComputedPath(path string) bool {
	// var.* references are rewritten to variables.* by ResolveDefinition,
	// so we only need to handle the other computed prefixes here.
	for _, prefix := range computedPrefixes {
		if strings.HasSuffix(prefix, ".") {
			// Dot-terminated: match any sub-path.
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else {
			// Exact match only.
			if path == prefix {
				return true
			}
		}
	}
	return false
}
