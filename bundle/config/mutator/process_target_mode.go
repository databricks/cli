package mutator

import (
	"context"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
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
func transformDevelopmentMode(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !b.Config.Bundle.Deployment.Lock.IsExplicitlyEnabled() {
		log.Infof(ctx, "Development mode: disabling deployment lock since bundle.deployment.lock.enabled is not set to true")
		disabled := false
		b.Config.Bundle.Deployment.Lock.Enabled = &disabled
	}

	r := b.Config.Resources
	shortName := b.Config.Workspace.CurrentUser.ShortName
	prefix := "[dev " + shortName + "] "

	// Generate a normalized version of the short name that can be used as a tag value.
	tagValue := b.Tagging.NormalizeValue(shortName)

	for i := range r.Jobs {
		r.Jobs[i].Name = prefix + r.Jobs[i].Name
		if r.Jobs[i].Tags == nil {
			r.Jobs[i].Tags = make(map[string]string)
		}
		r.Jobs[i].Tags["dev"] = tagValue
		if r.Jobs[i].MaxConcurrentRuns == 0 {
			r.Jobs[i].MaxConcurrentRuns = developmentConcurrentRuns
		}

		// Pause each job. As an exception, we don't pause jobs that are explicitly
		// marked as "unpaused". This allows users to override the default behavior
		// of the development mode.
		if r.Jobs[i].Schedule != nil && r.Jobs[i].Schedule.PauseStatus != jobs.PauseStatusUnpaused {
			r.Jobs[i].Schedule.PauseStatus = jobs.PauseStatusPaused
		}
		if r.Jobs[i].Continuous != nil && r.Jobs[i].Continuous.PauseStatus != jobs.PauseStatusUnpaused {
			r.Jobs[i].Continuous.PauseStatus = jobs.PauseStatusPaused
		}
		if r.Jobs[i].Trigger != nil && r.Jobs[i].Trigger.PauseStatus != jobs.PauseStatusUnpaused {
			r.Jobs[i].Trigger.PauseStatus = jobs.PauseStatusPaused
		}
	}

	for i := range r.Pipelines {
		r.Pipelines[i].Name = prefix + r.Pipelines[i].Name
		r.Pipelines[i].Development = true
		// (pipelines don't yet support tags)
	}

	for i := range r.Models {
		r.Models[i].Name = prefix + r.Models[i].Name
		r.Models[i].Tags = append(r.Models[i].Tags, ml.ModelTag{Key: "dev", Value: tagValue})
	}

	for i := range r.Experiments {
		filepath := r.Experiments[i].Name
		dir := path.Dir(filepath)
		base := path.Base(filepath)
		if dir == "." {
			r.Experiments[i].Name = prefix + base
		} else {
			r.Experiments[i].Name = dir + "/" + prefix + base
		}
		r.Experiments[i].Tags = append(r.Experiments[i].Tags, ml.ExperimentTag{Key: "dev", Value: tagValue})
	}

	for i := range r.ModelServingEndpoints {
		prefix = "dev_" + b.Config.Workspace.CurrentUser.ShortName + "_"
		r.ModelServingEndpoints[i].Name = prefix + r.ModelServingEndpoints[i].Name
		// (model serving doesn't yet support tags)
	}

	for i := range r.RegisteredModels {
		prefix = "dev_" + b.Config.Workspace.CurrentUser.ShortName + "_"
		r.RegisteredModels[i].Name = prefix + r.RegisteredModels[i].Name
		// (registered models in Unity Catalog don't yet support tags)
	}

	for i := range r.QualityMonitors {
		// Remove all schedules from monitors, since they don't support pausing/unpausing.
		// Quality monitors might support the "pause" property in the future, so at the
		// CLI level we do respect that property if it is set to "unpaused".
		if r.QualityMonitors[i].Schedule != nil && r.QualityMonitors[i].Schedule.PauseStatus != catalog.MonitorCronSchedulePauseStatusUnpaused {
			r.QualityMonitors[i].Schedule = nil
		}
	}

	for i := range r.Schemas {
		prefix = "dev_" + b.Config.Workspace.CurrentUser.ShortName + "_"
		r.Schemas[i].Name = prefix + r.Schemas[i].Name
		// HTTP API for schemas doesn't yet support tags. It's only supported in
		// the Databricks UI and via the SQL API.
	}

	return nil
}

func validateDevelopmentMode(b *bundle.Bundle) diag.Diagnostics {
	if path := findNonUserPath(b); path != "" {
		return diag.Errorf("%s must start with '~/' or contain the current username when using 'mode: development'", path)
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
		if diags != nil {
			return diags
		}
		return transformDevelopmentMode(ctx, b)
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
