package mutator

import (
	"context"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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
	var diags diag.Diagnostics

	if d := validateCatalogAndSchema(b); d != nil {
		return d // fast fail since code below would fail
	}
	if d := validatePauseStatus(b); d != nil {
		diags = diags.Extend(d)
	}

	r := b.Config.Resources
	t := b.Config.Presets
	prefix := t.NamePrefix
	tags := toTagArray(t.Tags)

	// Jobs presets: Prefix, Tags, JobsMaxConcurrentRuns, TriggerPauseStatus, Catalog, Schema
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
		if t.Catalog != "" || t.Schema != "" {
			diags = diags.Extend(validateJobUsesCatalogAndSchema(b, key, j))
		}
	}

	// Pipelines presets: Prefix, PipelinesDevelopment, Catalog, Schema
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
		if t.Catalog != "" && p.Catalog == "" {
			p.Catalog = t.Catalog
		}
		if t.Schema != "" && p.Target == "" {
			p.Target = t.Schema
		}
		// As of 2024-10, pipelines don't yet support tags
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
	for key, e := range r.ModelServingEndpoints {
		if e.CreateServingEndpoint == nil {
			diags = diags.Extend(diag.Errorf("model serving endpoint %s is not defined", key))
			continue
		}
		e.Name = normalizePrefix(prefix) + e.Name

		// As of 2024-06, model serving endpoints don't yet support tags
	}

	// Registered models presets: Prefix, Catalog, Schema
	for key, m := range r.RegisteredModels {
		if m.CreateRegisteredModelRequest == nil {
			diags = diags.Extend(diag.Errorf("registered model %s is not defined", key))
			continue
		}
		m.Name = normalizePrefix(prefix) + m.Name
		if t.Catalog != "" && m.CatalogName == "" {
			m.CatalogName = t.Catalog
		}
		if t.Schema != "" && m.SchemaName == "" {
			m.SchemaName = t.Schema
		}
		// As of 2024-06, registered models don't yet support tags
	}

	// Quality monitors presets: Schedule, Catalog, Schema
	if t.TriggerPauseStatus == config.Paused {
		for key, q := range r.QualityMonitors {
			if q.CreateMonitor == nil {
				diags = diags.Extend(diag.Errorf("quality monitor %s is not defined", key))
				continue
			}
			// Remove all schedules from monitors, since they don't support pausing/unpausing.
			// Quality monitors might support the "pause" property in the future, so at the
			// CLI level we do respect that property if it is set to "unpaused."
			if q.Schedule != nil && q.Schedule.PauseStatus != catalog.MonitorCronSchedulePauseStatusUnpaused {
				q.Schedule = nil
			}
			if t.Catalog != "" && t.Schema != "" {
				parts := strings.Split(q.TableName, ".")
				if len(parts) != 3 {
					q.TableName = fmt.Sprintf("%s.%s.%s", t.Catalog, t.Schema, q.TableName)
				}
			}
		}
	}

	// Schemas: Prefix, Catalog, Schema
	for key, s := range r.Schemas {
		if s.CreateSchema == nil {
			diags = diags.Extend(diag.Errorf("schema %s is not defined", key))
			continue
		}
		s.Name = normalizePrefix(prefix) + s.Name
		if t.Catalog != "" && s.CatalogName == "" {
			s.CatalogName = t.Catalog
		}
		if t.Schema != "" && s.Name == "" {
			// If there is a schema preset such as 'dev', we directly
			// use that name and don't add any prefix (which might result in dev_dev).
			s.Name = t.Schema
		}
		// HTTP API for schemas doesn't yet support tags. It's only supported in
		// the Databricks UI and via the SQL API.
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

func validateCatalogAndSchema(b *bundle.Bundle) diag.Diagnostics {
	p := b.Config.Presets
	if (p.Catalog != "" && p.Schema == "") || (p.Catalog == "" && p.Schema != "") {
		return diag.Diagnostics{{
			Summary:   "presets.catalog and presets.schema must always be set together",
			Severity:  diag.Error,
			Locations: []dyn.Location{b.Config.GetLocation("presets")},
		}}
	}
	return nil
}

func validateJobUsesCatalogAndSchema(b *bundle.Bundle, key string, job *resources.Job) diag.Diagnostics {
	if !hasParameter(job.Parameters, "catalog") || !hasParameter(job.Parameters, "schema") {
		return diag.Diagnostics{{
			Summary: fmt.Sprintf("job %s must pass catalog and schema presets as parameters as follows:\n"+
				"  parameters:\n"+
				"    - name: catalog:\n"+
				"      default: ${presets.catalog}\n"+
				"    - name: schema\n"+
				"      default: ${presets.schema}\n", key),
			Severity:  diag.Error,
			Locations: []dyn.Location{b.Config.GetLocation("resources.jobs." + key)},
		}}
	}
	return nil
}

func hasParameter(parameters []jobs.JobParameterDefinition, name string) bool {
	if parameters == nil {
		return false
	}
	for _, p := range parameters {
		if p.Name == name {
			return true
		}
	}
	return false
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
