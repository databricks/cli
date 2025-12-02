package lakeview

import (
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/spf13/cobra"
)

func publishOverride(cmd *cobra.Command, req *dashboards.PublishRequest) {
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Always send embed_credentials to the API, even when false.
		// This ensures the false value is explicitly sent rather than omitted,
		// which would cause the API to default to true.
		req.ForceSendFields = append(req.ForceSendFields, "EmbedCredentials")
		return originalRunE(cmd, args)
	}
}

func init() {
	publishOverrides = append(publishOverrides, publishOverride)
}
