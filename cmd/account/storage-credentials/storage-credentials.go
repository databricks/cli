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
	Use:   "create",
	Short: `Create a storage credential.`,
	Long: `Create a storage credential.
  
  Creates a new storage credential. The request object is specific to the cloud:
  
  * **AwsIamRole** for AWS credentials * **AzureServicePrincipal** for Azure
  credentials * **GcpServiceAcountKey** for GCP credentials.
  
  The caller must be a metastore admin and have the
  **CREATE_STORAGE_CREDENTIAL** privilege on the metastore.`,

	Annotations: map[string]string{},
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

// end service AccountStorageCredentials
