package tui

import (
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

// clusterItem mirrors the shape used by libs/databrickscfg/cfgpickers/clusters.go,
// where Active/Inactive templates reference Name, State, Runtime, and Id.
type clusterItem struct {
	Name    string
	State   string
	Runtime string
	Id      string
}

func buildClusterItems() []clusterItem {
	return []clusterItem{
		{Name: "shared-autoscaling-prod", State: "RUNNING", Runtime: "DBR 14.3 LTS", Id: "0123-456789-abcdef01"},
		{Name: "ml-gpu-experiments", State: "TERMINATED", Runtime: "DBR 15.0 ML", Id: "0123-456789-abcdef02"},
		{Name: "job-compute-bronze-etl", State: "RUNNING", Runtime: "DBR 13.3 LTS", Id: "0123-456789-abcdef03"},
		{Name: "interactive-analytics", State: "PENDING", Runtime: "DBR 14.3", Id: "0123-456789-abcdef04"},
		{Name: "photon-streaming-realtime", State: "RUNNING", Runtime: "DBR 14.3 Photon", Id: "0123-456789-abcdef05"},
		{Name: "single-node-dev", State: "TERMINATED", Runtime: "DBR 14.3 LTS", Id: "0123-456789-abcdef06"},
		{Name: "all-purpose-shared", State: "RUNNING", Runtime: "DBR 15.0", Id: "0123-456789-abcdef07"},
		{Name: "legacy-data-eng", State: "TERMINATED", Runtime: "DBR 12.2 LTS", Id: "0123-456789-abcdef08"},
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
