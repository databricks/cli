// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"github.com/spf13/cobra"
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

		// This service is being previewed; hide from help output.
		Hidden: true,
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// end service Settings
