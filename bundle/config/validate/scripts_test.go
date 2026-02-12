package validate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptsWithValidDABInterpolation(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Scripts: map[string]config.Script{
				"valid_var": {
					Content: "echo ${var.my_variable}",
				},
				"valid_bundle": {
					Content: "echo ${bundle.name}",
				},
				"valid_workspace": {
					Content: "echo ${workspace.host}",
				},
				"valid_resources": {
					Content: "echo ${resources.jobs.my_job.id}",
				},
				"valid_multiple": {
					Content: "echo ${var.foo} and ${bundle.name}",
				},
			},
		},
	}

	bundletest.SetLocation(b, "scripts.valid_var.content", []dyn.Location{{File: "databricks.yml", Line: 1, Column: 1}})
	bundletest.SetLocation(b, "scripts.valid_bundle.content", []dyn.Location{{File: "databricks.yml", Line: 2, Column: 1}})
	bundletest.SetLocation(b, "scripts.valid_workspace.content", []dyn.Location{{File: "databricks.yml", Line: 3, Column: 1}})
	bundletest.SetLocation(b, "scripts.valid_resources.content", []dyn.Location{{File: "databricks.yml", Line: 4, Column: 1}})
	bundletest.SetLocation(b, "scripts.valid_multiple.content", []dyn.Location{{File: "databricks.yml", Line: 5, Column: 1}})

	ctx := context.Background()
	diags := Scripts().Apply(ctx, b)
	assert.Empty(t, diags, "valid DAB interpolation should not produce errors")
}

func TestScriptsWithInvalidInterpolation(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Scripts: map[string]config.Script{
				"invalid_single": {
					Content: "echo ${FOO}",
				},
			},
		},
	}

	bundletest.SetLocation(b, "scripts.invalid_single.content", []dyn.Location{{File: "databricks.yml", Line: 1, Column: 1}})

	ctx := context.Background()
	diags := Scripts().Apply(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "${FOO}")
	assert.Contains(t, diags[0].Summary, "Invalid interpolation reference")
	assert.Contains(t, diags[0].Detail, "$FOO")
	assert.Contains(t, diags[0].Detail, "${var.FOO}")
}

func TestScriptsWithBashEnvVars(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Scripts: map[string]config.Script{
				"bash_simple": {
					Content: "echo $FOO",
				},
				"bash_param_expansion": {
					Content: "echo ${VAR:-default}",
				},
			},
		},
	}

	bundletest.SetLocation(b, "scripts.bash_simple.content", []dyn.Location{{File: "databricks.yml", Line: 1, Column: 1}})
	bundletest.SetLocation(b, "scripts.bash_param_expansion.content", []dyn.Location{{File: "databricks.yml", Line: 2, Column: 1}})

	ctx := context.Background()
	diags := Scripts().Apply(ctx, b)
	assert.Empty(t, diags, "bash env vars should not produce errors")
}

func TestScriptsWithMixedContent(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Scripts: map[string]config.Script{
				"mixed": {
					Content: "databricks psql ${var.instance} -- -d $LAKEBASE_DATABASE -c 'CREATE SCHEMA my_schema;'",
				},
			},
		},
	}

	bundletest.SetLocation(b, "scripts.mixed.content", []dyn.Location{{File: "databricks.yml", Line: 1, Column: 1}})

	ctx := context.Background()
	diags := Scripts().Apply(ctx, b)
	assert.Empty(t, diags, "valid DAB interpolation with bash env vars should not produce errors")
}

func TestScriptsWithEmptyContent(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Scripts: map[string]config.Script{
				"empty": {
					Content: "",
				},
			},
		},
	}

	bundletest.SetLocation(b, "scripts.empty.content", []dyn.Location{{File: "databricks.yml", Line: 1, Column: 1}})

	ctx := context.Background()
	diags := Scripts().Apply(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "has no content")
}

func TestScriptsMultipleInvalidReferences(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Scripts: map[string]config.Script{
				"multiple_invalid": {
					Content: "echo ${FOO} ${BAR}",
				},
			},
		},
	}

	bundletest.SetLocation(b, "scripts.multiple_invalid.content", []dyn.Location{{File: "databricks.yml", Line: 1, Column: 1}})

	ctx := context.Background()
	diags := Scripts().Apply(ctx, b)
	require.Len(t, diags, 2)
	// Order matches order of appearance in the string
	assert.Contains(t, diags[0].Summary, "${FOO}")
	assert.Contains(t, diags[1].Summary, "${BAR}")
}
