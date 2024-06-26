package python

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type pythonDiagnostic struct {
	Severity pythonSeverity `json:"severity"`
	Summary  string         `json:"summary"`
	Detail   string         `json:"detail,omitempty"`
	Location string         `json:"location,omitempty"`
	Path     string         `json:"path,omitempty"`
}

type pythonSeverity = string

var locationRegex = regexp.MustCompile(`^(.*):(\d+):(\d+)$`)

const (
	pythonError   pythonSeverity = "error"
	pythonWarning pythonSeverity = "warning"
)

// parsePythonDiagnostics parses diagnostics from the Python mutator.
//
// diagnostics file is newline-separated JSON objects with pythonDiagnostic structure.
func parsePythonDiagnostics(input io.Reader) (diag.Diagnostics, error) {
	// the default limit is 64 Kb which should be enough for diagnostics
	scanner := bufio.NewScanner(input)
	diagnostics := diag.Diagnostics{}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		parsedLine := pythonDiagnostic{}

		err := json.Unmarshal([]byte(line), &parsedLine)

		if err != nil {
			return nil, fmt.Errorf("failed to parse diagnostics: %s", err)
		}

		severity, err := convertPythonSeverity(parsedLine.Severity)
		if err != nil {
			return nil, fmt.Errorf("failed to parse severity: %s", err)
		}

		location, err := convertPythonLocation(parsedLine.Location)
		if err != nil {
			return nil, fmt.Errorf("failed to parse location: %s", err)
		}

		path, err := convertPythonPath(parsedLine.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse path: %s", err)
		}

		diagnostic := diag.Diagnostic{
			Severity: severity,
			Summary:  parsedLine.Summary,
			Detail:   parsedLine.Detail,
			Location: location,
			Path:     path,
		}

		diagnostics = diagnostics.Append(diagnostic)
	}

	return diagnostics, nil
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

func convertPythonLocation(location string) (dyn.Location, error) {
	if location == "" {
		return dyn.Location{}, nil
	}

	matches := locationRegex.FindStringSubmatch(location)

	if len(matches) == 4 {
		line, err := strconv.Atoi(matches[2])
		if err != nil {
			return dyn.Location{}, fmt.Errorf("failed to parse line number: %s", location)
		}

		column, err := strconv.Atoi(matches[3])
		if err != nil {
			return dyn.Location{}, fmt.Errorf("failed to parse column number: %s", location)
		}

		return dyn.Location{
			File:   matches[1],
			Line:   line,
			Column: column,
		}, nil
	}

	return dyn.Location{}, fmt.Errorf("failed to parse location: %s", location)
}
