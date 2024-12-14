package mutator

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
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

	// Jobs presets.
	// Supported: Prefix, Tags, JobsMaxConcurrentRuns, TriggerPauseStatus, Catalog, Schema
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
			for _, task := range j.Tasks {
				if task.DbtTask != nil {
					if task.DbtTask.Catalog == "" {
						task.DbtTask.Catalog = t.Catalog
					}
					if task.DbtTask.Schema == "" {
						task.DbtTask.Schema = t.Schema
					}
				}
			}

			diags = diags.Extend(addCatalogSchemaParameters(b, key, j, t))
			diags = diags.Extend(recommendCatalogSchemaUsage(b, ctx, key, j))
		}
	}

	// Pipelines presets.
	// Supported: Prefix, PipelinesDevelopment, Catalog, Schema
	// Not supported: Tags (as of 2024-10 not in pipelines API)
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
		// TODO: add recommendation when catalog is already set?
		if t.Catalog != "" && p.Catalog == "" && p.Catalog != "hive_metastore" {
			p.Catalog = t.Catalog
		}
		if t.Schema != "" && p.Target == "" {
			p.Target = t.Schema
		}
	}

	// Models presets
	// Supported: Prefix, Tags
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

	// Experiments presets
	// Supported: Prefix, Tags
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

	// Model serving endpoint presets
	// Supported: Prefix, Catalog, Schema
	// Not supported: Tags (not in API as of 2024-10)
	for key, e := range r.ModelServingEndpoints {
		if e.CreateServingEndpoint == nil {
			diags = diags.Extend(diag.Errorf("model serving endpoint %s is not defined", key))
			continue
		}
		e.Name = normalizePrefix(prefix) + e.Name

		if t.Catalog != "" || t.Schema != "" {
			// Apply catalog & schema to inference table config if not set
			if e.CreateServingEndpoint.AiGateway != nil && e.CreateServingEndpoint.AiGateway.InferenceTableConfig != nil {
				if t.Catalog != "" && e.CreateServingEndpoint.AiGateway.InferenceTableConfig.CatalogName == "" {
					e.CreateServingEndpoint.AiGateway.InferenceTableConfig.CatalogName = t.Catalog
				}
				if t.Schema != "" && e.CreateServingEndpoint.AiGateway.InferenceTableConfig.SchemaName == "" {
					e.CreateServingEndpoint.AiGateway.InferenceTableConfig.SchemaName = t.Schema
				}
			}

			// Apply catalog & schema to auto capture config if not set
			if e.CreateServingEndpoint.Config.AutoCaptureConfig != nil {
				if t.Catalog != "" && e.CreateServingEndpoint.Config.AutoCaptureConfig.CatalogName == "" {
					e.CreateServingEndpoint.Config.AutoCaptureConfig.CatalogName = t.Catalog
				}
				if t.Schema != "" && e.CreateServingEndpoint.Config.AutoCaptureConfig.SchemaName == "" {
					e.CreateServingEndpoint.Config.AutoCaptureConfig.SchemaName = t.Schema
				}
			}

			// Fully qualify served entities and models if they are not already qualified
			for i := range e.CreateServingEndpoint.Config.ServedEntities {
				e.CreateServingEndpoint.Config.ServedEntities[i].EntityName = fullyQualifyName(
					e.CreateServingEndpoint.Config.ServedEntities[i].EntityName, t.Catalog, t.Schema,
				)
			}
			for i := range e.CreateServingEndpoint.Config.ServedModels {
				e.CreateServingEndpoint.Config.ServedModels[i].ModelName = fullyQualifyName(
					e.CreateServingEndpoint.Config.ServedModels[i].ModelName, t.Catalog, t.Schema,
				)
			}
		}
	}

	// Registered models presets
	// Supported: Prefix, Catalog, Schema
	// Not supported: Tags (not in API as of 2024-10)
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
	}

	// Quality monitors presets
	// Supported: Schedule, Catalog, Schema
	// Not supported: Tags (not in API as of 2024-10)
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
	// Not supported: Tags (as of 2024-10, only supported in Databricks UI / SQL API)
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
	}

	// Clusters: Prefix, Tags
	// Not supported: Catalog / Schema (not applicable)
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

