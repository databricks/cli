// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package storage

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "storage",
	Short: `These APIs manage storage configurations for this workspace.`,
	Long: `These APIs manage storage configurations for this workspace. A root storage S3
  bucket in your account is required to store objects like cluster logs,
  notebook revisions, and job results. You can also use the root storage S3
  bucket for storage of non-production DBFS data. A storage configuration
  encapsulates this bucket information, and its ID is used when creating a new
  workspace.`,
}

// start create command

var createReq provisioning.CreateStorageConfigurationRequest
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.StorageConfigurationName = args[0]
		_, err = fmt.Sscan(args[1], &createReq.RootBucketInfo)
		if err != nil {
			return fmt.Errorf("invalid ROOT_BUCKET_INFO: %s", args[1])
		}

		response, err := a.Storage.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq provisioning.DeleteStorageRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete STORAGE_CONFIGURATION_ID",
	Short: `Delete storage configuration.`,
	Long: `Delete storage configuration.
  
  Deletes a Databricks storage configuration. You cannot delete a storage
  configuration that is associated with any workspace.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Storage.StorageConfigurationStorageConfigurationNameToStorageConfigurationIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Databricks Account API storage configuration ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks account api storage configuration id")
		}
		deleteReq.StorageConfigurationId = args[0]

		err = a.Storage.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq provisioning.GetStorageRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get STORAGE_CONFIGURATION_ID",
	Short: `Get storage configuration.`,
	Long: `Get storage configuration.
  
  Gets a Databricks storage configuration for an account, both specified by ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Storage.StorageConfigurationStorageConfigurationNameToStorageConfigurationIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Databricks Account API storage configuration ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks account api storage configuration id")
		}
		getReq.StorageConfigurationId = args[0]

		response, err := a.Storage.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all storage configurations.`,
	Long: `Get all storage configurations.
  
  Gets a list of all Databricks storage configurations for your account,
  specified by ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.Storage.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service Storage
