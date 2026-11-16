// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"

	csp_enablement_account "github.com/databricks/cli/cmd/account/csp-enablement-account"
	disable_legacy_features "github.com/databricks/cli/cmd/account/disable-legacy-features"
	enable_ip_access_lists "github.com/databricks/cli/cmd/account/enable-ip-access-lists"
	esm_enablement_account "github.com/databricks/cli/cmd/account/esm-enablement-account"
	llm_proxy_partner_powered_account "github.com/databricks/cli/cmd/account/llm-proxy-partner-powered-account"
	llm_proxy_partner_powered_enforce "github.com/databricks/cli/cmd/account/llm-proxy-partner-powered-enforce"
	personal_compute "github.com/databricks/cli/cmd/account/personal-compute"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "settings",
	Short:   `Accounts Settings API allows users to manage settings at the account level.`,
	GroupID: "settings",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Accounts Settings API allows users to manage settings at the account level.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "GA",
		"launch_stage_display": "GA",
	}

	// Add subservices
	cmd.AddCommand(root.NewCommand(ctx, csp_enablement_account.Spec))
	cmd.AddCommand(root.NewCommand(ctx, disable_legacy_features.Spec))
	cmd.AddCommand(root.NewCommand(ctx, enable_ip_access_lists.Spec))
	cmd.AddCommand(root.NewCommand(ctx, esm_enablement_account.Spec))
	cmd.AddCommand(root.NewCommand(ctx, llm_proxy_partner_powered_account.Spec))
	cmd.AddCommand(root.NewCommand(ctx, llm_proxy_partner_powered_enforce.Spec))
	cmd.AddCommand(root.NewCommand(ctx, personal_compute.Spec))

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// end service AccountSettings
