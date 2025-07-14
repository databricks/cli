package resourcemutator

import (
	"context"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type applyPresets struct{}

// Apply all presets, e.g. the prefix presets that
// adds a prefix to all names of all resources.
func ApplyPresets() bundle.Mutator {
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
	var diags diag.Diagnostics

	if d := validatePauseStatus(b); d != nil {
		diags = diags.Extend(d)
	}

	r := b.Config.Resources
	t := b.Config.Presets
	prefix := t.NamePrefix

	b.Metrics.AddBoolValue(metrics.PresetsNamePrefixIsSet, prefix != "")

	tags := toTagArray(t.Tags)

	// Jobs presets: Prefix, Tags, JobsMaxConcurrentRuns, TriggerPauseStatus
	for _, j := range r.Jobs {
		// Here and for other resources we follow the same approach for nil checks:
		// If resource itself is nil, ignore it (It is currently allowed, but should have generic check with location-enriched diagnostics that disallows nils anywhere).
		// If embedded pointer is nil, initialize it with empty struct. The fact that we embed pointer struct is implementation detail.
		// It will remain nil if there are no fields in it that are provided by users, but that maybe ok, users could be relying on presets or defaults.
		if j == nil {
			continue
		}
		j.Name = prefix + j.Name
		if len(tags) > 0 {
			if j.Tags == nil {
				// Note: only create this map if tags is not empty, to avoid inserting "tags: {}" entry in the config
				j.Tags = make(map[string]string, len(tags))
			}
			for _, tag := range tags {
				if j.Tags[tag.Key] == "" {
					j.Tags[tag.Key] = tag.Value
				}
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
	for _, p := range r.Pipelines {
		if p == nil {
			continue
		}
		p.Name = prefix + p.Name
		if config.IsExplicitlyEnabled(t.PipelinesDevelopment) {
			p.Development = true
		}
		if t.TriggerPauseStatus == config.Paused {
			p.Continuous = false
		}
		// As of 2024-06, pipelines don't yet support tags
	}

	// Models presets: Prefix, Tags
	for _, m := range r.Models {
		if m == nil {
			continue
		}
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
		if e == nil {
			continue
		}
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
	for _, e := range r.ModelServingEndpoints {
		if e == nil {
			continue
		}
		e.Name = normalizePrefix(prefix) + e.Name

		// As of 2024-06, model serving endpoints don't yet support tags
	}

	// Registered models presets: Prefix
	for _, m := range r.RegisteredModels {
		if m == nil {
			continue
		}
		m.Name = normalizePrefix(prefix) + m.Name

		// As of 2024-06, registered models don't yet support tags
	}

	// Quality monitors presets: Schedule
	if t.TriggerPauseStatus == config.Paused {
		for _, q := range r.QualityMonitors {
			// Remove all schedules from monitors, since they don't support pausing/unpausing.
			// Quality monitors might support the "pause" property in the future, so at the
			// CLI level we do respect that property if it is set to "unpaused."
			if q.Schedule != nil && q.Schedule.PauseStatus != catalog.MonitorCronSchedulePauseStatusUnpaused {
				q.Schedule = nil
			}
		}
	}

	// Schemas: Prefix
	for _, s := range r.Schemas {
		if b.Config.Experimental != nil && b.Config.Experimental.SkipNamePrefixForSchema {
			break
		}

		if s == nil {
			continue
		}
		s.Name = normalizePrefix(prefix) + s.Name
		// HTTP API for schemas doesn't yet support tags. It's only supported in
		// the Databricks UI and via the SQL API.
	}

	// Clusters: Prefix, Tags
	for _, c := range r.Clusters {
		if c == nil {
			continue
		}
		c.ClusterName = prefix + c.ClusterName
		if len(tags) > 0 {
			if c.CustomTags == nil {
				c.CustomTags = make(map[string]string, len(tags))
			}
			for _, tag := range tags {
				normalisedKey := b.Tagging.NormalizeKey(tag.Key)
				normalisedValue := b.Tagging.NormalizeValue(tag.Value)
				if _, ok := c.CustomTags[normalisedKey]; !ok {
					c.CustomTags[normalisedKey] = normalisedValue
				}
			}
		}
	}

	// Dashboards: Prefix
	for _, dashboard := range r.Dashboards {
		if dashboard == nil {
			continue
		}
		dashboard.DisplayName = prefix + dashboard.DisplayName
	}

	// Apps: No presets

	// SQL Warehouses: Prefix, Tags
	for _, w := range r.SqlWarehouses {
		if w == nil {
			continue
		}
		w.Name = prefix + w.Name
		if len(tags) > 0 {
			if w.Tags == nil {
				w.Tags = &sql.EndpointTags{}
			}
			for _, tag := range tags {
				normalisedKey := b.Tagging.NormalizeKey(tag.Key)
				normalisedValue := b.Tagging.NormalizeValue(tag.Value)

				// Check if the tag already exists
				exists := false
				for _, t := range w.Tags.CustomTags {
					if t.Key == normalisedKey {
						exists = true
						break
					}
				}

				if !exists {
					w.Tags.CustomTags = append(w.Tags.CustomTags, sql.EndpointTagPair{
						Key:   normalisedKey,
						Value: normalisedValue,
					})
				}
			}
		}
	}

	return diags
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
