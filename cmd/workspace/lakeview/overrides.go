package lakeview

import (
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/spf13/cobra"
)

func publishOverride(cmd *cobra.Command, req *dashboards.PublishRequest) {
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Force send embed_credentials even when false, otherwise the API defaults to true.
		req.ForceSendFields = append(req.ForceSendFields, "EmbedCredentials")
		return originalRunE(cmd, args)
	}
}

func init() {
	publishOverrides = append(publishOverrides, publishOverride)
}
