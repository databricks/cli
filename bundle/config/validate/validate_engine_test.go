package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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
	bundletest.SetLocation(b, "bundle.engine", []dyn.Location{{File: "databricks.yml", Line: 5, Column: 3}})
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestValidateEngineTerraformDeprecated(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Engine: engine.EngineTerraform,
			},
		},
	}
	loc := dyn.Location{File: "databricks.yml", Line: 5, Column: 3}
	bundletest.SetLocation(b, "bundle.engine", []dyn.Location{loc})
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "deprecated")
	assert.Equal(t, []dyn.Location{loc}, diags[0].Locations)
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
	bundletest.SetLocation(b, "bundle.engine", []dyn.Location{{File: "databricks.yml", Line: 5, Column: 3}})
	diags := ValidateEngine().Apply(t.Context(), b)
	assert.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "invalid")
}
