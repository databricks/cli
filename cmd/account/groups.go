// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package account

import (
	"github.com/spf13/cobra"
)

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
