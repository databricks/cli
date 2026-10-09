package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
)

func TestValidateEngineDirect(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineDirect,
			},
		},
	}
	bundletest.SetLocation(b, "bundle.engine", []diag.Location{{File: "databricks.yml", Line: 5, Column: 3}})
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestValidateEngineTerraformRemoved(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineTerraform,
			},
		},
	}
	loc := diag.Location{File: "databricks.yml", Line: 5, Column: 3}
	bundletest.SetLocation(b, "bundle.engine", []diag.Location{loc})
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "has been removed")
	assert.Equal(t, "bundle.engine", diags[0].Paths[0].String())
}

func TestValidateEngineNotSet(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{},
	}
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestValidateEngineInvalid(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineType("invalid"),
			},
		},
	}
	bundletest.SetLocation(b, "bundle.engine", []diag.Location{{File: "databricks.yml", Line: 5, Column: 3}})
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "invalid")
}

func TestValidateEngineTerraformConfigDeprecated(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Terraform: &config.Terraform{ExecPath: "/usr/bin/terraform"},
			},
		},
	}
	loc := diag.Location{File: "databricks.yml", Line: 3, Column: 5}
	bundletest.SetLocation(b, "bundle.terraform", []diag.Location{loc})
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "bundle.terraform is deprecated")
	assert.Equal(t, []diag.Location{loc}, diags[0].Locations)
}

func TestValidateEngineTerraformAllowed(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineTerraform,
			},
		},
		AllowTerraformEngineConfig: true,
	}
	bundletest.SetLocation(b, "bundle.engine", []diag.Location{{File: "databricks.yml", Line: 5, Column: 3}})
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "has been removed")
}
