// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"github.com/spf13/cobra"

	aibi_dashboard_embedding_access_policy "github.com/databricks/cli/cmd/workspace/aibi-dashboard-embedding-access-policy"
	aibi_dashboard_embedding_approved_domains "github.com/databricks/cli/cmd/workspace/aibi-dashboard-embedding-approved-domains"
	automatic_cluster_update "github.com/databricks/cli/cmd/workspace/automatic-cluster-update"
	compliance_security_profile "github.com/databricks/cli/cmd/workspace/compliance-security-profile"
	default_namespace "github.com/databricks/cli/cmd/workspace/default-namespace"
	disable_legacy_access "github.com/databricks/cli/cmd/workspace/disable-legacy-access"
	disable_legacy_dbfs "github.com/databricks/cli/cmd/workspace/disable-legacy-dbfs"
	enhanced_security_monitoring "github.com/databricks/cli/cmd/workspace/enhanced-security-monitoring"
	restrict_workspace_admins "github.com/databricks/cli/cmd/workspace/restrict-workspace-admins"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Short:   `Workspace Settings API allows users to manage settings at the workspace level.`,
		Long:    `Workspace Settings API allows users to manage settings at the workspace level.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
	}

	// Add subservices
	cmd.AddCommand(aibi_dashboard_embedding_access_policy.New())
	cmd.AddCommand(aibi_dashboard_embedding_approved_domains.New())
	cmd.AddCommand(automatic_cluster_update.New())
	cmd.AddCommand(compliance_security_profile.New())
	cmd.AddCommand(default_namespace.New())
	cmd.AddCommand(disable_legacy_access.New())
	cmd.AddCommand(disable_legacy_dbfs.New())
	cmd.AddCommand(enhanced_security_monitoring.New())
	cmd.AddCommand(restrict_workspace_admins.New())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// end service Settings
