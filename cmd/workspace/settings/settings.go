// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"

	aibi_dashboard_embedding_access_policy "github.com/databricks/cli/cmd/workspace/aibi-dashboard-embedding-access-policy"
	aibi_dashboard_embedding_approved_domains "github.com/databricks/cli/cmd/workspace/aibi-dashboard-embedding-approved-domains"
	automatic_cluster_update "github.com/databricks/cli/cmd/workspace/automatic-cluster-update"
	compliance_security_profile "github.com/databricks/cli/cmd/workspace/compliance-security-profile"
	dashboard_email_subscriptions "github.com/databricks/cli/cmd/workspace/dashboard-email-subscriptions"
	default_namespace "github.com/databricks/cli/cmd/workspace/default-namespace"
	default_warehouse_id "github.com/databricks/cli/cmd/workspace/default-warehouse-id"
	disable_legacy_access "github.com/databricks/cli/cmd/workspace/disable-legacy-access"
	disable_legacy_dbfs "github.com/databricks/cli/cmd/workspace/disable-legacy-dbfs"
	enable_export_notebook "github.com/databricks/cli/cmd/workspace/enable-export-notebook"
	enable_notebook_table_clipboard "github.com/databricks/cli/cmd/workspace/enable-notebook-table-clipboard"
	enable_results_downloading "github.com/databricks/cli/cmd/workspace/enable-results-downloading"
	enhanced_security_monitoring "github.com/databricks/cli/cmd/workspace/enhanced-security-monitoring"
	llm_proxy_partner_powered_workspace "github.com/databricks/cli/cmd/workspace/llm-proxy-partner-powered-workspace"
	restrict_workspace_admins "github.com/databricks/cli/cmd/workspace/restrict-workspace-admins"
	sql_results_download "github.com/databricks/cli/cmd/workspace/sql-results-download"
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
		RunE: root.ReportUnknownSubcommand,
	}

	// Add subservices
	cmd.AddCommand(aibi_dashboard_embedding_access_policy.New())
	cmd.AddCommand(aibi_dashboard_embedding_approved_domains.New())
	cmd.AddCommand(automatic_cluster_update.New())
	cmd.AddCommand(compliance_security_profile.New())
	cmd.AddCommand(dashboard_email_subscriptions.New())
	cmd.AddCommand(default_namespace.New())
	cmd.AddCommand(default_warehouse_id.New())
	cmd.AddCommand(disable_legacy_access.New())
	cmd.AddCommand(disable_legacy_dbfs.New())
	cmd.AddCommand(enable_export_notebook.New())
	cmd.AddCommand(enable_notebook_table_clipboard.New())
	cmd.AddCommand(enable_results_downloading.New())
	cmd.AddCommand(enhanced_security_monitoring.New())
	cmd.AddCommand(llm_proxy_partner_powered_workspace.New())
	cmd.AddCommand(restrict_workspace_admins.New())
	cmd.AddCommand(sql_results_download.New())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// end service Settings
