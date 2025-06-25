// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package redash_config

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "redash-config",
		Short:   `Redash V2 service for workspace configurations (internal).`,
		Long:    `Redash V2 service for workspace configurations (internal)`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGetConfig())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getConfigOverrides []func(
	*cobra.Command,
)

func newGetConfig() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-config"
	cmd.Short = `Read workspace configuration for Redash-v2.`
	cmd.Long = `Read workspace configuration for Redash-v2.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.RedashConfig.GetConfig(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getConfigOverrides {
		fn(cmd)
	}

	return cmd
}

// end service RedashConfig
