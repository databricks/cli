package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type applyTargetMode struct{}

const developmentConcurrentRuns = 4

// ApplyTargetMode configures the default values for presets in development mode
func ApplyTargetMode() bundle.Mutator {
	return &applyTargetMode{}
}

func (m *applyTargetMode) Name() string {
	return "ApplyTargetMode"
}

// Mark all resources as being for 'development' purposes, i.e.
// changing their their name, adding tags, and (in the future)
// marking them as 'hidden' in the UI.
func transformDevelopmentMode(ctx context.Context, b *bundle.Bundle) {
	if !b.Config.Bundle.Deployment.Lock.IsExplicitlyEnabled() {
		log.Infof(ctx, "Development mode: disabling deployment lock since bundle.deployment.lock.enabled is not set to true")
		disabled := false
		b.Config.Bundle.Deployment.Lock.Enabled = &disabled
	}

	t := &b.Config.Presets
	shortName := b.Config.Workspace.CurrentUser.ShortName

	if t.NamePrefix == "" {
		t.NamePrefix = "[dev " + shortName + "] "
	}

	if t.Tags == nil {
		t.Tags = map[string]string{}
	}
	_, exists := t.Tags["dev"]
	if !exists {
		t.Tags["dev"] = b.Tagging.NormalizeValue(shortName)
	}

	if t.JobsMaxConcurrentRuns == 0 {
		t.JobsMaxConcurrentRuns = developmentConcurrentRuns
	}

	if t.TriggerPauseStatus == "" {
		t.TriggerPauseStatus = config.Paused
	}

	if !config.IsExplicitlyDisabled(t.PipelinesDevelopment) {
		enabled := true
		t.PipelinesDevelopment = &enabled
	}
}

func (m *applyTargetMode) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Bundle.Mode == config.Development {
		transformDevelopmentMode(ctx, b)
	}

	return nil
}
