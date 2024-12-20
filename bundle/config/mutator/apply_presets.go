package mutator

import (
	"context"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dbr"
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
//
// Note that the catalog/schema presets are applied in ApplyPresetsCatalogSchema.
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
	var diags diag.Diagnostics

	if d := validatePauseStatus(b); d != nil {
		diags = diags.Extend(d)
	}

	r := b.Config.Resources
	t := b.Config.Presets
	prefix := t.NamePrefix
	tags := toTagArray(t.Tags)

	// Jobs presets: Prefix, Tags, JobsMaxConcurrentRuns, TriggerPauseStatus
	for key, j := range r.Jobs {
		if j.JobSettings == nil {
			diags = diags.Extend(diag.Errorf("job %s is not defined", key))
			continue
		}
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
	// Not supported: Tags (not in API as of 2024-12)
	for key, p := range r.Pipelines {
		if p.PipelineSpec == nil {
			diags = diags.Extend(diag.Errorf("pipeline %s is not defined", key))
			continue
		}
		p.Name = prefix + p.Name
		if config.IsExplicitlyEnabled(t.PipelinesDevelopment) {
			p.Development = true
		}
		if t.TriggerPauseStatus == config.Paused {
			p.Continuous = false
		}
	}

	// Models presets: Prefix, Tags
	for key, m := range r.Models {
		if m.Model == nil {
			diags = diags.Extend(diag.Errorf("model %s is not defined", key))
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
	for key, e := range r.Experiments {
		if e.Experiment == nil {
			diags = diags.Extend(diag.Errorf("experiment %s is not defined", key))
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
	// Not supported: Tags (not in API as of 2024-12)
	for key, e := range r.ModelServingEndpoints {
		if e.CreateServingEndpoint == nil {
			diags = diags.Extend(diag.Errorf("model serving endpoint %s is not defined", key))
			continue
		}
		e.Name = normalizePrefix(prefix) + e.Name
	}

	// Registered models presets: Prefix
	// Not supported: Tags (not in API as of 2024-12)
	for key, m := range r.RegisteredModels {
		if m.CreateRegisteredModelRequest == nil {
			diags = diags.Extend(diag.Errorf("registered model %s is not defined", key))
			continue
		}
		m.Name = normalizePrefix(prefix) + m.Name
	}

	// Quality monitors presets: Schedule
	// Not supported: Tags (not in API as of 2024-12)
	for key, q := range r.QualityMonitors {
		if q.CreateMonitor == nil {
			diags = diags.Extend(diag.Errorf("quality monitor %s is not defined", key))
			continue
		}
		// Remove all schedules from monitors, since they don't support pausing/unpausing.
		// Quality monitors might support the "pause" property in the future, so at the
		// CLI level we do respect that property if it is set to "unpaused."
		if t.TriggerPauseStatus == config.Paused {
			if q.Schedule != nil && q.Schedule.PauseStatus != catalog.MonitorCronSchedulePauseStatusUnpaused {
				q.Schedule = nil
			}
		}
	}

	// Schemas: Prefix
	// Not supported: Tags (only supported in Databricks UI / SQL API as of 2024-12)
	for key, s := range r.Schemas {
		if s.CreateSchema == nil {
			diags = diags.Extend(diag.Errorf("schema %s is not defined", key))
			continue
		}
		s.Name = normalizePrefix(prefix) + s.Name
	}

	// Clusters: Prefix, Tags
	for key, c := range r.Clusters {
		if c.ClusterSpec == nil {
			diags = diags.Extend(diag.Errorf("cluster %s is not defined", key))
			continue
		}
		c.ClusterName = prefix + c.ClusterName
		if c.CustomTags == nil {
			c.CustomTags = make(map[string]string)
		}
		for _, tag := range tags {
			normalizedKey := b.Tagging.NormalizeKey(tag.Key)
			normalizedValue := b.Tagging.NormalizeValue(tag.Value)
			if _, ok := c.CustomTags[normalizedKey]; !ok {
				c.CustomTags[normalizedKey] = normalizedValue
			}
		}
	}

	// Dashboards: Prefix
	for key, dashboard := range r.Dashboards {
		if dashboard == nil || dashboard.Dashboard == nil {
			diags = diags.Extend(diag.Errorf("dashboard %s s is not defined", key))
			continue
		}
		dashboard.DisplayName = prefix + dashboard.DisplayName
	}

	if config.IsExplicitlyEnabled((b.Config.Presets.SourceLinkedDeployment)) {
		isDatabricksWorkspace := dbr.RunsOnRuntime(ctx) && strings.HasPrefix(b.SyncRootPath, "/Workspace/")
		if !isDatabricksWorkspace {
			target := b.Config.Bundle.Target
			path := dyn.NewPath(dyn.Key("targets"), dyn.Key(target), dyn.Key("presets"), dyn.Key("source_linked_deployment"))
			diags = diags.Append(
				diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "source-linked deployment is available only in the Databricks Workspace",
					Paths: []dyn.Path{
						path,
					},
					Locations: b.Config.GetLocations(path[2:].String()),
				},
			)

			disabled := false
			b.Config.Presets.SourceLinkedDeployment = &disabled
		}
	}

	return diags
}

// validatePauseStatus checks the user-provided pause status is valid.
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
// We sort tags to ensure stable ordering.
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
