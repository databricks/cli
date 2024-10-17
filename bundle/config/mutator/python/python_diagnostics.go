package python

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type pythonDiagnostic struct {
	Severity pythonSeverity           `json:"severity"`
	Summary  string                   `json:"summary"`
	Detail   string                   `json:"detail,omitempty"`
	Location pythonDiagnosticLocation `json:"location,omitempty"`
	Path     string                   `json:"path,omitempty"`
}

type pythonDiagnosticLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type pythonSeverity = string

const (
	pythonError   pythonSeverity = "error"
	pythonWarning pythonSeverity = "warning"
)

// parsePythonDiagnostics parses diagnostics from the Python mutator.
//
// diagnostics file is newline-separated JSON objects with pythonDiagnostic structure.
func parsePythonDiagnostics(input io.Reader) (diag.Diagnostics, error) {
	diags := diag.Diagnostics{}
	decoder := json.NewDecoder(input)

	for decoder.More() {
		var parsedLine pythonDiagnostic

		err := decoder.Decode(&parsedLine)
		if err != nil {
			return nil, fmt.Errorf("failed to parse diags: %s", err)
		}

		severity, err := convertPythonSeverity(parsedLine.Severity)
		if err != nil {
			return nil, fmt.Errorf("failed to parse severity: %s", err)
		}

		path, err := convertPythonPath(parsedLine.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse path: %s", err)
		}

		location := convertPythonLocation(parsedLine.Location)

		// TODO: test this.
		locationPathPairs := []diag.LocationPathPair{}
		if path != nil || location != (dyn.Location{}) {
			locationPathPairs = append(locationPathPairs, diag.LocationPathPair{
				L: location,
				P: path,
			})
		}

		diag := diag.Diagnostic{
			Severity:          severity,
			Summary:           parsedLine.Summary,
			Detail:            parsedLine.Detail,
			LocationPathPairs: locationPathPairs,
		}

		diags = diags.Append(diag)
	}

	return diags, nil
}

func convertPythonPath(path string) (dyn.Path, error) {
	if path == "" {
		return nil, nil
	}

	return dyn.NewPathFromString(path)
}

func convertPythonSeverity(severity pythonSeverity) (diag.Severity, error) {
	switch severity {
	case pythonError:
		return diag.Error, nil
	case pythonWarning:
		return diag.Warning, nil
	default:
		return 0, fmt.Errorf("unexpected value: %s", severity)
	}
}

func convertPythonLocation(location pythonDiagnosticLocation) dyn.Location {
	return dyn.Location{
		File:   location.File,
		Line:   location.Line,
		Column: location.Column,
	}
}
