// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"

	csp_enablement_account "github.com/databricks/cli/cmd/account/csp-enablement-account"
	disable_legacy_features "github.com/databricks/cli/cmd/account/disable-legacy-features"
	enable_ip_access_lists "github.com/databricks/cli/cmd/account/enable-ip-access-lists"
	esm_enablement_account "github.com/databricks/cli/cmd/account/esm-enablement-account"
	personal_compute "github.com/databricks/cli/cmd/account/personal-compute"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Short:   `Accounts Settings API allows users to manage settings at the account level.`,
		Long:    `Accounts Settings API allows users to manage settings at the account level.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// Check if the subcommand exists
				for _, subcmd := range cmd.Commands() {
					if subcmd.Name() == args[0] {
						// Let Cobra handle the valid subcommand
						return nil
					}
				}
				// Return error for unknown subcommands
				return &root.InvalidArgsError{
					Message: fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath()),
					Command: cmd,
				}
			}
			return cmd.Help()
		},
	}

	// Add subservices
	cmd.AddCommand(csp_enablement_account.New())
	cmd.AddCommand(disable_legacy_features.New())
	cmd.AddCommand(enable_ip_access_lists.New())
	cmd.AddCommand(esm_enablement_account.New())
	cmd.AddCommand(personal_compute.New())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// end service AccountSettings
