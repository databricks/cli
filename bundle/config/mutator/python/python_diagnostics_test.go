package python

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestConvertPythonLocation(t *testing.T) {
	location := convertPythonLocation(pythonDiagnosticLocation{
		File:   "src/examples/file.py",
		Line:   1,
		Column: 2,
	})

	assert.Equal(t, dyn.Location{
		File:   "src/examples/file.py",
		Line:   1,
		Column: 2,
	}, location)
}

type parsePythonDiagnosticsTest struct {
	name     string
	input    string
	expected diag.Diagnostics
}

func TestParsePythonDiagnostics(t *testing.T) {
	testCases := []parsePythonDiagnosticsTest{
		{
			name:  "short error with location",
			input: `{"severity": "error", "summary": "error summary", "location": {"file": "src/examples/file.py", "line": 1, "column": 2}}`,
			expected: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "error summary",
					Locations: []dyn.Location{
						{
							File:   "src/examples/file.py",
							Line:   1,
							Column: 2,
						},
					},
				},
			},
		},
		{
			name:  "short error with path",
			input: `{"severity": "error", "summary": "error summary", "path": "resources.jobs.job0.name"}`,
			expected: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "error summary",
					Paths:    []dyn.Path{dyn.MustPathFromString("resources.jobs.job0.name")},
				},
			},
		},
		{
			name:     "empty file",
			input:    "",
			expected: diag.Diagnostics{},
		},
		{
			name:     "newline file",
			input:    "\n",
			expected: diag.Diagnostics{},
		},
		{
			name:  "warning with detail",
			input: `{"severity": "warning", "summary": "warning summary", "detail": "warning detail"}`,
			expected: diag.Diagnostics{
				{
					Severity: diag.Warning,
					Summary:  "warning summary",
					Detail:   "warning detail",
				},
			},
		},
		{
			name: "multiple errors",
			input: `{"severity": "error", "summary": "error summary (1)"}` + "\n" +
				`{"severity": "error", "summary": "error summary (2)"}`,
			expected: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "error summary (1)",
				},
				{
					Severity: diag.Error,
					Summary:  "error summary (2)",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diags, err := parsePythonDiagnostics(bytes.NewReader([]byte(tc.input)))

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, diags)
		})
	}
}
