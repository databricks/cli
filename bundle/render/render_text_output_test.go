package render

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSummaryHeaderTemplate_nilBundle(t *testing.T) {
	writer := &bytes.Buffer{}

	err := renderSummaryHeaderTemplate(context.Background(), writer, nil)
	require.NoError(t, err)

	assert.Equal(t, "", writer.String())
}

func TestRenderDiagnosticsSummary(t *testing.T) {
	// Disable colors for consistent test output
	oldNoColor := color.NoColor
	color.NoColor = true
	defer func() {
		color.NoColor = oldNoColor
	}()

	testCases := []struct {
		name            string
		bundle          *bundle.Bundle
		errors          int
		warnings        int
		recommendations int
		expectedSummary string
	}{
		{
			name:            "no diagnostics",
			bundle:          nil,
			errors:          0,
			warnings:        0,
			recommendations: 0,
			expectedSummary: "Validation OK!\n",
		},
		{
			name:            "1 error",
			bundle:          nil,
			errors:          1,
			warnings:        0,
			recommendations: 0,
			expectedSummary: "Found 1 error\n",
		},
		{
			name:            "1 warning",
			bundle:          nil,
			errors:          0,
			warnings:        1,
			recommendations: 0,
			expectedSummary: "Found 1 warning\n",
		},
		{
			name:            "1 recommendation",
			bundle:          nil,
			errors:          0,
			warnings:        0,
			recommendations: 1,
			expectedSummary: "Found 1 recommendation\n",
		},
		{
			name:            "multiple diagnostics",
			bundle:          nil,
			errors:          2,
			warnings:        1,
			recommendations: 1,
			expectedSummary: "Found 2 errors, 1 warning, and 1 recommendation\n",
		},
		{
			name: "with bundle info",
			bundle: &bundle.Bundle{
				Config: config.Root{
					Bundle: config.Bundle{
						Name:   "test-bundle",
						Target: "test-target",
					},
					Workspace: config.Workspace{
						Host:     "https://test.databricks.com",
						RootPath: "/Users/test@test.com/.bundle/test-bundle/test-target",
						CurrentUser: &config.User{
							User: &iam.User{UserName: "test@test.com"},
						},
					},
				},
			},
			errors:          1,
			warnings:        0,
			recommendations: 0,
			expectedSummary: "Name: test-bundle\nTarget: test-target\nWorkspace:\n  Host: https://test.databricks.com\n  User: test@test.com\n  Path: /Users/test@test.com/.bundle/test-bundle/test-target\n\nFound 1 error\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logdiag.InitContext(context.Background())
			logdiag.SetCollect(ctx, true) // Collect diagnostics instead of outputting to stderr

			// Simulate diagnostic counts by logging fake diagnostics
			for range tc.errors {
				logdiag.LogError(ctx, errors.New("test error"))
			}
			for range tc.warnings {
				logdiag.LogDiag(ctx, diag.Diagnostic{Severity: diag.Warning, Summary: "test warning"})
			}
			for range tc.recommendations {
				logdiag.LogDiag(ctx, diag.Diagnostic{Severity: diag.Recommendation, Summary: "test recommendation"})
			}

			writer := &bytes.Buffer{}
			err := RenderDiagnosticsSummary(ctx, writer, tc.bundle)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedSummary, writer.String())
		})
	}
}

type renderDiagnosticsTestCase struct {
	name     string
	diags    diag.Diagnostics
	expected string
}

func TestRenderDiagnostics(t *testing.T) {
	// Disable colors for consistent test output
	oldNoColor := color.NoColor
	color.NoColor = true
	defer func() {
		color.NoColor = oldNoColor
	}()

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

			err := cmdio.RenderDiagnostics(writer, tc.diags)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, writer.String())
		})
	}
}

func TestRenderSummaryTemplate_nilBundle(t *testing.T) {
	// Disable colors for consistent test output
	oldNoColor := color.NoColor
	color.NoColor = true
	defer func() {
		color.NoColor = oldNoColor
	}()

	ctx := logdiag.InitContext(context.Background())
	writer := &bytes.Buffer{}

	err := renderSummaryHeaderTemplate(ctx, writer, nil)
	require.NoError(t, err)

	_, err = io.WriteString(writer, buildTrailer(ctx))
	require.NoError(t, err)

	assert.Equal(t, "Validation OK!\n", writer.String())
}

func TestRenderSummary(t *testing.T) {
	ctx := context.Background()

	// Disable colors for consistent test output
	oldNoColor := color.NoColor
	color.NoColor = true
	defer func() {
		color.NoColor = oldNoColor
	}()

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
						JobSettings: jobs.JobSettings{Name: "job1-name"},
					},
					"job2": {
						ID:          "2",
						URL:         "https://url2",
						JobSettings: jobs.JobSettings{Name: "job2-name"},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline2": {
						ID: "4",
						// no URL
						CreatePipeline: pipelines.CreatePipeline{Name: "pipeline2-name"},
					},
					"pipeline1": {
						ID:             "3",
						URL:            "https://url3",
						CreatePipeline: pipelines.CreatePipeline{Name: "pipeline1-name"},
					},
				},
				Schemas: map[string]*resources.Schema{
					"schema1": {
						ID: "catalog.schema",
						CreateSchema: catalog.CreateSchema{
							Name: "schema",
						},
						// no URL
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint1": {
						ID: "7",
						CreateServingEndpoint: serving.CreateServingEndpoint{
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
