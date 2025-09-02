package workspace

import (
	"github.com/spf13/cobra"
)

// Groups returns an ordered list of command groups.
// The order matches the order used in the Databricks API explorer.
func Groups() []cobra.Group {
	return []cobra.Group{
		{
			ID:    "workspace",
			Title: "Databricks Workspace",
		},
		{
			ID:    "compute",
			Title: "Compute",
		},
		{
			ID:    "jobs",
			Title: "Workflows",
		},
		{
			ID:    "pipelines",
			Title: "Delta Live Tables",
		},
		{
			ID:    "ml",
			Title: "Machine Learning",
		},
		{
			ID:    "serving",
			Title: "Real-time Serving",
		},
		{
			ID:    "iam",
			Title: "Identity and Access Management",
		},
		{
			ID:    "sql",
			Title: "Databricks SQL",
		},
		{
			ID:    "catalog",
			Title: "Unity Catalog",
		},
		{
			ID:    "sharing",
			Title: "Delta Sharing",
		},
		{
			ID:    "settings",
			Title: "Settings",
		},
		{
			ID:    "development",
			Title: "Developer Tools",
		},
		{
			ID:    "vectorsearch",
			Title: "Vector Search",
		},
		{
			ID:    "dashboards",
			Title: "Dashboards",
		},
		{
			ID:    "marketplace",
			Title: "Marketplace",
		},
		{
			ID:    "apps",
			Title: "Apps",
		},
		{
			ID:    "cleanrooms",
			Title: "Clean Rooms",
		},
		{
			ID:    "aibuilder",
			Title: "AI Builder",
		},
		{
			ID:    "database",
			Title: "Database",
		},
		{
			ID:    "qualitymonitorv2",
			Title: "Quality Monitor v2",
		},
		{
			ID:    "oauth2",
			Title: "OAuth",
		},
	}
}
