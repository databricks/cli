package mutator

import (
	"context"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
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
	mutators := b.Config.Bundle.Transformers
	r := b.Config.Resources

	if isEnabled(mutators.Prefix.Enabled) {
		transformPrefix(r, mutators.Prefix.Value)
	}
	if isEnabled(mutators.Tags.Enabled) {
		transformTags(r, mutators.Tags.Tags)
	}
	if isEnabled(mutators.JobsMaxConcurrentRuns.Enabled) {
		transformJobsMaxConcurrentRuns(r, mutators.JobsMaxConcurrentRuns.Value)
	}
	if isEnabled(mutators.TriggerPauseStatus.Enabled) {
		transformTriggerPauseStatus(r, mutators.TriggerPauseStatus.Enabled)
	}
	if isEnabled(mutators.PipelinesDevelopment.Enabled) {
		transformPipelinesDevelopment(r)
	}

	return nil
}

func transformPrefix(r config.Resources, prefix string) {
	for i := range r.Jobs {
		r.Jobs[i].Name = prefix + r.Jobs[i].Name
		if isEnabled(mutators.JobsMaxConcurrentRuns.Enabled) && r.Jobs[i].MaxConcurrentRuns == 0 {
			r.Jobs[i].MaxConcurrentRuns = mutators.JobsMaxConcurrentRuns.Value
		}

		if isEnabled(mutators.TriggerPauseStatus.Enabled) {
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

	// Pipelines transformers: Prefix, PipelinesDevelopment
	for i := range r.Pipelines {
		r.Pipelines[i].Name = prefix + r.Pipelines[i].Name
		if isEnabled(mutators.PipelinesDevelopment.Enabled) {
			r.Pipelines[i].Development = true
		}
	}

	// Models transformers: Prefix
	for i := range r.Models {
		r.Models[i].Name = prefix + r.Models[i].Name
	}

	// Experiments transformers: Prefix
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

	// Model serving endpoint transformers: Prefix
	for i := range r.ModelServingEndpoints {
		r.ModelServingEndpoints[i].Name = normalizePrefix(prefix) + r.ModelServingEndpoints[i].Name
	}

	// Registered models transformers: Prefix
	for i := range r.RegisteredModels {
		r.RegisteredModels[i].Name = normalizePrefix(prefix) + r.RegisteredModels[i].Name
	}
}

// Test whether a transformer is enabled.
// Enablement has three  states: explicitly enabled, explicitly disabled, and not set.
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
func transformTags(r config.Resources, tags map[string]string) {
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

		// As of 2024-04, pipelines, model serving, and registered models in Unity Catalog
		// don't yet support tags.
	}
}
