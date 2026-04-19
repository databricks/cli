package lsp

import (
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// InterpolationRe matches ${...} interpolation expressions in strings.
var InterpolationRe = dynvar.InterpolationRegexp

// InterpolationRef represents a ${...} reference found at a cursor position.
type InterpolationRef struct {
	Path  string // e.g., "resources.jobs.my_job.name"
	Range Range  // range of the full "${...}" token
}

// FindInterpolationAtPosition finds the ${...} expression the cursor is inside.
func FindInterpolationAtPosition(lines []string, pos Position) (InterpolationRef, bool) {
	if pos.Line < 0 || pos.Line >= len(lines) {
		return InterpolationRef{}, false
	}

	line := lines[pos.Line]
	matches := InterpolationRe.FindAllStringSubmatchIndex(line, -1)
	for _, m := range matches {
		// m[0]:m[1] is the full match "${...}"
		// m[2]:m[3] is the first capture group (the path inside ${})
		start := m[0]
		end := m[1]
		if pos.Character >= start && pos.Character < end {
			path := line[m[2]:m[3]]
			return InterpolationRef{
				Path: path,
				Range: Range{
					Start: Position{Line: pos.Line, Character: start},
					End:   Position{Line: pos.Line, Character: end},
				},
			}, true
		}
	}
	return InterpolationRef{}, false
}

// ResolveDefinition resolves a path string against the merged tree and returns its source location.
func ResolveDefinition(tree dyn.Value, pathStr string) (dyn.Location, bool) {
	if !tree.IsValid() {
		return dyn.Location{}, false
	}

	// Handle var.X shorthand: rewrite to variables.X.
	if strings.HasPrefix(pathStr, "var.") {
		pathStr = "variables." + strings.TrimPrefix(pathStr, "var.")
	}

	p, err := dyn.NewPathFromString(pathStr)
	if err != nil {
		return dyn.Location{}, false
	}

	v, err := dyn.GetByPath(tree, p)
	if err != nil {
		return dyn.Location{}, false
	}

	loc := v.Location()
	if loc.File == "" {
		return dyn.Location{}, false
	}
	return loc, true
}

// InterpolationReference records a ${...} reference found in the merged tree.
type InterpolationReference struct {
	Path     string       // dyn path where the reference was found
	Location dyn.Location // source location of the string containing the reference
	RefStr   string       // the full "${...}" expression
}

// FindInterpolationReferences walks the merged tree to find all ${...} string values
// whose reference path starts with the given resource path prefix.
func FindInterpolationReferences(tree dyn.Value, resourcePath string) []InterpolationReference {
	if !tree.IsValid() {
		return nil
	}

	var refs []InterpolationReference
	dyn.Walk(tree, func(p dyn.Path, v dyn.Value) (dyn.Value, error) { //nolint:errcheck
		s, ok := v.AsString()
		if !ok {
			return v, nil
		}

		matches := InterpolationRe.FindAllStringSubmatch(s, -1)
		for _, m := range matches {
			refPath := m[1]
			if refPath == resourcePath || strings.HasPrefix(refPath, resourcePath+".") {
				refs = append(refs, InterpolationReference{
					Path:     p.String(),
					Location: v.Location(),
					RefStr:   m[0],
				})
			}
		}
		return v, nil
	})

	return refs
}

// DynLocationToLSPLocation converts a 1-based dyn.Location to a 0-based LSPLocation.
func DynLocationToLSPLocation(loc dyn.Location) LSPLocation {
	line := max(loc.Line-1, 0)
	col := max(loc.Column-1, 0)
	return LSPLocation{
		URI: PathToURI(loc.File),
		Range: Range{
			Start: Position{Line: line, Character: col},
			End:   Position{Line: line, Character: col},
		},
	}
}
