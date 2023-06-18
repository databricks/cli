package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type processEnvironmentMode struct{}

func ProcessEnvironmentMode() bundle.Mutator {
	return &processEnvironmentMode{}
}

func (m *processEnvironmentMode) Name() string {
	return "ProcessEnvironmentMode"
}

// Mark all resources as being for 'debug' purposes, i.e.
// changing their their name, adding tags, and (in the future)
// marking them as 'hidden' in the UI.
func processDebugMode(b *bundle.Bundle) error {
	r := b.Config.Resources

	for i := range r.Jobs {
		r.Jobs[i].Name = "[debug] " + r.Jobs[i].Name
		if r.Jobs[i].Tags == nil {
			r.Jobs[i].Tags = make(map[string]string)
		}
		r.Jobs[i].Tags["debug"] = ""
	}

	for i := range r.Pipelines {
		r.Pipelines[i].Name = "[debug] " + r.Pipelines[i].Name
		r.Pipelines[i].Development = true
		// (pipelines don't have tags)
	}

	for i := range r.Models {
		r.Models[i].Name = "[debug] " + r.Models[i].Name
		r.Models[i].Tags = append(r.Models[i].Tags, ml.ModelTag{Key: "debug", Value: ""})
	}

	for i := range r.Experiments {
		r.Experiments[i].Name = "[debug] " + r.Experiments[i].Name
		r.Experiments[i].Tags = append(r.Experiments[i].Tags, ml.ExperimentTag{Key: "debug", Value: ""})
	}

	return nil
}

func (m *processEnvironmentMode) Apply(ctx context.Context, b *bundle.Bundle) error {
	switch b.Config.Bundle.Mode {
	case config.Debug:
		return processDebugMode(b)
	case config.Default, "":
		// No action
	case config.PullRequest:
		return fmt.Errorf("not implemented")
	default:
		return fmt.Errorf("unsupported value specified for 'mode': %s", b.Config.Bundle.Mode)
	}

	return nil
}
