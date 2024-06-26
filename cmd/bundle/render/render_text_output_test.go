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
			name:   "bundle during 'load' and 1 error",
			bundle: loadingBundle,
			diags:  diag.Errorf("failed to load bundle"),
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
			expected: "Warning: failed to load bundle\n" +
				"\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"\n" +
				"Found 1 error\n",
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
			expected: "\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"Workspace:\n" +
				"  Host: https://localhost/\n" +
				"  User: test-user\n" +
				"  Path: /Users/test-user@databricks.com/.bundle/examples/test-target\n" +
				"\n" +
				"Validation OK!\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &bytes.Buffer{}

			err := RenderTextOutput(writer, tc.bundle, tc.diags)
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
				"  in foo.yaml:1:2\n" +
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
