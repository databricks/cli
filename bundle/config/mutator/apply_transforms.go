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
	"github.com/databricks/databricks-sdk-go/service/catalog"
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
	for _, j := range r.Jobs {
		j.Name = prefix + j.Name
		if j.Tags == nil {
			j.Tags = make(map[string]string)
		}
		for _, tag := range tags {
			if j.Tags[tag.Key] == "" {
				j.Tags[tag.Key] = tag.Value
			}
		}
		if j.MaxConcurrentRuns == 0 {
			j.MaxConcurrentRuns = t.DefaultJobsMaxConcurrentRuns
		}
		if t.DefaultTriggerPauseStatus != "" {
			paused := jobs.PauseStatusPaused
			if t.DefaultTriggerPauseStatus == config.Unpaused {
				paused = jobs.PauseStatusUnpaused
			}

			if j.Schedule != nil && j.Schedule.PauseStatus == "" {
				j.Schedule.PauseStatus = paused
			}
			if j.Continuous != nil && j.Continuous.PauseStatus == "" {
				j.Continuous.PauseStatus = paused
			}
			if j.Trigger != nil && j.Trigger.PauseStatus == "" {
				j.Trigger.PauseStatus = paused
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
	for _, m := range r.Models {
		m.Name = prefix + m.Name
		for _, t := range tags {
			exists := slices.ContainsFunc(m.Tags, func(modelTag ml.ModelTag) bool {
				return modelTag.Key == t.Key
			})
			if !exists {
				// Only add this tag if the resource didn't include any tag that overrides its value.
				m.Tags = append(m.Tags, ml.ModelTag{Key: t.Key, Value: t.Value})
			}
		}
	}

	// Experiments transforms: Prefix, Tags
	for _, e := range r.Experiments {
		filepath := e.Name
		dir := path.Dir(filepath)
		base := path.Base(filepath)
		if dir == "." {
			e.Name = prefix + base
		} else {
			e.Name = dir + "/" + prefix + base
		}
		for _, t := range tags {
			exists := false
			for _, experimentTag := range e.Tags {
				if experimentTag.Key == t.Key {
					exists = true
					break
				}
			}
			if !exists {
				e.Tags = append(e.Tags, ml.ExperimentTag{Key: t.Key, Value: t.Value})
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

	// Quality monitors transforms: Prefix
	if t.DefaultTriggerPauseStatus == config.Paused {
		for i := range r.QualityMonitors {
			// Remove all schedules from monitors, since they don't support pausing/unpausing.
			// Quality monitors might support the "pause" property in the future, so at the
			// CLI level we do respect that property if it is set to "unpaused."
			if r.QualityMonitors[i].Schedule != nil && r.QualityMonitors[i].Schedule.PauseStatus != catalog.MonitorCronSchedulePauseStatusUnpaused {
				r.QualityMonitors[i].Schedule = nil
			}
		}
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

// toTagArray converts a map of tags to an array of tags.
// We sort tags so ensure stable ordering.
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

// normalizePrefix prefixes strings like '[dev lennart] ' to 'dev_lennart_'.
// We leave unicode letters and numbers but remove all "special characters."
func normalizePrefix(prefix string) string {
	prefix = strings.ReplaceAll(prefix, "[", "")
	prefix = strings.ReplaceAll(prefix, "] ", "_")
	return textutil.NormalizeString(prefix)
}
