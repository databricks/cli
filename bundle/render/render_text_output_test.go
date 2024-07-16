package render

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

type renderTestOutputTestCase struct {
	name     string
	bundle   *bundle.Bundle
	diags    diag.Diagnostics
	opts     RenderOptions
	expected string
}

func TestRenderTextOutput(t *testing.T) {
	loadingBundle := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name:   "test-bundle",
				Target: "test-target",
			},
		},
	}

	testCases := []renderTestOutputTestCase{
		{
			name: "nil bundle and 1 error",
			diags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "failed to load xxx",
				},
			},
			opts: RenderOptions{RenderSummaryTable: true},
			expected: "Error: failed to load xxx\n" +
				"\n" +
				"Found 1 error\n",
		},
		{
			name:   "bundle during 'load' and 1 error",
			bundle: loadingBundle,
			diags:  diag.Errorf("failed to load bundle"),
			opts:   RenderOptions{RenderSummaryTable: true},
			expected: "Error: failed to load bundle\n" +
				"\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"\n" +
				"Found 1 error\n",
		},
		{
			name:   "bundle during 'load' and 1 warning",
			bundle: loadingBundle,
			diags:  diag.Warningf("failed to load bundle"),
			opts:   RenderOptions{RenderSummaryTable: true},
			expected: "Warning: failed to load bundle\n" +
				"\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"\n" +
				"Found 1 warning\n",
		},
		{
			name:   "bundle during 'load' and 2 warnings",
			bundle: loadingBundle,
			diags:  diag.Warningf("warning (1)").Extend(diag.Warningf("warning (2)")),
			opts:   RenderOptions{RenderSummaryTable: true},
			expected: "Warning: warning (1)\n" +
				"\n" +
				"Warning: warning (2)\n" +
				"\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"\n" +
				"Found 2 warnings\n",
		},
		{
			name:   "bundle during 'load' and 2 errors, 1 warning with details",
			bundle: loadingBundle,
			diags: diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "error (1)",
					Detail:   "detail (1)",
					Location: dyn.Location{
						File:   "foo.py",
						Line:   1,
						Column: 1,
					},
				},
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "error (2)",
					Detail:   "detail (2)",
					Location: dyn.Location{
						File:   "foo.py",
						Line:   2,
						Column: 1,
					},
				},
				diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "warning (3)",
					Detail:   "detail (3)",
					Location: dyn.Location{
						File:   "foo.py",
						Line:   3,
						Column: 1,
					},
				},
			},
			opts: RenderOptions{RenderSummaryTable: true},
			expected: "Error: error (1)\n" +
				"  in foo.py:1:1\n" +
				"\n" +
				"detail (1)\n" +
				"\n" +
				"Error: error (2)\n" +
				"  in foo.py:2:1\n" +
				"\n" +
				"detail (2)\n" +
				"\n" +
				"Warning: warning (3)\n" +
				"  in foo.py:3:1\n" +
				"\n" +
				"detail (3)\n" +
				"\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"\n" +
				"Found 2 errors and 1 warning\n",
		},
		{
			name: "bundle during 'init'",
			bundle: &bundle.Bundle{
				Config: config.Root{
					Bundle: config.Bundle{
						Name:   "test-bundle",
						Target: "test-target",
					},
					Workspace: config.Workspace{
						Host: "https://localhost/",
						CurrentUser: &config.User{
							User: &iam.User{
								UserName: "test-user",
							},
						},
						RootPath: "/Users/test-user@databricks.com/.bundle/examples/test-target",
					},
				},
			},
			diags: nil,
			opts:  RenderOptions{RenderSummaryTable: true},
			expected: "Name: test-bundle\n" +
				"Target: test-target\n" +
				"Workspace:\n" +
				"  Host: https://localhost/\n" +
				"  User: test-user\n" +
				"  Path: /Users/test-user@databricks.com/.bundle/examples/test-target\n" +
				"\n" +
				"Validation OK!\n",
		},
		{
			name:   "nil bundle without summary with 1 error and 1 warning",
			bundle: nil,
			diags: diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "error (1)",
					Detail:   "detail (1)",
					Location: dyn.Location{
						File:   "foo.py",
						Line:   1,
						Column: 1,
					},
				},
				diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "warning (2)",
					Detail:   "detail (2)",
					Location: dyn.Location{
						File:   "foo.py",
						Line:   3,
						Column: 1,
					},
				},
			},
			opts: RenderOptions{RenderSummaryTable: false},
			expected: "Error: error (1)\n" +
				"  in foo.py:1:1\n" +
				"\n" +
				"detail (1)\n" +
				"\n" +
				"Warning: warning (2)\n" +
				"  in foo.py:3:1\n" +
				"\n" +
				"detail (2)\n" +
				"\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &bytes.Buffer{}

			err := RenderTextOutput(writer, tc.bundle, tc.diags, tc.opts)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, writer.String())
		})
	}
}

type renderDiagnosticsTestCase struct {
	name     string
	diags    diag.Diagnostics
	expected string
}

func TestRenderDiagnostics(t *testing.T) {
	bundle := &bundle.Bundle{}

	testCases := []renderDiagnosticsTestCase{
		{
			name:     "empty diagnostics",
			diags:    diag.Diagnostics{},
			expected: "",
		},
		{
			name: "error with short summary",
			diags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "failed to load xxx",
				},
			},
			expected: "Error: failed to load xxx\n\n",
		},
		{
			name: "error with source location",
			diags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "failed to load xxx",
					Detail:   "'name' is required",
					Location: dyn.Location{
						File:   "foo.yaml",
						Line:   1,
						Column: 2,
					},
				},
			},
			expected: "Error: failed to load xxx\n" +
				"  in foo.yaml:1:2\n\n" +
				"'name' is required\n\n",
		},
		{
			name: "error with path",
			diags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Detail:   "'name' is required",
					Summary:  "failed to load xxx",
					Path:     dyn.MustPathFromString("resources.jobs.xxx"),
				},
			},
			expected: "Error: failed to load xxx\n" +
				"  at resources.jobs.xxx\n" +
				"\n" +
				"'name' is required\n\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &bytes.Buffer{}

			err := renderDiagnostics(writer, bundle, tc.diags)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, writer.String())
		})
	}
}

func TestRenderSummaryTemplate_nilBundle(t *testing.T) {
	writer := &bytes.Buffer{}

	err := renderSummaryTemplate(writer, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "Validation OK!\n", writer.String())
}
