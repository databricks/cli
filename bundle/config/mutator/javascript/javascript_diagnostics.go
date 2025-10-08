package javascript

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// javaScriptDiagnostic is a single entry in diagnostics.json
type javaScriptDiagnostic struct {
	Severity javaScriptSeverity           `json:"severity"`
	Summary  string                       `json:"summary"`
	Detail   string                       `json:"detail,omitempty"`
	Location javaScriptDiagnosticLocation `json:"location,omitempty"`
	Path     string                       `json:"path,omitempty"`
}

type javaScriptDiagnosticLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type javaScriptSeverity = string

const (
	javaScriptError   javaScriptSeverity = "error"
	javaScriptWarning javaScriptSeverity = "warning"
)

// parseJavaScriptDiagnostics parses diagnostics from the JavaScript mutator.
//
// diagnostics file is newline-separated JSON objects with javaScriptDiagnostic structure.
func parseJavaScriptDiagnostics(input io.Reader) (diag.Diagnostics, error) {
	diags := diag.Diagnostics{}
	decoder := json.NewDecoder(input)

	for decoder.More() {
		var parsedLine javaScriptDiagnostic

		err := decoder.Decode(&parsedLine)
		if err != nil {
			return nil, fmt.Errorf("failed to parse diags: %s", err)
		}

		severity, err := convertJavaScriptSeverity(parsedLine.Severity)
		if err != nil {
			return nil, fmt.Errorf("failed to parse severity: %s", err)
		}

		path, err := convertJavaScriptPath(parsedLine.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse path: %s", err)
		}
		var paths []dyn.Path
		if path != nil {
			paths = []dyn.Path{path}
		}

		var locations []dyn.Location
		location := convertJavaScriptLocation(parsedLine.Location)
		if location != (dyn.Location{}) {
			locations = append(locations, location)
		}

		diag := diag.Diagnostic{
			Severity:  severity,
			Summary:   parsedLine.Summary,
			Detail:    parsedLine.Detail,
			Locations: locations,
			Paths:     paths,
		}

		diags = diags.Append(diag)
	}

	return diags, nil
}

func convertJavaScriptPath(path string) (dyn.Path, error) {
	if path == "" {
		return nil, nil
	}

	return dyn.NewPathFromString(path)
}

func convertJavaScriptSeverity(severity javaScriptSeverity) (diag.Severity, error) {
	switch severity {
	case javaScriptError:
		return diag.Error, nil
	case javaScriptWarning:
		return diag.Warning, nil
	default:
		return 0, fmt.Errorf("unexpected value: %s", severity)
	}
}

func convertJavaScriptLocation(location javaScriptDiagnosticLocation) dyn.Location {
	return dyn.Location{
		File:   location.File,
		Line:   location.Line,
		Column: location.Column,
	}
}
