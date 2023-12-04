package mutator

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/log"
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
func transformDevelopmentMode(b *bundle.Bundle) error {
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
		r.Models[i].Tags = append(r.Models[i].Tags, ml.ModelTag{Key: "dev", Value: ""})
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

	return nil
}

func validateDevelopmentMode(b *bundle.Bundle) error {
	if path := findIncorrectPath(b, config.Development); path != "" {
		return fmt.Errorf("%s must start with '~/' or contain the current username when using 'mode: development'", path)
	}
	return nil
}

func findIncorrectPath(b *bundle.Bundle, mode config.Mode) string {
	username := b.Config.Workspace.CurrentUser.UserName
	containsExpected := true
	if mode == config.Production {
		containsExpected = false
	}

	if strings.Contains(b.Config.Workspace.RootPath, username) != containsExpected && b.Config.Workspace.RootPath != "" {
		return "root_path"
	}
	if strings.Contains(b.Config.Workspace.StatePath, username) != containsExpected {
		return "state_path"
	}
	if strings.Contains(b.Config.Workspace.FilePath, username) != containsExpected {
		return "file_path"
	}
	if strings.Contains(b.Config.Workspace.ArtifactPath, username) != containsExpected {
		return "artifact_path"
	}
	return ""
}

func validateProductionMode(ctx context.Context, b *bundle.Bundle, isPrincipalUsed bool) error {
	if b.Config.Bundle.Git.Inferred {
		env := b.Config.Bundle.Target
		log.Warnf(ctx, "target with 'mode: production' should specify an explicit 'targets.%s.git' configuration", env)
	}

	r := b.Config.Resources
	for i := range r.Pipelines {
		if r.Pipelines[i].Development {
			return fmt.Errorf("target with 'mode: production' cannot specify a pipeline with 'development: true'")
		}
	}

	if !isPrincipalUsed {
		if path := findIncorrectPath(b, config.Production); path != "" {
			message := "%s must not contain the current username when using 'mode: production'"
			if path == "root_path" {
				return fmt.Errorf(message+"\n  tip: set workspace.root_path to a shared path such as /Shared/.bundle/${bundle.name}/${bundle.target}", path)
			} else {
				return fmt.Errorf(message, path)
			}
		}

		if !isRunAsSet(r) {
			return fmt.Errorf("'run_as' must be set for all jobs when using 'mode: production'")
		}
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

func (m *processTargetMode) Apply(ctx context.Context, b *bundle.Bundle) error {
	switch b.Config.Bundle.Mode {
	case config.Development:
		err := validateDevelopmentMode(b)
		if err != nil {
			return err
		}
		return transformDevelopmentMode(b)
	case config.Production:
		isPrincipal := auth.IsServicePrincipal(b.Config.Workspace.CurrentUser.UserName)
		return validateProductionMode(ctx, b, isPrincipal)
	case "":
		// No action
	default:
		return fmt.Errorf("unsupported value '%s' specified for 'mode': must be either 'development' or 'production'", b.Config.Bundle.Mode)
	}

	return nil
}
