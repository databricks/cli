package mutator

import (
	"context"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type transformers struct{}

func TransformersMutator() *transformers {
	return &transformers{}
}

func (m *transformers) Name() string {
	return "TransformersMutator"
}

func (m *transformers) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	r := b.Config.Resources
	mutators := b.Config.Bundle.Transformers

	prefix := mutators.Prefix.Value
	if !isEnabled(mutators.Prefix.Enabled) {
		prefix = ""
	}

	for i := range r.Jobs {
		r.Jobs[i].Name = prefix + r.Jobs[i].Name
		if isEnabled(mutators.JobsMaxConcurrentRuns.Enabled) && r.Jobs[i].MaxConcurrentRuns == 0 {
			r.Jobs[i].MaxConcurrentRuns = mutators.JobsMaxConcurrentRuns.Value
		}

		if isEnabled(mutators.JobsSchedulePauseStatus.Enabled) {
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
	}

	for i := range r.Pipelines {
		r.Pipelines[i].Name = prefix + r.Pipelines[i].Name
		if isEnabled(mutators.PipelinesDevelopment.Enabled) {
			r.Pipelines[i].Development = true
		}
	}

	for i := range r.Models {
		r.Models[i].Name = prefix + r.Models[i].Name
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
	}

	for i := range r.ModelServingEndpoints {
		r.ModelServingEndpoints[i].Name = normalizePrefix(prefix) + r.ModelServingEndpoints[i].Name
	}

	for i := range r.RegisteredModels {
		r.RegisteredModels[i].Name = normalizePrefix(prefix) + r.RegisteredModels[i].Name
	}

	addTags(b)

	return nil
}

func isEnabled(enabled *bool) bool {
	return enabled != nil && *enabled
}

// Normalize prefix strings like '[dev lennart] ' to 'dev_lennart_'.
// We leave unicode letters and numbers but remove all "special characters."
func normalizePrefix(prefix string) string {
	prefix = strings.ReplaceAll(prefix, "[", "")
	prefix = strings.ReplaceAll(prefix, "] ", "_")
	return textutil.NormalizeString(prefix)
}

// Add tags for supported resources.
// As of 2024-04, pipelines, model serving, and registered models in Unity Catalog
// don't yet support tags.
func addTags(b *bundle.Bundle) {
	mutators := b.Config.Bundle.Transformers
	if !isEnabled(mutators.Tags.Enabled) {
		return
	}

	r := b.Config.Resources
	tags := mutators.Tags.Tags
	for tagKey, tagValue := range tags {

		for i := range r.Jobs {
			if r.Jobs[i].Tags == nil {
				r.Jobs[i].Tags = make(map[string]string)
			}
			r.Jobs[i].Tags[tagKey] = tagValue
			if r.Jobs[i].MaxConcurrentRuns == 0 {
				r.Jobs[i].MaxConcurrentRuns = developmentConcurrentRuns
			}
		}

		for i := range r.Models {
			r.Models[i].Tags = append(r.Models[i].Tags, ml.ModelTag{Key: tagKey, Value: tagValue})
		}

		for i := range r.Experiments {
			r.Experiments[i].Tags = append(r.Experiments[i].Tags, ml.ExperimentTag{Key: tagKey, Value: tagValue})
		}
	}
}
