package shares

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "shares",
	Short: `Databricks Delta Sharing: Shares REST API.`,
}

var createReq unitycatalog.CreateShare

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", "", `[Create: OPT] comment when creating the share.`)
	createCmd.Flags().Int64Var(&createReq.CreatedAt, "created-at", 0, `[Create:IGN] Time at which this Share was created, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.CreatedBy, "created-by", "", `[Create:IGN] Username of Share creator.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `[Create:REQ] Name of the Share.`)
	// TODO: array: objects

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a share.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Shares.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq unitycatalog.DeleteShareRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", "", `The name of the share.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a share.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Shares.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq unitycatalog.GetShareRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().BoolVar(&getReq.IncludeSharedData, "include-shared-data", false, `Query for data to include in the share.`)
	getCmd.Flags().StringVar(&getReq.Name, "name", "", `The name of the share.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a share.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Shares.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List shares.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Shares.ListAll(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var sharePermissionsReq unitycatalog.SharePermissionsRequest

func init() {
	Cmd.AddCommand(sharePermissionsCmd)
	// TODO: short flags

	sharePermissionsCmd.Flags().StringVar(&sharePermissionsReq.Name, "name", "", `Required.`)

}

var sharePermissionsCmd = &cobra.Command{
	Use:   "share-permissions",
	Short: `Get permissions.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Shares.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var updateReq unitycatalog.UpdateShare

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Name, "name", "", `The name of the share.`)
	// TODO: array: updates

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a share.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Shares.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updatePermissionsReq unitycatalog.UpdateSharePermissions

func init() {
	Cmd.AddCommand(updatePermissionsCmd)
	// TODO: short flags

	// TODO: array: changes
	updatePermissionsCmd.Flags().StringVar(&updatePermissionsReq.Name, "name", "", `Required.`)

}

var updatePermissionsCmd = &cobra.Command{
	Use:   "update-permissions",
	Short: `Update permissions.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Shares.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}

		return nil
	},
}
