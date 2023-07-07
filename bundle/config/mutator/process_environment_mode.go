package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type processEnvironmentMode struct{}

const developmentConcurrentRuns = 4

func ProcessEnvironmentMode() bundle.Mutator {
	return &processEnvironmentMode{}
}

func (m *processEnvironmentMode) Name() string {
	return "ProcessEnvironmentMode"
}

// Mark all resources as being for 'development' purposes, i.e.
// changing their their name, adding tags, and (in the future)
// marking them as 'hidden' in the UI.
func processDevelopmentMode(b *bundle.Bundle) error {
	r := b.Config.Resources

	for i := range r.Jobs {
		r.Jobs[i].Name = "[dev] " + r.Jobs[i].Name
		if r.Jobs[i].Tags == nil {
			r.Jobs[i].Tags = make(map[string]string)
		}
		r.Jobs[i].Tags["dev"] = ""
		if r.Jobs[i].MaxConcurrentRuns == 0 {
			r.Jobs[i].MaxConcurrentRuns = developmentConcurrentRuns
		}
		if r.Jobs[i].Schedule != nil {
			r.Jobs[i].Schedule.PauseStatus = jobs.PauseStatusPaused
		}
		if r.Jobs[i].Continuous != nil {
			r.Jobs[i].Continuous.PauseStatus = jobs.PauseStatusPaused
		}
		if r.Jobs[i].Trigger != nil {
			r.Jobs[i].Trigger.PauseStatus = jobs.PauseStatusPaused
		}
	}

	for i := range r.Pipelines {
		r.Pipelines[i].Name = "[dev] " + r.Pipelines[i].Name
		r.Pipelines[i].Development = true
		// (pipelines don't yet support tags)
	}

	for i := range r.Models {
		r.Models[i].Name = "[dev] " + r.Models[i].Name
		r.Models[i].Tags = append(r.Models[i].Tags, ml.ModelTag{Key: "dev", Value: ""})
	}

	for i := range r.Experiments {
		path := r.Experiments[i].Name
		dir := filepath.Dir(path)
		base := filepath.Base(path)
		if dir == "." {
			r.Experiments[i].Name = "[dev] " + base
		} else {
			r.Experiments[i].Name = dir + "/[dev] " + base
		}
		r.Experiments[i].Tags = append(r.Experiments[i].Tags, ml.ExperimentTag{Key: "dev", Value: ""})
	}

	return nil
}

func (m *processEnvironmentMode) Apply(ctx context.Context, b *bundle.Bundle) error {
	switch b.Config.Bundle.Mode {
	case config.Development:
		return processDevelopmentMode(b)
	case "":
		// No action
	default:
		return fmt.Errorf("unsupported value specified for 'mode': %s", b.Config.Bundle.Mode)
	}

	return nil
}
