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
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type applyPresets struct{}

// Apply all presets, e.g. the prefix presets that
// adds a prefix to all names of all resources.
func ApplyPresets() *applyPresets {
	return &applyPresets{}
}

type Tag struct {
	Key   string
	Value string
}

func (m *applyPresets) Name() string {
	return "ApplyPresets"
}

func (m *applyPresets) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if d := validatePauseStatus(b); d != nil {
		return d
	}

	r := b.Config.Resources
	t := b.Config.Presets
	prefix := t.NamePrefix
	tags := toTagArray(t.Tags)

	// Jobs presets: Prefix, Tags, JobsMaxConcurrentRuns, TriggerPauseStatus
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
			j.MaxConcurrentRuns = t.JobsMaxConcurrentRuns
		}
		if t.TriggerPauseStatus != "" {
			paused := jobs.PauseStatusPaused
			if t.TriggerPauseStatus == config.Unpaused {
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

	// Pipelines presets: Prefix, PipelinesDevelopment
	for i := range r.Pipelines {
		r.Pipelines[i].Name = prefix + r.Pipelines[i].Name
		if config.IsExplicitlyEnabled(t.PipelinesDevelopment) {
			r.Pipelines[i].Development = true
		}
		if t.TriggerPauseStatus == config.Paused {
			r.Pipelines[i].Continuous = false
		}

		// As of 2024-06, pipelines don't yet support tags
	}

	// Models presets: Prefix, Tags
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

	// Experiments presets: Prefix, Tags
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

	// Model serving endpoint presets: Prefix
	for i := range r.ModelServingEndpoints {
		r.ModelServingEndpoints[i].Name = normalizePrefix(prefix) + r.ModelServingEndpoints[i].Name

		// As of 2024-06, model serving endpoints don't yet support tags
	}

	// Registered models presets: Prefix
	for i := range r.RegisteredModels {
		r.RegisteredModels[i].Name = normalizePrefix(prefix) + r.RegisteredModels[i].Name

		// As of 2024-06, registered models don't yet support tags
	}

	// Quality monitors presets: Prefix
	if t.TriggerPauseStatus == config.Paused {
		for i := range r.QualityMonitors {
			// Remove all schedules from monitors, since they don't support pausing/unpausing.
			// Quality monitors might support the "pause" property in the future, so at the
			// CLI level we do respect that property if it is set to "unpaused."
			if r.QualityMonitors[i].Schedule != nil && r.QualityMonitors[i].Schedule.PauseStatus != catalog.MonitorCronSchedulePauseStatusUnpaused {
				r.QualityMonitors[i].Schedule = nil
			}
		}
	}

	// Schemas: Prefix
	for i := range r.Schemas {
		r.Schemas[i].Name = normalizePrefix(prefix) + r.Schemas[i].Name
		// HTTP API for schemas doesn't yet support tags. It's only supported in
		// the Databricks UI and via the SQL API.
	}

	return nil
}

func validatePauseStatus(b *bundle.Bundle) diag.Diagnostics {
	p := b.Config.Presets.TriggerPauseStatus
	if p == "" || p == config.Paused || p == config.Unpaused {
		return nil
	}
	return diag.Diagnostics{{
		Summary:   "Invalid value for trigger_pause_status, should be PAUSED or UNPAUSED",
		Severity:  diag.Error,
		Locations: []dyn.Location{b.Config.GetLocation("presets.trigger_pause_status")},
	}}
}

// toTagArray converts a map of tags to an array of tags.
// We sort tags so ensure stable ordering.
func toTagArray(tags map[string]string) []Tag {
	var tagArray []Tag
	if tags == nil {
		return tagArray
	}
	for key, value := range tags {
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
	prefix = strings.Trim(prefix, " ")

	// If the prefix ends with a ']', we add an underscore to the end.
	// This makes sure that we get names like "dev_user_endpoint" instead of "dev_userendpoint"
	suffix := ""
	if strings.HasSuffix(prefix, "]") {
		suffix = "_"
	}

	return textutil.NormalizeString(prefix) + suffix
}
