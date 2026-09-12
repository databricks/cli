package validate

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stateSizeBundle returns a bundle with one job whose description is n bytes,
// which lands in the recorded state and so drives its size.
func stateSizeBundle(t *testing.T, n int) *bundle.Bundle {
	t.Helper()
	b := &bundle.Bundle{
		Config: config.Root{
			Experimental: &config.Experimental{RecordDeploymentHistory: true},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"big": {JobSettings: jobs.JobSettings{Name: "big"}},
				},
			},
		},
	}
	description := strings.Repeat("x", n)
	b.Config.Resources.Jobs["big"].Description = description
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.big.description", dyn.V(description))
	})
	return b
}

func TestValidateStateSizeUnderLimit(t *testing.T) {
	b := stateSizeBundle(t, 1024)

	diags := ValidateStateSize(engine.EngineDirect).Apply(t.Context(), b)
	require.Empty(t, diags)
}

func TestValidateStateSizeOverLimit(t *testing.T) {
	b := stateSizeBundle(t, MaxStateSizeBytes+1)

	diags := ValidateStateSize(engine.EngineDirect).Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "resources.jobs.big")
	assert.Contains(t, diags[0].Summary, "exceeds the 65536 byte limit")
	assert.Equal(t, []dyn.Path{dyn.MustPathFromString("resources.jobs.big")}, diags[0].Paths)
	assert.Contains(t, diags[0].Detail, "split this job into multiple jobs")
}

// The advice names the split that applies to the resource type, since "split it
// up" means a different edit for a job than for an alert.
func TestSizeAdvice(t *testing.T) {
	assert.Contains(t, sizeAdvice("jobs"), "split this job into multiple jobs")
	assert.Contains(t, sizeAdvice("alerts"), "split this alert into multiple alerts")
	assert.Contains(t, sizeAdvice("pipelines"), "split this pipeline into multiple pipelines")
	assert.Contains(t, sizeAdvice("schemas"), "split this resource into multiple smaller resources")
}

// The state is only uploaded by the direct engine, so a terraform deployment of
// the same oversized resource must not be blocked.
func TestValidateStateSizeSkippedForTerraform(t *testing.T) {
	b := stateSizeBundle(t, MaxStateSizeBytes+1)

	diags := ValidateStateSize(engine.EngineTerraform).Apply(t.Context(), b)
	require.Empty(t, diags)
}

func TestValidateStateSizeSkippedWithoutOptIn(t *testing.T) {
	b := stateSizeBundle(t, MaxStateSizeBytes+1)
	b.Config.Experimental.RecordDeploymentHistory = false

	diags := ValidateStateSize(engine.EngineDirect).Apply(t.Context(), b)
	require.Empty(t, diags)
}

func TestValidateStateSizeNoExperimentalBlock(t *testing.T) {
	b := stateSizeBundle(t, MaxStateSizeBytes+1)
	b.Config.Experimental = nil

	diags := ValidateStateSize(engine.EngineDirect).Apply(t.Context(), b)
	require.Empty(t, diags)
}

func TestValidateStateSizeAlertAdvice(t *testing.T) {
	queryText := strings.Repeat("s", MaxStateSizeBytes+1)
	b := &bundle.Bundle{
		Config: config.Root{
			Experimental: &config.Experimental{RecordDeploymentHistory: true},
			Resources: config.Resources{
				Alerts: map[string]*resources.Alert{
					"noisy": {AlertV2: sql.AlertV2{QueryText: queryText}},
				},
			},
		},
	}
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.alerts.noisy", dyn.V(map[string]dyn.Value{
			"query_text": dyn.V(queryText),
		}))
	})

	diags := ValidateStateSize(engine.EngineDirect).Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "resources.alerts.noisy")
	assert.Contains(t, diags[0].Detail, "split this alert into multiple alerts")
}

// Every oversized resource is reported, so one deploy attempt surfaces them all.
func TestValidateStateSizeReportsEveryResource(t *testing.T) {
	b := stateSizeBundle(t, MaxStateSizeBytes+1)
	description := strings.Repeat("y", MaxStateSizeBytes+1)
	b.Config.Resources.Jobs["also_big"] = &resources.Job{
		JobSettings: jobs.JobSettings{Name: "also_big", Description: description},
	}
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.also_big", dyn.V(map[string]dyn.Value{
			"name":        dyn.V("also_big"),
			"description": dyn.V(description),
		}))
	})

	diags := ValidateStateSize(engine.EngineDirect).Apply(t.Context(), b)
	require.Len(t, diags, 2)
	// Diagnostics are ordered by resource key so output does not depend on map
	// iteration order.
	assert.Contains(t, diags[0].Summary, "resources.jobs.also_big")
	assert.Contains(t, diags[1].Summary, "resources.jobs.big")
}
