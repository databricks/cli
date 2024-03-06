// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"github.com/spf13/cobra"

	automatic_cluster_update "github.com/databricks/cli/cmd/workspace/automatic-cluster-update"
	csp_enablement "github.com/databricks/cli/cmd/workspace/csp-enablement"
	default_namespace "github.com/databricks/cli/cmd/workspace/default-namespace"
	esm_enablement "github.com/databricks/cli/cmd/workspace/esm-enablement"
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
	cmd.AddCommand(automatic_cluster_update.New())
	cmd.AddCommand(csp_enablement.New())
	cmd.AddCommand(default_namespace.New())
	cmd.AddCommand(esm_enablement.New())
	cmd.AddCommand(restrict_workspace_admins.New())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// end service Settings
