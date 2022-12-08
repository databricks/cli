package private_access

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "private-access",
	Short: `These APIs manage private access settings for this account.`,
}

var createReq deployment.UpsertPrivateAccessSettingsRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: array: allowed_vpc_endpoint_ids
	createCmd.Flags().Var(&createReq.PrivateAccessLevel, "private-access-level", `The private access level controls which VPC endpoints can connect to the UI or API of any workspace that attaches this private access settings object.`)
	createCmd.Flags().StringVar(&createReq.PrivateAccessSettingsId, "private-access-settings-id", "", `Databricks Account API private access settings ID.`)
	createCmd.Flags().StringVar(&createReq.PrivateAccessSettingsName, "private-access-settings-name", "", `The human-readable name of the private access settings object.`)
	createCmd.Flags().BoolVar(&createReq.PublicAccessEnabled, "public-access-enabled", false, `Determines if the workspace can be accessed over public internet.`)
	createCmd.Flags().StringVar(&createReq.Region, "region", "", `The AWS region for workspaces associated with this private access settings object.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create private access settings.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.PrivateAccess.Create(ctx, createReq)
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

var deleteReq deployment.DeletePrivateAccesRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.PrivateAccessSettingsId, "private-access-settings-id", "", `Databricks Account API private access settings ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a private access settings object.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.PrivateAccess.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq deployment.GetPrivateAccesRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.PrivateAccessSettingsId, "private-access-settings-id", "", `Databricks Account API private access settings ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a private access settings object.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.PrivateAccess.Get(ctx, getReq)
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
	Short: `Get all private access settings objects.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.PrivateAccess.List(ctx)
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

var replaceReq deployment.UpsertPrivateAccessSettingsRequest

func init() {
	Cmd.AddCommand(replaceCmd)
	// TODO: short flags

	// TODO: array: allowed_vpc_endpoint_ids
	replaceCmd.Flags().Var(&replaceReq.PrivateAccessLevel, "private-access-level", `The private access level controls which VPC endpoints can connect to the UI or API of any workspace that attaches this private access settings object.`)
	replaceCmd.Flags().StringVar(&replaceReq.PrivateAccessSettingsId, "private-access-settings-id", "", `Databricks Account API private access settings ID.`)
	replaceCmd.Flags().StringVar(&replaceReq.PrivateAccessSettingsName, "private-access-settings-name", "", `The human-readable name of the private access settings object.`)
	replaceCmd.Flags().BoolVar(&replaceReq.PublicAccessEnabled, "public-access-enabled", false, `Determines if the workspace can be accessed over public internet.`)
	replaceCmd.Flags().StringVar(&replaceReq.Region, "region", "", `The AWS region for workspaces associated with this private access settings object.`)

}

var replaceCmd = &cobra.Command{
	Use:   "replace",
	Short: `Replace private access settings.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.PrivateAccess.Replace(ctx, replaceReq)
		if err != nil {
			return err
		}

		return nil
	},
}
