package workspaces

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspaces",
	Short: `These APIs manage workspaces for this account.`,
}

var createReq deployment.CreateWorkspaceRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.AwsRegion, "aws-region", "", `The AWS region of the workspace's data plane.`)
	createCmd.Flags().StringVar(&createReq.Cloud, "cloud", "", `The cloud provider which the workspace uses.`)
	// TODO: complex arg: cloud_resource_bucket
	createCmd.Flags().StringVar(&createReq.CredentialsId, "credentials-id", "", `ID of the workspace's credential configuration object.`)
	createCmd.Flags().StringVar(&createReq.DeploymentName, "deployment-name", "", `The deployment name defines part of the subdomain for the workspace.`)
	createCmd.Flags().StringVar(&createReq.Location, "location", "", `The Google Cloud region of the workspace data plane in your Google account.`)
	createCmd.Flags().StringVar(&createReq.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", "", `The ID of the workspace's managed services encryption key configuration object.`)
	// TODO: complex arg: network
	createCmd.Flags().StringVar(&createReq.NetworkId, "network-id", "", `The ID of the workspace's network configuration object.`)
	// TODO: complex arg: pricing_tier
	createCmd.Flags().StringVar(&createReq.PrivateAccessSettingsId, "private-access-settings-id", "", `ID of the workspace's private access settings object.`)
	createCmd.Flags().StringVar(&createReq.StorageConfigurationId, "storage-configuration-id", "", `The ID of the workspace's storage configuration object.`)
	createCmd.Flags().StringVar(&createReq.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", "", `The ID of the workspace's storage encryption key configuration object.`)
	createCmd.Flags().StringVar(&createReq.WorkspaceName, "workspace-name", "", `The workspace's human-readable name.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new workspace.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.Workspaces.Create(ctx, createReq)
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

var deleteReq deployment.DeleteWorkspaceRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().Int64Var(&deleteReq.WorkspaceId, "workspace-id", 0, `Workspace ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete workspace.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.Workspaces.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq deployment.GetWorkspaceRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().Int64Var(&getReq.WorkspaceId, "workspace-id", 0, `Workspace ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get workspace.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.Workspaces.Get(ctx, getReq)
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

var getWorkspaceKeyHistoryReq deployment.GetWorkspaceKeyHistoryRequest

func init() {
	Cmd.AddCommand(getWorkspaceKeyHistoryCmd)
	// TODO: short flags

	getWorkspaceKeyHistoryCmd.Flags().Int64Var(&getWorkspaceKeyHistoryReq.WorkspaceId, "workspace-id", 0, `Workspace ID.`)

}

var getWorkspaceKeyHistoryCmd = &cobra.Command{
	Use:   "get-workspace-key-history",
	Short: `Get the history of a workspace's associations with keys.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.Workspaces.GetWorkspaceKeyHistory(ctx, getWorkspaceKeyHistoryReq)
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
	Short: `Get all workspaces.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.Workspaces.List(ctx)
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

var updateReq deployment.UpdateWorkspaceRequest

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.AwsRegion, "aws-region", "", `The AWS region of the workspace's data plane (for example, us-west-2).`)
	updateCmd.Flags().StringVar(&updateReq.CredentialsId, "credentials-id", "", `ID of the workspace's credential configuration object.`)
	updateCmd.Flags().StringVar(&updateReq.ManagedServicesCustomerManagedKeyId, "managed-services-customer-managed-key-id", "", `The ID of the workspace's managed services encryption key configuration object.`)
	updateCmd.Flags().StringVar(&updateReq.NetworkId, "network-id", "", `The ID of the workspace's network configuration object.`)
	updateCmd.Flags().StringVar(&updateReq.StorageConfigurationId, "storage-configuration-id", "", `The ID of the workspace's storage configuration object.`)
	updateCmd.Flags().StringVar(&updateReq.StorageCustomerManagedKeyId, "storage-customer-managed-key-id", "", `The ID of the key configuration object for workspace storage.`)
	updateCmd.Flags().Int64Var(&updateReq.WorkspaceId, "workspace-id", 0, `Workspace ID.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update workspace configuration.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.Workspaces.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
