package mutator

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type applyPresetsCatalogSchema struct{}

// ApplyPresetsCatalogSchema applies catalog and schema presets to bundle resources.
func ApplyPresetsCatalogSchema() *applyPresetsCatalogSchema {
	return &applyPresetsCatalogSchema{}
}

func (m *applyPresetsCatalogSchema) Name() string {
	return "ApplyPresetsCatalogSchema"
}

func (m *applyPresetsCatalogSchema) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := validateCatalogAndSchema(b)
	if diags.HasError() {
		return diags
	}

	r := b.Config.Resources
	p := b.Config.Presets

	// Jobs
	for key, j := range r.Jobs {
		if j.JobSettings == nil {
			continue
		}
		if p.Catalog != "" || p.Schema != "" {
			for _, task := range j.Tasks {
				if task.DbtTask != nil {
					if task.DbtTask.Catalog == "" {
						task.DbtTask.Catalog = p.Catalog
					}
					if task.DbtTask.Schema == "" {
						task.DbtTask.Schema = p.Schema
					}
				}
			}

			diags = diags.Extend(addCatalogSchemaParameters(b, key, j, p))
			diags = diags.Extend(recommendCatalogSchemaUsage(b, ctx, key, j))
		}
	}

	// Pipelines
	for _, pl := range r.Pipelines {
		if pl.PipelineSpec == nil {
			continue
		}
		if p.Catalog != "" && p.Schema != "" {
			if pl.Catalog == "" {
				pl.Catalog = p.Catalog
			}
			if pl.Target == "" {
				pl.Target = p.Schema
			}
			if pl.GatewayDefinition != nil {
				if pl.GatewayDefinition.GatewayStorageCatalog == "" {
					pl.GatewayDefinition.GatewayStorageCatalog = p.Catalog
				}
				if pl.GatewayDefinition.GatewayStorageSchema == "" {
					pl.GatewayDefinition.GatewayStorageSchema = p.Schema
				}
			}
			if pl.IngestionDefinition != nil {
				for _, obj := range pl.IngestionDefinition.Objects {
					if obj.Report != nil {
						if obj.Report.DestinationCatalog == "" {
							obj.Report.DestinationCatalog = p.Catalog
						}
						if obj.Report.DestinationSchema == "" {
							obj.Report.DestinationSchema = p.Schema
						}
					}
					if obj.Schema != nil {
						if obj.Schema.SourceCatalog == "" {
							obj.Schema.SourceCatalog = p.Catalog
						}
						if obj.Schema.SourceSchema == "" {
							obj.Schema.SourceSchema = p.Schema
						}
						if obj.Schema.DestinationCatalog == "" {
							obj.Schema.DestinationCatalog = p.Catalog
						}
						if obj.Schema.DestinationSchema == "" {
							obj.Schema.DestinationSchema = p.Schema
						}
					}
					if obj.Table != nil {
						if obj.Table.SourceCatalog == "" {
							obj.Table.SourceCatalog = p.Catalog
						}
						if obj.Table.SourceSchema == "" {
							obj.Table.SourceSchema = p.Schema
						}
						if obj.Table.DestinationCatalog == "" {
							obj.Table.DestinationCatalog = p.Catalog
						}
						if obj.Table.DestinationSchema == "" {
							obj.Table.DestinationSchema = p.Schema
						}
					}
				}
			}
		}
	}

	// Model serving endpoints
	for _, e := range r.ModelServingEndpoints {
		if e.CreateServingEndpoint == nil {
			continue
		}

		if p.Catalog != "" || p.Schema != "" {
			if e.CreateServingEndpoint.AiGateway != nil && e.CreateServingEndpoint.AiGateway.InferenceTableConfig != nil {
				if p.Catalog != "" && e.CreateServingEndpoint.AiGateway.InferenceTableConfig.CatalogName == "" {
					e.CreateServingEndpoint.AiGateway.InferenceTableConfig.CatalogName = p.Catalog
				}
				if p.Schema != "" && e.CreateServingEndpoint.AiGateway.InferenceTableConfig.SchemaName == "" {
					e.CreateServingEndpoint.AiGateway.InferenceTableConfig.SchemaName = p.Schema
				}
			}

			if e.CreateServingEndpoint.Config.AutoCaptureConfig != nil {
				if p.Catalog != "" && e.CreateServingEndpoint.Config.AutoCaptureConfig.CatalogName == "" {
					e.CreateServingEndpoint.Config.AutoCaptureConfig.CatalogName = p.Catalog
				}
				if p.Schema != "" && e.CreateServingEndpoint.Config.AutoCaptureConfig.SchemaName == "" {
					e.CreateServingEndpoint.Config.AutoCaptureConfig.SchemaName = p.Schema
				}
			}

			for i := range e.CreateServingEndpoint.Config.ServedEntities {
				e.CreateServingEndpoint.Config.ServedEntities[i].EntityName = fullyQualifyName(
					e.CreateServingEndpoint.Config.ServedEntities[i].EntityName, p.Catalog, p.Schema,
				)
			}
			for i := range e.CreateServingEndpoint.Config.ServedModels {
				e.CreateServingEndpoint.Config.ServedModels[i].ModelName = fullyQualifyName(
					e.CreateServingEndpoint.Config.ServedModels[i].ModelName, p.Catalog, p.Schema,
				)
			}
		}
	}

	// Registered models
	for _, m := range r.RegisteredModels {
		if m.CreateRegisteredModelRequest == nil {
			continue
		}
		if p.Catalog != "" && m.CatalogName == "" {
			m.CatalogName = p.Catalog
		}
		if p.Schema != "" && m.SchemaName == "" {
			m.SchemaName = p.Schema
		}
	}

	// Quality monitors
	for _, q := range r.QualityMonitors {
		if q.CreateMonitor == nil {
			continue
		}
		if p.Catalog != "" && p.Schema != "" {
			q.TableName = fullyQualifyName(q.TableName, p.Catalog, p.Schema)
			if q.OutputSchemaName == "" {
				q.OutputSchemaName = p.Catalog + "." + p.Schema
			}
		}
	}

	// Schemas
	for _, s := range r.Schemas {
		if s.CreateSchema == nil {
			continue
		}
		if p.Catalog != "" && s.CatalogName == "" {
			s.CatalogName = p.Catalog
		}
		if p.Schema != "" && s.Name == "" {
			// If there is a schema preset such as 'dev', we directly
			// use that name and don't add any prefix (which might result in dev_dev).
			s.Name = p.Schema
		}
	}

	return diags
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
	return diag.Diagnostics{}
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
					Locations: b.Config.GetLocations("resources.jobs." + key + ".parameters"),
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
				Summary:  "Use the 'catalog' and 'schema' parameters provided via 'presets.catalog' and 'presets.schema' using\n\n" + fix,
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
