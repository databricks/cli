package account

import "github.com/spf13/cobra"

// Groups returns an ordered list of command groups.
// The order matches the order used in the Databricks API explorer.
func Groups() []cobra.Group {
	return []cobra.Group{
		{
			ID:    "iam",
			Title: "Identity and Access Management",
		},
		{
			ID:    "catalog",
			Title: "Unity Catalog",
		},
		{
			ID:    "settings",
			Title: "Settings",
		},
		{
			ID:    "provisioning",
			Title: "Provisioning",
		},
		{
			ID:    "billing",
			Title: "Billing",
		},
		{
			ID:    "oauth2",
			Title: "OAuth",
		},
	}
}
