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
	t := b.Config.Bundle.Transformers
	prefix := t.Prefix.Value
	tags := t.Tags.Tags

	// Jobs transformers: Prefix, Tags, JobsMaxConcurrentRuns, TriggerPauseStatus
	for i := range r.Jobs {
		if t.Prefix.IsEnabled() {
			r.Jobs[i].Name = prefix + r.Jobs[i].Name
		}
		if t.Tags.IsEnabled() {
			for tagKey, tagValue := range tags {
				if r.Jobs[i].Tags == nil {
					r.Jobs[i].Tags = make(map[string]string)
				}
				r.Jobs[i].Tags[tagKey] = tagValue
			}
		}
		if t.JobsMaxConcurrentRuns.IsEnabled() && r.Jobs[i].MaxConcurrentRuns == 0 {
			r.Jobs[i].MaxConcurrentRuns = t.JobsMaxConcurrentRuns.Value
		}
		if t.TriggerPauseStatus.IsEnabled() {
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
		if t.Prefix.IsEnabled() {
			r.Pipelines[i].Name = prefix + r.Pipelines[i].Name
		}
		if t.PipelinesDevelopment.IsEnabled() {
			r.Pipelines[i].Development = true
		}

		// As of 2024-06, pipelines don't yet support tags
	}

	// Models transformers: Prefix, Tags
	for i := range r.Models {
		if t.Prefix.IsEnabled() {
			r.Models[i].Name = prefix + r.Models[i].Name
		}
		if t.Tags.IsEnabled() {
			for tagKey, tagValue := range tags {
				r.Models[i].Tags = append(r.Models[i].Tags, ml.ModelTag{Key: tagKey, Value: tagValue})
			}
		}
	}

	// Experiments transformers: Prefix, Tags
	for i := range r.Experiments {
		if t.Prefix.IsEnabled() {
			filepath := r.Experiments[i].Name
			dir := path.Dir(filepath)
			base := path.Base(filepath)
			if dir == "." {
				r.Experiments[i].Name = prefix + base
			} else {
				r.Experiments[i].Name = dir + "/" + prefix + base
			}
		}
		if t.Tags.IsEnabled() {
			for tagKey, tagValue := range tags {
				r.Experiments[i].Tags = append(r.Experiments[i].Tags, ml.ExperimentTag{Key: tagKey, Value: tagValue})
			}
		}
	}

	// Model serving endpoint transformers: Prefix
	for i := range r.ModelServingEndpoints {
		if t.Prefix.IsEnabled() {
			r.ModelServingEndpoints[i].Name = normalizePrefix(prefix) + r.ModelServingEndpoints[i].Name
		}

		// As of 2024-06, model serving endpoints don't yet support tags
	}

	// Registered models transformers: Prefix
	for i := range r.RegisteredModels {
		if t.Prefix.IsEnabled() {
			r.RegisteredModels[i].Name = normalizePrefix(prefix) + r.RegisteredModels[i].Name
		}

		// As of 2024-06, registered models don't yet support tags
	}

	return nil
}

// Normalize prefix strings like '[dev lennart] ' to 'dev_lennart_'.
// We leave unicode letters and numbers but remove all "special characters."
func normalizePrefix(prefix string) string {
	prefix = strings.ReplaceAll(prefix, "[", "")
	prefix = strings.ReplaceAll(prefix, "] ", "_")
	return textutil.NormalizeString(prefix)
}
