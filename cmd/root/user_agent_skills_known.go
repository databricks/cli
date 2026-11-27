package root

import "slices"

// knownSkills lists every skill name Databricks has published, current and
// retired. Only these names are reported in skill/<name> user agent pairs.
//
// The skills-invoked value comes from whatever agent runs the CLI, and agents
// pass all the skills they loaded, not only Databricks ones. A third-party or
// customer-written skill name can identify a customer or describe their
// internal work, so it must never leave the machine. A prefix check is not
// enough: customers name their own skills "databricks-<something>" too.
//
// Sources:
//   - https://github.com/databricks/databricks-agent-skills: every SKILL.md
//     directory name across the repository's history, including skills that were
//     renamed (databricks -> databricks-core, databricks-bundles -> databricks-dabs)
//     or retired, so older installs keep reporting.
//   - The databricks-quickstart skill printed by `databricks quickstart --skill`.
//
// When a new skill is published, add its name here. Until a CLI release includes
// it, its usage is not reported. It is a slice, not a map, so it costs nothing
// at startup.
var knownSkills = []string{
	"data-app-design",
	"databricks",
	"databricks-agent-bricks",
	"databricks-ai-functions",
	"databricks-ai-runtime",
	"databricks-aibi-dashboards",
	"databricks-app-design",
	"databricks-apps",
	"databricks-apps-python",
	"databricks-bundles",
	"databricks-cli",
	"databricks-cli-auth",
	"databricks-cli-install",
	"databricks-config",
	"databricks-core",
	"databricks-dabs",
	"databricks-data-discovery",
	"databricks-dbsql",
	"databricks-docs",
	"databricks-execution-compute",
	"databricks-genie",
	"databricks-genie-agents",
	"databricks-iceberg",
	"databricks-jobs",
	"databricks-lakebase",
	"databricks-lakebase-autoscale",
	"databricks-lakebase-provisioned",
	"databricks-lakeflow-connect",
	"databricks-lakeflow-connect-saas",
	"databricks-lakehouse-mcp",
	"databricks-metric-view-advisor",
	"databricks-metric-views",
	"databricks-ml-training",
	"databricks-mlflow-evaluation",
	"databricks-mlflow-tracing",
	"databricks-model-serving",
	"databricks-pipelines",
	"databricks-python-sdk",
	"databricks-quickstart",
	"databricks-sandbox",
	"databricks-serverless",
	"databricks-serverless-migration",
	"databricks-serverless-migration-api",
	"databricks-setup-local",
	"databricks-spark-declarative-pipelines",
	"databricks-spark-structured-streaming",
	"databricks-synthetic-data-gen",
	"databricks-unity-catalog",
	"databricks-unity-gateway",
	"databricks-unstructured-pdf-generation",
	"databricks-vector-search",
	"databricks-zerobus-ingest",
	"lakeflow-unstructured",
	"spark-python-data-source",
}

// isKnownSkill reports whether name is a Databricks-published skill.
func isKnownSkill(name string) bool {
	return slices.Contains(knownSkills, name)
}
