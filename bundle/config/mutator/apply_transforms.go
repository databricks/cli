package mutator

import (
	"context"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type applyTransforms struct{}

// Apply all transforms, e.g. the prefix transform that
// adds a prefix to all names of all resources.
func ApplyTransforms() *applyTransforms {
	return &applyTransforms{}
}

type Tag struct {
	Key   string
	Value string
}

func (m *applyTransforms) Name() string {
	return "ApplyTransforms"
}

func (m *applyTransforms) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diag := validatePauseStatus(b)
	if diag != nil {
		return diag
	}

	r := b.Config.Resources
	t := b.Config.Transform
	prefix := t.Prefix
	tags := toTagArray(t.Tags)

	// Jobs transforms: Prefix, Tags, JobsMaxConcurrentRuns, TriggerPauseStatus
	for i := range r.Jobs {
		r.Jobs[i].Name = prefix + r.Jobs[i].Name
		if r.Jobs[i].Tags == nil {
			r.Jobs[i].Tags = make(map[string]string)
		}
		for _, tag := range tags {
			if r.Jobs[i].Tags[tag.Key] == "" {
				r.Jobs[i].Tags[tag.Key] = tag.Value
			}
		}
		if r.Jobs[i].MaxConcurrentRuns == 0 {
			r.Jobs[i].MaxConcurrentRuns = t.DefaultJobsMaxConcurrentRuns
		}
		if t.DefaultTriggerPauseStatus != "" {
			paused := jobs.PauseStatusPaused
			if t.DefaultTriggerPauseStatus == config.Unpaused {
				paused = jobs.PauseStatusUnpaused
			}

			if r.Jobs[i].Schedule != nil && r.Jobs[i].Schedule.PauseStatus == "" {
				r.Jobs[i].Schedule.PauseStatus = paused
			}
			if r.Jobs[i].Continuous != nil && r.Jobs[i].Continuous.PauseStatus == "" {
				r.Jobs[i].Continuous.PauseStatus = paused
			}
			if r.Jobs[i].Trigger != nil && r.Jobs[i].Trigger.PauseStatus == "" {
				r.Jobs[i].Trigger.PauseStatus = paused
			}
		}
	}

	// Pipelines transforms: Prefix, PipelinesDevelopment
	for i := range r.Pipelines {
		r.Pipelines[i].Name = prefix + r.Pipelines[i].Name
		if config.IsExplicitlyEnabled(t.DefaultPipelinesDevelopment) {
			r.Pipelines[i].Development = true
		}

		// As of 2024-06, pipelines don't yet support tags
	}

	// Models transforms: Prefix, Tags
	for i := range r.Models {
		r.Models[i].Name = prefix + r.Models[i].Name
		for _, t := range tags {
			exists := slices.ContainsFunc(r.Models[i].Tags, func(modelTag ml.ModelTag) bool {
				return modelTag.Key == t.Key
			})
			if !exists {
				// Only add this tag if the resource didn't include any tag that overrides its value.
				r.Models[i].Tags = append(r.Models[i].Tags, ml.ModelTag{Key: t.Key, Value: t.Value})
			}
		}
	}

	// Experiments transforms: Prefix, Tags
	for i := range r.Experiments {
		filepath := r.Experiments[i].Name
		dir := path.Dir(filepath)
		base := path.Base(filepath)
		if dir == "." {
			r.Experiments[i].Name = prefix + base
		} else {
			r.Experiments[i].Name = dir + "/" + prefix + base
		}
		for _, t := range tags {
			exists := false
			for _, experimentTag := range r.Experiments[i].Tags {
				if experimentTag.Key == t.Key {
					exists = true
					break
				}
			}
			if !exists {
				r.Experiments[i].Tags = append(r.Experiments[i].Tags, ml.ExperimentTag{Key: t.Key, Value: t.Value})
			}
		}
	}

	// Model serving endpoint transforms: Prefix
	for i := range r.ModelServingEndpoints {
		r.ModelServingEndpoints[i].Name = normalizePrefix(prefix) + r.ModelServingEndpoints[i].Name

		// As of 2024-06, model serving endpoints don't yet support tags
	}

	// Registered models transforms: Prefix
	for i := range r.RegisteredModels {
		r.RegisteredModels[i].Name = normalizePrefix(prefix) + r.RegisteredModels[i].Name

		// As of 2024-06, registered models don't yet support tags
	}

	return nil
}

func validatePauseStatus(b *bundle.Bundle) diag.Diagnostics {
	p := b.Config.Transform.DefaultTriggerPauseStatus
	if p == "" || p == config.Paused || p == config.Unpaused {
		return nil
	}
	return diag.Diagnostics{{
		Summary:  "Invalid value for default_trigger_pause_status, should be PAUSED or UNPAUSED",
		Severity: diag.Error,
		Location: b.Config.GetLocation("transform.default_trigger_pause_status"),
	}}
}

// Convert a map of tags to an array of tags.
// We sort tags so we always produce a consistent list of tags.
func toTagArray(tags *map[string]string) []Tag {
	var tagArray []Tag
	if tags == nil {
		return tagArray
	}
	for key, value := range *tags {
		tagArray = append(tagArray, Tag{Key: key, Value: value})
	}
	sort.Slice(tagArray, func(i, j int) bool {
		return tagArray[i].Key < tagArray[j].Key
	})
	return tagArray
}

// Normalize prefix strings like '[dev lennart] ' to 'dev_lennart_'.
// We leave unicode letters and numbers but remove all "special characters."
func normalizePrefix(prefix string) string {
	prefix = strings.ReplaceAll(prefix, "[", "")
	prefix = strings.ReplaceAll(prefix, "] ", "_")
	return textutil.NormalizeString(prefix)
}
