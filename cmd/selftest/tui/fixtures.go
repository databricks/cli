package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/cmdio"
)

// deploymentChoices feeds the AskSelect scenario; the entries vary in
// length so the rendering is tested across short, medium, and wrap-width
// options.
var deploymentChoices = []string{
	"Deploy bundle with Terraform",
	"Deploy bundle with the direct engine",
	"Validate bundle configuration",
	"Generate deployment artifacts only",
	"Run pre-deploy checks",
	"Skip deploy and tail logs",
	"Cancel",
	"Show me the full plan output before deciding",
}

type spinnerMessage struct {
	text     string
	duration time.Duration
}

var spinnerMessages = []spinnerMessage{
	{"Initializing...", time.Second},
	{"Loading configuration", time.Second},
	{"Connecting to workspace", time.Second},
	{"Processing files", time.Second},
	{"Finalizing", time.Second},
}

// databricksFeatures is a stable list of Databricks product / feature names
// used as fixture data for the prompt scenarios. Drawn from the public docs
// (https://docs.databricks.com) so the demo data looks like something a user
// would actually encounter.
var databricksFeatures = []string{
	"unity-catalog",
	"delta-lake",
	"delta-sharing",
	"photon",
	"mlflow",
	"mosaic-ai",
	"genie",
	"lakeflow-connect",
	"lakeflow-jobs",
	"vector-search",
	"model-serving",
	"feature-store",
	"databricks-sql",
	"ai-playground",
	"foundation-models",
	"lakehouse-monitoring",
	"liquid-clustering",
	"predictive-optimization",
	"governed-tags",
	"lakeflow-designer",
}

// buildItems uses zero-padded ids so the alphabetical Select scenario has
// a stable sort order.
func buildItems(n int) []cmdio.Tuple {
	n = min(n, len(databricksFeatures))
	items := make([]cmdio.Tuple, 0, n)
	for i := range n {
		items = append(items, cmdio.Tuple{
			Name: databricksFeatures[i],
			Id:   fmt.Sprintf("id-%02d", i+1),
		})
	}
	return items
}

// buildFilterItems returns 15 items where 5 share the substring "lake", so
// progressive typing narrows the list, and a non-matching substring ("xyz")
// hits the "No results" path.
func buildFilterItems() []cmdio.Tuple {
	names := []string{
		"lakehouse-monitoring",
		"lakeflow-connect",
		"lakeflow-jobs",
		"delta-lake",
		"lakebase-postgres",
		"unity-catalog",
		"mosaic-ai",
		"vector-search",
		"model-serving",
		"feature-store",
		"ai-playground",
		"genie-spaces",
		"mlflow-tracking",
		"liquid-clustering",
		"predictive-optimization",
	}
	items := make([]cmdio.Tuple, 0, len(names))
	for i, name := range names {
		items = append(items, cmdio.Tuple{
			Name: name,
			Id:   fmt.Sprintf("id-%02d", i+1),
		})
	}
	return items
}

// buildLongItems uses fully-qualified workspace URLs as ids so that the
// rendered field overflows a typical terminal width.
func buildLongItems() []cmdio.Tuple {
	hosts := []string{
		"https://adb-1234567890123456.78.azuredatabricks.net/?o=1234567890123456",
		"https://adb-2345678901234567.89.azuredatabricks.net/?o=2345678901234567",
		"https://acme-prod.cloud.databricks.com/?o=3456789012345678",
		"https://acme-staging.cloud.databricks.com/?o=4567890123456789",
		"https://acme-dev.cloud.databricks.com/?o=5678901234567890",
		"https://1234567890123456.7.gcp.databricks.com/?o=6789012345678901",
		"https://2345678901234567.8.gcp.databricks.com/?o=7890123456789012",
		"https://field-eng-east.cloud.databricks.com/?o=8901234567890123",
	}
	items := make([]cmdio.Tuple, 0, len(hosts))
	for i, host := range hosts {
		items = append(items, cmdio.Tuple{
			Name: fmt.Sprintf("workspace-%02d", i+1),
			Id:   host,
		})
	}
	return items
}

// clusterItem mirrors libs/databrickscfg/cfgpickers/clusters.go's
// compatibleCluster: State, Access, and Runtime are exposed as methods so
// the Active/Inactive templates exercise text/template's method-resolution
// path, and State returns a pre-rendered colored string (matching the
// renderedState cache in production) so the demo also exercises ANSI codes
// emitted from inside a template.
type clusterItem struct {
	Name string
	Id   string

	access        string
	runtimeName   string
	renderedState string
}

func (c clusterItem) Access() string  { return c.access }
func (c clusterItem) Runtime() string { return c.runtimeName }
func (c clusterItem) State() string   { return c.renderedState }

func buildClusterItems(ctx context.Context) []clusterItem {
	green := func(s string) string { return cmdio.Green(ctx, s) }
	red := func(s string) string { return cmdio.Red(ctx, s) }
	blue := func(s string) string { return cmdio.Blue(ctx, s) }
	return []clusterItem{
		{Name: "shared-autoscaling-prod", Id: "0123-456789-abcdef01", access: "Shared", runtimeName: "DBR 14.3 LTS", renderedState: green("RUNNING")},
		{Name: "ml-gpu-experiments", Id: "0123-456789-abcdef02", access: "Assigned", runtimeName: "DBR 15.0 ML", renderedState: red("TERMINATED")},
		{Name: "job-compute-bronze-etl", Id: "0123-456789-abcdef03", access: "Shared", runtimeName: "DBR 13.3 LTS", renderedState: green("RUNNING")},
		{Name: "interactive-analytics", Id: "0123-456789-abcdef04", access: "Assigned", runtimeName: "DBR 14.3", renderedState: blue("PENDING")},
		{Name: "photon-streaming-realtime", Id: "0123-456789-abcdef05", access: "Shared", runtimeName: "DBR 14.3 Photon", renderedState: green("RUNNING")},
		{Name: "single-node-dev", Id: "0123-456789-abcdef06", access: "Assigned", runtimeName: "DBR 14.3 LTS", renderedState: red("TERMINATED")},
		{Name: "all-purpose-shared", Id: "0123-456789-abcdef07", access: "Shared", runtimeName: "DBR 15.0", renderedState: green("RUNNING")},
		{Name: "legacy-data-eng", Id: "0123-456789-abcdef08", access: "Assigned", runtimeName: "DBR 12.2 LTS", renderedState: red("TERMINATED")},
	}
}

// profileItem mirrors the profile picker in cmd/auth/token.go: regular items
// have a Host, the trailing meta items do not (so the {{if .Host}} branch fires).
type profileItem struct {
	Name string
	Host string
}

// buildProfileItems returns 6 profile-shaped items across AWS / Azure / GCP
// hosts plus the two trailing meta-rows ("Create a new profile", "Enter a
// host URL manually") used by the real profile picker.
func buildProfileItems() []profileItem {
	return []profileItem{
		{Name: "DEFAULT", Host: "https://acme.cloud.databricks.com"},
		{Name: "production", Host: "https://acme-prod.cloud.databricks.com"},
		{Name: "staging", Host: "https://acme-stg.cloud.databricks.com"},
		{Name: "field-eng", Host: "https://field-eng.cloud.databricks.com"},
		{Name: "azure-personal", Host: "https://adb-1234567890123456.78.azuredatabricks.net"},
		{Name: "gcp-sandbox", Host: "https://1234567890123456.7.gcp.databricks.com"},
		{Name: "Create a new profile"},
		{Name: "Enter a host URL manually"},
	}
}
