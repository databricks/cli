package storage_credentials

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "storage-credentials",
	Short: `A storage credential represents an authentication and authorization mechanism for accessing data stored on your cloud tenant, using an IAM role.`,
	Long: `A storage credential represents an authentication and authorization mechanism
  for accessing data stored on your cloud tenant, using an IAM role. Each
  storage credential is subject to Unity Catalog access-control policies that
  control which users and groups can access the credential. If a user does not
  have access to a storage credential in Unity Catalog, the request fails and
  Unity Catalog does not attempt to authenticate to your cloud tenant on the
  user’s behalf.
  
  Databricks recommends using external locations rather than using storage
  credentials directly.
  
  To create storage credentials, you must be a Databricks account admin. The
  account admin who creates the storage credential can delegate ownership to
  another user or group to manage permissions on it.`,
}

// start create command

var createReq unitycatalog.CreateStorageCredential
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_service_principal
	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Comment associated with the credential.`)
	// TODO: complex arg: gcp_service_account_key
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `The credential name.`)
	createCmd.Flags().BoolVar(&createReq.SkipValidation, "skip-validation", createReq.SkipValidation, `Optional.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create credentials.`,
	Long: `Create credentials.
  
  Creates a new storage credential. The request object is specific to the cloud:
  
  * **AwsIamRole** for AWS credentials * **AzureServicePrincipal** for Azure
  credentials * **GcpServiceAcountKey** for GCP credentials.
  
  The caller must be a Metastore admin and have the CREATE STORAGE CREDENTIAL
  privilege on the Metastore.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.StorageCredentials.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq unitycatalog.DeleteStorageCredentialRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if there are dependent external locations or external tables.`)
	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", deleteReq.Name, `Required.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a credential.`,
	Long: `Delete a credential.
  
  Deletes a storage credential from the Metastore. The caller must be an owner
  of the storage credential.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.StorageCredentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq unitycatalog.GetStorageCredentialRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Name, "name", getReq.Name, `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a credential.`,
	Long: `Get a credential.
  
  Gets a storage credential from the Metastore. The caller must be a Metastore
  admin, the owner of the storage credential, or have a level of privilege on
  the storage credential.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.StorageCredentials.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List credentials.`,
	Long: `List credentials.
  
  Gets an array of storage credentials (as StorageCredentialInfo objects). The
  array is limited to only those storage credentials the caller has the
  privilege level to access. If the caller is a Metastore admin, all storage
  credentials will be retrieved.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.StorageCredentials.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq unitycatalog.UpdateStorageCredential
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_service_principal
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Comment associated with the credential.`)
	// TODO: complex arg: gcp_service_account_key
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The credential name.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of credential.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a credential.`,
	Long: `Update a credential.
  
  Updates a storage credential on the Metastore. The caller must be the owner of
  the storage credential. If the caller is a Metastore admin, only the __owner__
  credential can be changed.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.StorageCredentials.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service StorageCredentials
