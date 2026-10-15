package lakeview

import (
	lakeview "github.com/databricks/cli/libs/sdktypes/lakeview/v1"
	"github.com/spf13/cobra"
)

func publishOverride(cmd *cobra.Command, req *lakeview.PublishDashboardRequest) {
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Always send embed_credentials, otherwise the API defaults to true.
		if req.EmbedCredentials == nil {
			req.EmbedCredentials = new(false)
		}
		return originalRunE(cmd, args)
	}
}

func init() {
	publishOverrides = append(publishOverrides, publishOverride)
}
