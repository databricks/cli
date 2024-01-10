// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package lakeview

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lakeview",
		Short: `These APIs provide specific management operations for Lakeview dashboards.`,
		Long: `These APIs provide specific management operations for Lakeview dashboards.
  Generic resource management can be done with Workspace API (import, export,
  get-status, list, delete).`,
		GroupID: "dashboards",
		Annotations: map[string]string{
			"package": "dashboards",
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

// start publish command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var publishOverrides []func(
	*cobra.Command,
	*dashboards.PublishRequest,
)

func newPublish() *cobra.Command {
	cmd := &cobra.Command{}

	var publishReq dashboards.PublishRequest
	var publishJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&publishJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&publishReq.EmbedCredentials, "embed-credentials", publishReq.EmbedCredentials, `Flag to indicate if the publisher's credentials should be embedded in the published dashboard.`)
	cmd.Flags().StringVar(&publishReq.WarehouseId, "warehouse-id", publishReq.WarehouseId, `The ID of the warehouse that can be used to override the warehouse which was set in the draft.`)

	cmd.Use = "publish DASHBOARD_ID"
	cmd.Short = `Publish dashboard.`
	cmd.Long = `Publish dashboard.
  
  Publish the current draft dashboard.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = publishJson.Unmarshal(&publishReq)
			if err != nil {
				return err
			}
		}
		publishReq.DashboardId = args[0]

		err = w.Lakeview.Publish(ctx, publishReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range publishOverrides {
		fn(cmd, &publishReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newPublish())
	})
}

// end service Lakeview
