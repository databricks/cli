package private_access

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "private-access",
	Short: `These APIs manage private access settings for this account.`,
	Long: `These APIs manage private access settings for this account. A private access
  settings object specifies how your workspace is accessed using AWS
  PrivateLink. Each workspace that has any PrivateLink connections must include
  the ID for a private access settings object is in its workspace configuration
  object. Your account must be enabled for PrivateLink to use these APIs. Before
  configuring PrivateLink, it is important to read the [Databricks article about
  PrivateLink].
  
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,
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
	Long: `Create private access settings.
  
  Creates a private access settings object, which specifies how your workspace
  is accessed over [AWS PrivateLink]. To use AWS PrivateLink, a workspace must
  have a private access settings object referenced by ID in the workspace's
  private_access_settings_id property.
  
  You can share one private access settings with multiple workspaces in a single
  account. However, private access settings are specific to AWS regions, so only
  workspaces in the same AWS region can use a given private access settings
  object.
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
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
	Long: `Delete a private access settings object.
  
  Deletes a private access settings object, which determines how your workspace
  is accessed over [AWS PrivateLink].
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
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
	Long: `Get a private access settings object.
  
  Gets a private access settings object, which specifies how your workspace is
  accessed over [AWS PrivateLink].
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
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
	Long: `Get all private access settings objects.
  
  Gets a list of all private access settings objects for an account, specified
  by ID.
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for AWS PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
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
	Long: `Replace private access settings.
  
  Updates an existing private access settings object, which specifies how your
  workspace is accessed over [AWS PrivateLink]. To use AWS PrivateLink, a
  workspace must have a private access settings object referenced by ID in the
  workspace's private_access_settings_id property.
  
  This operation completely overwrites your existing private access settings
  object attached to your workspaces. All workspaces attached to the private
  access settings are affected by any change. If public_access_enabled,
  private_access_level, or allowed_vpc_endpoint_ids are updated, effects of
  these changes might take several minutes to propagate to the workspace API.
  You can share one private access settings object with multiple workspaces in a
  single account. However, private access settings are specific to AWS regions,
  so only workspaces in the same AWS region can use a given private access
  settings object.
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  This operation is available only if your account is on the E2 version of the
  platform and your Databricks account is enabled for PrivateLink (Public
  Preview). Contact your Databricks representative to enable your account for
  PrivateLink.
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.PrivateAccess.Replace(ctx, replaceReq)
		if err != nil {
			return err
		}

		return nil
	},
}
