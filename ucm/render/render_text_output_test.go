package render

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSummaryHeaderTemplate_nilUcm(t *testing.T) {
	writer := &bytes.Buffer{}

	err := renderSummaryHeaderTemplate(t.Context(), writer, nil)
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
		ucm             *ucm.Ucm
		errors          int
		warnings        int
		recommendations int
		expectedSummary string
	}{
		{
			name:            "no diagnostics",
			ucm:             nil,
			errors:          0,
			warnings:        0,
			recommendations: 0,
			expectedSummary: "Validation OK!\n",
		},
		{
			name:            "1 error",
			ucm:             nil,
			errors:          1,
			warnings:        0,
			recommendations: 0,
			expectedSummary: "Found 1 error\n",
		},
		{
			name:            "1 warning",
			ucm:             nil,
			errors:          0,
			warnings:        1,
			recommendations: 0,
			expectedSummary: "Found 1 warning\n",
		},
		{
			name:            "1 recommendation",
			ucm:             nil,
			errors:          0,
			warnings:        0,
			recommendations: 1,
			expectedSummary: "Found 1 recommendation\n",
		},
		{
			name:            "multiple diagnostics",
			ucm:             nil,
			errors:          2,
			warnings:        1,
			recommendations: 1,
			expectedSummary: "Found 2 errors, 1 warning, and 1 recommendation\n",
		},
		{
			name: "with ucm info",
			ucm: &ucm.Ucm{
				Config: config.Root{
					Ucm: config.Ucm{
						Name:   "test-ucm",
						Target: "test-target",
					},
					Workspace: config.Workspace{
						Host:     "https://test.databricks.com",
						RootPath: "/Users/test@test.com/.ucm/test-ucm/test-target",
					},
				},
				CurrentUser: &config.User{
					User: &iam.User{UserName: "test@test.com"},
				},
			},
			errors:          1,
			warnings:        0,
			recommendations: 0,
			expectedSummary: "Name: test-ucm\nTarget: test-target\nWorkspace:\n  Host: https://test.databricks.com\n  User: test@test.com\n  Path: /Users/test@test.com/.ucm/test-ucm/test-target\n\nFound 1 error\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logdiag.InitContext(t.Context())
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
			err := RenderDiagnosticsSummary(ctx, writer, tc.ucm)
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
					Paths:    []dyn.Path{dyn.MustPathFromString("resources.catalogs.xxx")},
				},
			},
			expected: "Error: failed to load xxx\n" +
				"  at resources.catalogs.xxx\n" +
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
						dyn.MustPathFromString("resources.catalogs.xxx"),
						dyn.MustPathFromString("resources.catalogs.yyy"),
						dyn.MustPathFromString("resources.catalogs.zzz"),
					},
				},
			},
			expected: "Error: failed to load xxx\n" +
				"  at resources.catalogs.xxx\n" +
				"     resources.catalogs.yyy\n" +
				"     resources.catalogs.zzz\n" +
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
						dyn.MustPathFromString("resources.catalogs.xxx"),
						dyn.MustPathFromString("resources.catalogs.yyy"),
					},
					Locations: []dyn.Location{
						{File: "foo.yaml", Line: 1, Column: 2},
						{File: "bar.yaml", Line: 3, Column: 4},
					},
				},
			},
			expected: "Recommendation: summary\n" +
				"  at resources.catalogs.xxx\n" +
				"     resources.catalogs.yyy\n" +
				"  in foo.yaml:1:2\n" +
				"     bar.yaml:3:4\n\n" +
				"detail\n\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

			err := cmdio.RenderDiagnostics(ctx, tc.diags)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, stderr.String())
		})
	}
}

func TestRenderSummaryTemplate_nilUcm(t *testing.T) {
	// Disable colors for consistent test output
	oldNoColor := color.NoColor
	color.NoColor = true
	defer func() {
		color.NoColor = oldNoColor
	}()

	ctx := logdiag.InitContext(t.Context())
	writer := &bytes.Buffer{}

	err := renderSummaryHeaderTemplate(ctx, writer, nil)
	require.NoError(t, err)

	_, err = io.WriteString(writer, buildTrailer(ctx))
	require.NoError(t, err)

	assert.Equal(t, "Validation OK!\n", writer.String())
}

func TestRenderSummary(t *testing.T) {
	ctx := t.Context()

	// Disable colors for consistent test output
	oldNoColor := color.NoColor
	color.NoColor = true
	defer func() {
		color.NoColor = oldNoColor
	}()

	// Create a mock ucm with various resources
	u := &ucm.Ucm{
		Config: config.Root{
			Ucm: config.Ucm{
				Name:   "test-ucm",
				Target: "test-target",
			},
			Workspace: config.Workspace{
				Host: "https://mycompany.databricks.com/",
			},
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{
					"catalog1": {
						CreateCatalog: catalog.CreateCatalog{Name: "catalog1-name"},
						URL:           "https://url1",
					},
					"catalog2": {
						CreateCatalog: catalog.CreateCatalog{Name: "catalog2-name"},
						URL:           "https://url2",
					},
				},
				Schemas: map[string]*resources.Schema{
					"schema2": {
						CreateSchema: catalog.CreateSchema{Name: "schema", CatalogName: "catalog1-name"},
						// no URL
					},
					"schema1": {
						CreateSchema: catalog.CreateSchema{Name: "schema", CatalogName: "catalog2-name"},
						URL:          "https://url3",
					},
				},
			},
		},
	}

	writer := &bytes.Buffer{}
	err := RenderSummary(ctx, writer, u)
	require.NoError(t, err)

	expectedSummary := `Name: test-ucm
Target: test-target
Workspace:
  Host: https://mycompany.databricks.com/
Resources:
  Catalogs:
    catalog1:
      Name: catalog1-name
      URL:  https://url1
    catalog2:
      Name: catalog2-name
      URL:  https://url2
  Schemas:
    schema1:
      Name: catalog2-name.schema
      URL:  https://url3
    schema2:
      Name: catalog1-name.schema
      URL:  (not deployed)
`
	assert.Equal(t, expectedSummary, writer.String())
}
