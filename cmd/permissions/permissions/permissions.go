package permissions

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/permissions"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "permissions",
	Short: `Permissions API are used to create read, write, edit, update and manage access for various users on different objects and endpoints.`,
}

var getReq permissions.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.RequestObjectId, "request-object-id", "", ``)
	getCmd.Flags().StringVar(&getReq.RequestObjectType, "request-object-type", "", `<needs content>.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get object permissions.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Permissions.Get(ctx, getReq)
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

var getPermissionLevelsReq permissions.GetPermissionLevels

func init() {
	Cmd.AddCommand(getPermissionLevelsCmd)
	// TODO: short flags

	getPermissionLevelsCmd.Flags().StringVar(&getPermissionLevelsReq.RequestObjectId, "request-object-id", "", `<needs content>.`)
	getPermissionLevelsCmd.Flags().StringVar(&getPermissionLevelsReq.RequestObjectType, "request-object-type", "", `<needs content>.`)

}

var getPermissionLevelsCmd = &cobra.Command{
	Use:   "get-permission-levels",
	Short: `Get permission levels.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Permissions.GetPermissionLevels(ctx, getPermissionLevelsReq)
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

var setReq permissions.PermissionsRequest

func init() {
	Cmd.AddCommand(setCmd)
	// TODO: short flags

	// TODO: array: access_control_list
	setCmd.Flags().StringVar(&setReq.RequestObjectId, "request-object-id", "", ``)
	setCmd.Flags().StringVar(&setReq.RequestObjectType, "request-object-type", "", `<needs content>.`)

}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: `Set permissions.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Permissions.Set(ctx, setReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateReq permissions.PermissionsRequest

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	// TODO: array: access_control_list
	updateCmd.Flags().StringVar(&updateReq.RequestObjectId, "request-object-id", "", ``)
	updateCmd.Flags().StringVar(&updateReq.RequestObjectType, "request-object-type", "", `<needs content>.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update permission.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Permissions.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
