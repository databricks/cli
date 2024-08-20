package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
)

type processTargetMode struct{}

const developmentConcurrentRuns = 4

func ProcessTargetMode() bundle.Mutator {
	return &processTargetMode{}
}

func (m *processTargetMode) Name() string {
	return "ProcessTargetMode"
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

func validateDevelopmentMode(b *bundle.Bundle) diag.Diagnostics {
	p := b.Config.Presets
	u := b.Config.Workspace.CurrentUser

	// Make sure presets don't set the trigger status to UNPAUSED;
	// this could be surprising since most users (and tools) expect triggers
	// to be paused in development.
	// (Note that there still is an exceptional case where users set the trigger
	// status to UNPAUSED at the level of an individual object, whic hwas
	// historically allowed.)
	if p.TriggerPauseStatus == config.Unpaused {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "target with 'mode: development' cannot set trigger pause status to UNPAUSED by default",
			Locations: []dyn.Location{b.Config.GetLocation("presets.trigger_pause_status")},
		}}
	}

	// Make sure this development copy has unique names and paths to avoid conflicts
	if path := findNonUserPath(b); path != "" {
		return diag.Errorf("%s must start with '~/' or contain the current username when using 'mode: development'", path)
	}
	if p.NamePrefix != "" && !strings.Contains(p.NamePrefix, u.ShortName) && !strings.Contains(p.NamePrefix, u.UserName) {
		// Resources such as pipelines require a unique name, e.g. '[dev steve] my_pipeline'.
		// For this reason we require the name prefix to contain the current username;
		// it's a pitfall for users if they don't include it and later find out that
		// only a single user can do development deployments.
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "prefix should contain the current username or ${workspace.current_user.short_name} to ensure uniqueness when using 'mode: development'",
			Locations: []dyn.Location{b.Config.GetLocation("presets.name_prefix")},
		}}
	}
	return nil
}

func findNonUserPath(b *bundle.Bundle) string {
	username := b.Config.Workspace.CurrentUser.UserName

	if b.Config.Workspace.RootPath != "" && !strings.Contains(b.Config.Workspace.RootPath, username) {
		return "root_path"
	}
	if b.Config.Workspace.StatePath != "" && !strings.Contains(b.Config.Workspace.StatePath, username) {
		return "state_path"
	}
	if b.Config.Workspace.FilePath != "" && !strings.Contains(b.Config.Workspace.FilePath, username) {
		return "file_path"
	}
	if b.Config.Workspace.ArtifactPath != "" && !strings.Contains(b.Config.Workspace.ArtifactPath, username) {
		return "artifact_path"
	}
	return ""
}

func validateProductionMode(ctx context.Context, b *bundle.Bundle, isPrincipalUsed bool) diag.Diagnostics {
	if b.Config.Bundle.Git.Inferred {
		env := b.Config.Bundle.Target
		log.Warnf(ctx, "target with 'mode: production' should specify an explicit 'targets.%s.git' configuration", env)
	}

	r := b.Config.Resources
	for i := range r.Pipelines {
		if r.Pipelines[i].Development {
			return diag.Errorf("target with 'mode: production' cannot include a pipeline with 'development: true'")
		}
	}

	if !isPrincipalUsed && !isRunAsSet(r) {
		return diag.Errorf("'run_as' must be set for all jobs when using 'mode: production'")
	}
	return nil
}

// Determines whether run_as is explicitly set for all resources.
// We do this in a best-effort fashion rather than check the top-level
// 'run_as' field because the latter is not required to be set.
func isRunAsSet(r config.Resources) bool {
	for i := range r.Jobs {
		if r.Jobs[i].RunAs == nil {
			return false
		}
	}
	return true
}

func (m *processTargetMode) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	switch b.Config.Bundle.Mode {
	case config.Development:
		diags := validateDevelopmentMode(b)
		if diags.HasError() {
			return diags
		}
		transformDevelopmentMode(ctx, b)
		return diags
	case config.Production:
		isPrincipal := auth.IsServicePrincipal(b.Config.Workspace.CurrentUser.UserName)
		return validateProductionMode(ctx, b, isPrincipal)
	case "":
		// No action
	default:
		return diag.Errorf("unsupported value '%s' specified for 'mode': must be either 'development' or 'production'", b.Config.Bundle.Mode)
	}

	return nil
}
