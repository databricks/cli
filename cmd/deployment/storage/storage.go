package storage

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "storage",
	Short: `These APIs manage storage configurations for this workspace.`,
}

var createReq deployment.CreateStorageConfigurationRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: complex arg: root_bucket_info
	createCmd.Flags().StringVar(&createReq.StorageConfigurationName, "storage-configuration-name", "", `The human-readable name of the storage configuration.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create new storage configuration.`,
	Long: `Create new storage configuration.
  
  Creates new storage configuration for an account, specified by ID. Uploads a
  storage configuration object that represents the root AWS S3 bucket in your
  account. Databricks stores related workspace assets including DBFS, cluster
  logs, and job results. For the AWS S3 bucket, you need to configure the
  required bucket policy.
  
  For information about how to create a new workspace with this API, see [Create
  a new workspace using the Account API]
  
  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Storage.Create(ctx, createReq)
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

var deleteReq deployment.DeleteStorageRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.StorageConfigurationId, "storage-configuration-id", "", `Databricks Account API storage configuration ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete storage configuration.`,
	Long: `Delete storage configuration.
  
  Deletes a Databricks storage configuration. You cannot delete a storage
  configuration that is associated with any workspace.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.Storage.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq deployment.GetStorageRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.StorageConfigurationId, "storage-configuration-id", "", `Databricks Account API storage configuration ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get storage configuration.`,
	Long: `Get storage configuration.
  
  Gets a Databricks storage configuration for an account, both specified by ID.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Storage.Get(ctx, getReq)
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
	Short: `Get all storage configurations.`,
	Long: `Get all storage configurations.
  
  Gets a list of all Databricks storage configurations for your account,
  specified by ID.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Storage.List(ctx)
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
