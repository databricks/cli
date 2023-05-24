// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package storage_credentials

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "storage-credentials",
	Short: `These APIs manage storage credentials for a particular metastore.`,
	Long:  `These APIs manage storage credentials for a particular metastore.`,
}

// start create command

var createReq catalog.CreateStorageCredential
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_service_principal
	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Comment associated with the credential.`)
	// TODO: complex arg: gcp_service_account_key
	createCmd.Flags().BoolVar(&createReq.ReadOnly, "read-only", createReq.ReadOnly, `Whether the storage credential is only usable for read operations.`)
	createCmd.Flags().BoolVar(&createReq.SkipValidation, "skip-validation", createReq.SkipValidation, `Supplying true to this argument skips validation of the created credential.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME METASTORE_ID",
	Short: `Create a storage credential.`,
	Long: `Create a storage credential.
  
  Creates a new storage credential. The request object is specific to the cloud:
  
  * **AwsIamRole** for AWS credentials * **AzureServicePrincipal** for Azure
  credentials * **GcpServiceAcountKey** for GCP credentials.
  
  The caller must be a metastore admin and have the
  **CREATE_STORAGE_CREDENTIAL** privilege on the metastore.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = createJson.Unmarshal(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]
		createReq.MetastoreId = args[1]

		response, err := a.StorageCredentials.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteAccountStorageCredentialRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete METASTORE_ID NAME",
	Short: `Delete a storage credential.`,
	Long: `Delete a storage credential.
  
  Deletes a storage credential from the metastore. The caller must be an owner
  of the storage credential.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		deleteReq.MetastoreId = args[0]
		deleteReq.Name = args[1]

		err = a.StorageCredentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetAccountStorageCredentialRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get METASTORE_ID NAME",
	Short: `Gets the named storage credential.`,
	Long: `Gets the named storage credential.
  
  Gets a storage credential from the metastore. The caller must be a metastore
  admin, the owner of the storage credential, or have a level of privilege on
  the storage credential.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		getReq.MetastoreId = args[0]
		getReq.Name = args[1]

		response, err := a.StorageCredentials.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq catalog.ListAccountStorageCredentialsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

}

var listCmd = &cobra.Command{
	Use:   "list METASTORE_ID",
	Short: `Get all storage credentials assigned to a metastore.`,
	Long: `Get all storage credentials assigned to a metastore.
  
  Gets a list of all storage credentials that have been assigned to given
  metastore.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		listReq.MetastoreId = args[0]

		response, err := a.StorageCredentials.List(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.UpdateStorageCredential
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_service_principal
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Comment associated with the credential.`)
	updateCmd.Flags().BoolVar(&updateReq.Force, "force", updateReq.Force, `Force update even if there are dependent external locations or external tables.`)
	// TODO: complex arg: gcp_service_account_key
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The credential name.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of credential.`)
	updateCmd.Flags().BoolVar(&updateReq.ReadOnly, "read-only", updateReq.ReadOnly, `Whether the storage credential is only usable for read operations.`)
	updateCmd.Flags().BoolVar(&updateReq.SkipValidation, "skip-validation", updateReq.SkipValidation, `Supplying true to this argument skips validation of the updated credential.`)

}

var updateCmd = &cobra.Command{
	Use:   "update METASTORE_ID NAME",
	Short: `Updates a storage credential.`,
	Long: `Updates a storage credential.
  
  Updates a storage credential on the metastore. The caller must be the owner of
  the storage credential. If the caller is a metastore admin, only the __owner__
  credential can be changed.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		err = updateJson.Unmarshal(&updateReq)
		if err != nil {
			return err
		}
		updateReq.MetastoreId = args[0]
		updateReq.Name = args[1]

		response, err := a.StorageCredentials.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service AccountStorageCredentials
