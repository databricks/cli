// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"github.com/spf13/cobra"

	csp_enablement_account "github.com/databricks/cli/cmd/account/csp-enablement-account"
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
	}

	// Add subservices
	cmd.AddCommand(csp_enablement_account.New())
	cmd.AddCommand(esm_enablement_account.New())
	cmd.AddCommand(personal_compute.New())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// end service AccountSettings
