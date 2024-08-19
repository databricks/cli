package config_tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestUndefinedResourcesLoadWithError(t *testing.T) {
	b := load(t, "./undefined_resources")
	diags := bundle.Apply(context.Background(), b, validate.AllResourcesHaveValues())

	assert.Len(t, diags, 3)
	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "job undefined-job is not defined",
		Locations: []dyn.Location{{
			File:   filepath.FromSlash("undefined_resources/databricks.yml"),
			Line:   6,
			Column: 19,
		}},
		Paths: []dyn.Path{dyn.MustPathFromString("resources.jobs.undefined-job")},
	})
	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "experiment undefined-experiment is not defined",
		Locations: []dyn.Location{{
			File:   filepath.FromSlash("undefined_resources/databricks.yml"),
			Line:   11,
			Column: 26,
		}},
		Paths: []dyn.Path{dyn.MustPathFromString("resources.experiments.undefined-experiment")},
	})
	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "pipeline undefined-pipeline is not defined",
		Locations: []dyn.Location{{
			File:   filepath.FromSlash("undefined_resources/databricks.yml"),
			Line:   14,
			Column: 24,
		}},
		Paths: []dyn.Path{dyn.MustPathFromString("resources.pipelines.undefined-pipeline")},
	})
}