// addCatalogSchemaParameters adds catalog and schema parameters to a job if they don't already exist.
// Returns any warning diagnostics for existing parameters.
func addCatalogSchemaParameters(b *bundle.Bundle, key string, job *resources.Job, t config.Presets) diag.Diagnostics {
	var diags diag.Diagnostics

	// Check for existing catalog/schema parameters
	hasCatalog := false
	hasSchema := false
	if job.Parameters != nil {
		for _, param := range job.Parameters {
			if param.Name == "catalog" {
				hasCatalog = true
				diags = diags.Extend(diag.Diagnostics{{
					Summary:   fmt.Sprintf("job %s already has 'catalog' parameter defined; ignoring preset value", key),
					Severity:  diag.Warning,
					Locations: []dyn.Location{b.Config.GetLocation("resources.jobs." + key)},
				}})
			}
			if param.Name == "schema" {
				hasSchema = true
				diags = diags.Extend(diag.Diagnostics{{
					Summary:   fmt.Sprintf("job %s already has 'schema' parameter defined; ignoring preset value", key),
					Severity:  diag.Warning,
					Locations: []dyn.Location{b.Config.GetLocation("resources.jobs." + key)},
				}})
			}
		}
	}

	// Initialize parameters if nil
	if job.Parameters == nil {
		job.Parameters = []jobs.JobParameterDefinition{}
	}

	// Add catalog parameter if not already present
	if !hasCatalog && t.Catalog != "" {
		job.Parameters = append(job.Parameters, jobs.JobParameterDefinition{
			Name:    "catalog",
			Default: t.Catalog,
		})
	}

	// Add schema parameter if not already present
	if !hasSchema && t.Schema != "" {
		job.Parameters = append(job.Parameters, jobs.JobParameterDefinition{
			Name:    "schema",
			Default: t.Schema,
		})
	}

	return diags
}

func recommendCatalogSchemaUsage(b *bundle.Bundle, ctx context.Context, key string, job *resources.Job) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, t := range job.Tasks {
		var relPath string
		var expected string
		var fix string
		if t.NotebookTask != nil {
			relPath = t.NotebookTask.NotebookPath
			expected = `dbutils.widgets.text\(['"]schema|` +
				`USE[^)]+schema`
			fix = "  dbutils.widgets.text('catalog')\n" +
				"  dbutils.widgets.text('schema')\n" +
				"  catalog = dbutils.widgets.get('catalog')\n" +
				"  schema = dbutils.widgets.get('schema')\n" +
				"  spark.sql(f'USE {catalog}.{schema}')\n"
		} else if t.SparkPythonTask != nil {
			relPath = t.SparkPythonTask.PythonFile
			expected = `add_argument\(['"]--catalog'|` +
				`USE[^)]+catalog`
			fix = "  def main():\n" +
				"    parser = argparse.ArgumentParser()\n" +
				"    parser.add_argument('--catalog', required=True)\n" +
				"    parser.add_argument('--schema', '-s', required=True)\n" +
				"    args, unknown = parser.parse_known_args()\n" +
				"    spark.sql(f\"USE {args.catalog}.{args.schema}\")\n"
		} else if t.SqlTask != nil && t.SqlTask.File != nil {
			relPath = t.SqlTask.File.Path
			expected = `:schema|\{\{schema\}\}`
			fix = "  USE CATALOG {{catalog}};\n" +
				"  USE IDENTIFIER({schema});\n"
		} else {
			continue
		}

		sourceDir, err := b.Config.GetLocation("resources.jobs." + key).Directory()
		if err != nil {
			continue
		}

		localPath, _, err := GetLocalPath(ctx, b, sourceDir, relPath)
		if err != nil || localPath == "" {
			// We ignore errors (they're reported by another mutator)
			// and ignore empty local paths (which means we'd have to download the file)
			continue
		}

		if !fileIncludesPattern(ctx, localPath, expected) {
			diags = diags.Extend(diag.Diagnostics{{
				Summary: fmt.Sprintf("Use the 'catalog' and 'schema' parameters provided via 'presets.catalog' and 'presets.schema' using\n\n" +
					fix),
				Severity: diag.Recommendation,
				Locations: []dyn.Location{{
					File:   localPath,
					Line:   1,
					Column: 1,
				}},
			}})
		}
	}

	return diags

}

// fullyQualifyName checks if the given name is already qualified with a catalog and schema.
// If not, and both catalog and schema are available, it prefixes the name with catalog.schema.
// If name is empty, returns name as-is.
func fullyQualifyName(name, catalog, schema string) string {
	if name == "" || catalog == "" || schema == "" {
		return name
	}
	// If it's already qualified (contains at least two '.'), we assume it's fully qualified.
	parts := strings.Split(name, ".")
	if len(parts) >= 3 {
		// Already fully qualified
		return name
	}
	// Otherwise, fully qualify it
	return fmt.Sprintf("%s.%s.%s", catalog, schema, name)
}

func fileIncludesPattern(ctx context.Context, filePath string, expected string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Warnf(ctx, "failed to check file %s: %v", filePath, err)
		return true
	}

	matched, err := regexp.MatchString(expected, string(content))
	if err != nil {
		log.Warnf(ctx, "failed to check pattern in %s: %v", filePath, err)
		return true
	}
	return matched
}
