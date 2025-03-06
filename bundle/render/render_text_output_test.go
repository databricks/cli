package render

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
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
			name: "nil bundle and 1 recommendation",
			diags: diag.Diagnostics{
				{
					Severity: diag.Recommendation,
					Summary:  "recommendation",
				},
			},
			opts: RenderOptions{RenderSummaryTable: true},
			expected: "Recommendation: recommendation\n" +
				"\n" +
				"Found 1 recommendation\n",
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
			name:   "bundle during 'load' and 2 errors, 1 warning and 1 recommendation with details",
			bundle: loadingBundle,
			diags: diag.Diagnostics{
				diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "error (1)",
					Detail:    "detail (1)",
					Locations: []dyn.Location{{File: "foo.py", Line: 1, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "error (2)",
					Detail:    "detail (2)",
					Locations: []dyn.Location{{File: "foo.py", Line: 2, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "warning (3)",
					Detail:    "detail (3)",
					Locations: []dyn.Location{{File: "foo.py", Line: 3, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Recommendation,
					Summary:   "recommendation (4)",
					Detail:    "detail (4)",
					Locations: []dyn.Location{{File: "foo.py", Line: 4, Column: 1}},
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
				"Recommendation: recommendation (4)\n" +
				"  in foo.py:4:1\n" +
				"\n" +
				"detail (4)\n" +
				"\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"\n" +
				"Found 2 errors, 1 warning, and 1 recommendation\n",
		},
		{
			name:   "bundle during 'load' and 1 error and 1 warning",
			bundle: loadingBundle,
			diags: diag.Diagnostics{
				diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "error (1)",
					Detail:    "detail (1)",
					Locations: []dyn.Location{{File: "foo.py", Line: 1, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "warning (2)",
					Detail:    "detail (2)",
					Locations: []dyn.Location{{File: "foo.py", Line: 2, Column: 1}},
				},
			},
			opts: RenderOptions{RenderSummaryTable: true},
			expected: "Error: error (1)\n" +
				"  in foo.py:1:1\n" +
				"\n" +
				"detail (1)\n" +
				"\n" +
				"Warning: warning (2)\n" +
				"  in foo.py:2:1\n" +
				"\n" +
				"detail (2)\n" +
				"\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"\n" +
				"Found 1 error and 1 warning\n",
		},
		{
			name:   "bundle during 'load' and 1 errors, 2 warning and 2 recommendations with details",
			bundle: loadingBundle,
			diags: diag.Diagnostics{
				diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "error (1)",
					Detail:    "detail (1)",
					Locations: []dyn.Location{{File: "foo.py", Line: 1, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "warning (2)",
					Detail:    "detail (2)",
					Locations: []dyn.Location{{File: "foo.py", Line: 2, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "warning (3)",
					Detail:    "detail (3)",
					Locations: []dyn.Location{{File: "foo.py", Line: 3, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Recommendation,
					Summary:   "recommendation (4)",
					Detail:    "detail (4)",
					Locations: []dyn.Location{{File: "foo.py", Line: 4, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Recommendation,
					Summary:   "recommendation (5)",
					Detail:    "detail (5)",
					Locations: []dyn.Location{{File: "foo.py", Line: 5, Column: 1}},
				},
			},
			opts: RenderOptions{RenderSummaryTable: true},
			expected: "Error: error (1)\n" +
				"  in foo.py:1:1\n" +
				"\n" +
				"detail (1)\n" +
				"\n" +
				"Warning: warning (2)\n" +
				"  in foo.py:2:1\n" +
				"\n" +
				"detail (2)\n" +
				"\n" +
				"Warning: warning (3)\n" +
				"  in foo.py:3:1\n" +
				"\n" +
				"detail (3)\n" +
				"\n" +
				"Recommendation: recommendation (4)\n" +
				"  in foo.py:4:1\n" +
				"\n" +
				"detail (4)\n" +
				"\n" +
				"Recommendation: recommendation (5)\n" +
				"  in foo.py:5:1\n" +
				"\n" +
				"detail (5)\n" +
				"\n" +
				"Name: test-bundle\n" +
				"Target: test-target\n" +
				"\n" +
				"Found 1 error, 2 warnings, and 2 recommendations\n",
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
			name:   "nil bundle without summary with 1 error, 1 warning and 1 recommendation",
			bundle: nil,
			diags: diag.Diagnostics{
				diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "error (1)",
					Detail:    "detail (1)",
					Locations: []dyn.Location{{File: "foo.py", Line: 1, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   "warning (2)",
					Detail:    "detail (2)",
					Locations: []dyn.Location{{File: "foo.py", Line: 3, Column: 1}},
				},
				diag.Diagnostic{
					Severity:  diag.Recommendation,
					Summary:   "recommendation (3)",
					Detail:    "detail (3)",
					Locations: []dyn.Location{{File: "foo.py", Line: 5, Column: 1}},
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
				"\n" +
				"Recommendation: recommendation (3)\n" +
				"  in foo.py:5:1\n" +
				"\n" +
				"detail (3)\n" +
				"\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &bytes.Buffer{}

			err := RenderDiagnostics(writer, tc.bundle, tc.diags, tc.opts)
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
					Locations: []dyn.Location{{
						File:   "foo.yaml",
						Line:   1,
						Column: 2,
					}},
				},
			},
			expected: "Error: failed to load xxx\n" +
				"  in foo.yaml:1:2\n\n" +
				"'name' is required\n\n",
		},
		{
			name: "error with multiple source locations",
			diags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "failed to load xxx",
					Detail:   "'name' is required",
					Locations: []dyn.Location{
						{
							File:   "foo.yaml",
							Line:   1,
							Column: 2,
						},
						{
							File:   "bar.yaml",
							Line:   3,
							Column: 4,
						},
					},
				},
			},
			expected: "Error: failed to load xxx\n" +
				"  in foo.yaml:1:2\n" +
				"     bar.yaml:3:4\n\n" +
				"'name' is required\n\n",
		},
		{
			name: "error with path",
			diags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Detail:   "'name' is required",
					Summary:  "failed to load xxx",
					Paths:    []dyn.Path{dyn.MustPathFromString("resources.jobs.xxx")},
				},
			},
			expected: "Error: failed to load xxx\n" +
				"  at resources.jobs.xxx\n" +
				"\n" +
				"'name' is required\n\n",
		},
		{
			name: "error with multiple paths",
			diags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Detail:   "'name' is required",
					Summary:  "failed to load xxx",
					Paths: []dyn.Path{
						dyn.MustPathFromString("resources.jobs.xxx"),
						dyn.MustPathFromString("resources.jobs.yyy"),
						dyn.MustPathFromString("resources.jobs.zzz"),
					},
				},
			},
			expected: "Error: failed to load xxx\n" +
				"  at resources.jobs.xxx\n" +
				"     resources.jobs.yyy\n" +
				"     resources.jobs.zzz\n" +
				"\n" +
				"'name' is required\n\n",
		},
		{
			name: "recommendation with multiple paths and locations",
			diags: diag.Diagnostics{
				{
					Severity: diag.Recommendation,
					Summary:  "summary",
					Detail:   "detail",
					Paths: []dyn.Path{
						dyn.MustPathFromString("resources.jobs.xxx"),
						dyn.MustPathFromString("resources.jobs.yyy"),
					},
					Locations: []dyn.Location{
						{File: "foo.yaml", Line: 1, Column: 2},
						{File: "bar.yaml", Line: 3, Column: 4},
					},
				},
			},
			expected: "Recommendation: summary\n" +
				"  at resources.jobs.xxx\n" +
				"     resources.jobs.yyy\n" +
				"  in foo.yaml:1:2\n" +
				"     bar.yaml:3:4\n\n" +
				"detail\n\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &bytes.Buffer{}

			err := renderDiagnosticsOnly(writer, bundle, tc.diags)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, writer.String())
		})
	}
}

func TestRenderSummaryTemplate_nilBundle(t *testing.T) {
	writer := &bytes.Buffer{}

	err := renderSummaryHeaderTemplate(writer, nil)
	require.NoError(t, err)

	_, err = io.WriteString(writer, buildTrailer(nil))
	require.NoError(t, err)

	assert.Equal(t, "Validation OK!\n", writer.String())
}

func TestRenderSummary(t *testing.T) {
	ctx := context.Background()

	// Create a mock bundle with various resources
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name:   "test-bundle",
				Target: "test-target",
			},
			Workspace: config.Workspace{
				Host: "https://mycompany.databricks.com/",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						ID:          "1",
						URL:         "https://url1",
						JobSettings: &jobs.JobSettings{Name: "job1-name"},
					},
					"job2": {
						ID:          "2",
						URL:         "https://url2",
						JobSettings: &jobs.JobSettings{Name: "job2-name"},
					},
					"job3": {
						ID:  "3",
						URL: "https://url3", // This emulates deleted job
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline2": {
						ID: "4",
						// no URL
						CreatePipeline: &pipelines.CreatePipeline{Name: "pipeline2-name"},
					},
					"pipeline1": {
						ID:             "3",
						URL:            "https://url3",
						CreatePipeline: &pipelines.CreatePipeline{Name: "pipeline1-name"},
					},
				},
				Schemas: map[string]*resources.Schema{
					"schema1": {
						ID: "catalog.schema",
						CreateSchema: &catalog.CreateSchema{
							Name: "schema",
						},
						// no URL
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint1": {
						ID: "7",
						CreateServingEndpoint: &serving.CreateServingEndpoint{
							Name: "my_serving_endpoint",
						},
						URL: "https://url4",
					},
				},
			},
		},
	}

	writer := &bytes.Buffer{}
	err := RenderSummary(ctx, writer, b)
	require.NoError(t, err)

	expectedSummary := `Name: test-bundle
Target: test-target
Workspace:
  Host: https://mycompany.databricks.com/
Resources:
  Jobs:
    job1:
      Name: job1-name
      URL:  https://url1
    job2:
      Name: job2-name
      URL:  https://url2
  Model Serving Endpoints:
    endpoint1:
      Name: my_serving_endpoint
      URL:  https://url4
  Pipelines:
    pipeline1:
      Name: pipeline1-name
      URL:  https://url3
    pipeline2:
      Name: pipeline2-name
      URL:  (not deployed)
  Schemas:
    schema1:
      Name: schema
      URL:  (not deployed)
`
	assert.Equal(t, expectedSummary, writer.String())
}
