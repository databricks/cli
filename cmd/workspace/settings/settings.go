// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"context"

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

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "settings",
	Short:   `Workspace Settings API allows users to manage settings at the workspace level.`,
	GroupID: "settings",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Workspace Settings API allows users to manage settings at the workspace level.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "GA",
		"launch_stage_display": "GA",
	}

	// Add subservices
	cmd.AddCommand(root.NewCommand(ctx, aibi_dashboard_embedding_access_policy.Spec))
	cmd.AddCommand(root.NewCommand(ctx, aibi_dashboard_embedding_approved_domains.Spec))
	cmd.AddCommand(root.NewCommand(ctx, automatic_cluster_update.Spec))
	cmd.AddCommand(root.NewCommand(ctx, compliance_security_profile.Spec))
	cmd.AddCommand(root.NewCommand(ctx, dashboard_email_subscriptions.Spec))
	cmd.AddCommand(root.NewCommand(ctx, default_namespace.Spec))
	cmd.AddCommand(root.NewCommand(ctx, default_warehouse_id.Spec))
	cmd.AddCommand(root.NewCommand(ctx, disable_legacy_access.Spec))
	cmd.AddCommand(root.NewCommand(ctx, disable_legacy_dbfs.Spec))
	cmd.AddCommand(root.NewCommand(ctx, enable_export_notebook.Spec))
	cmd.AddCommand(root.NewCommand(ctx, enable_notebook_table_clipboard.Spec))
	cmd.AddCommand(root.NewCommand(ctx, enable_results_downloading.Spec))
	cmd.AddCommand(root.NewCommand(ctx, enhanced_security_monitoring.Spec))
	cmd.AddCommand(root.NewCommand(ctx, llm_proxy_partner_powered_workspace.Spec))
	cmd.AddCommand(root.NewCommand(ctx, restrict_workspace_admins.Spec))
	cmd.AddCommand(root.NewCommand(ctx, sql_results_download.Spec))

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// end service Settings
