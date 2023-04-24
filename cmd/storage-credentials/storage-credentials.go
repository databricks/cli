// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package storage_credentials

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "storage-credentials",
	Short: `A storage credential represents an authentication and authorization mechanism for accessing data stored on your cloud tenant.`,
	Long: `A storage credential represents an authentication and authorization mechanism
  for accessing data stored on your cloud tenant. Each storage credential is
  subject to Unity Catalog access-control policies that control which users and
  groups can access the credential. If a user does not have access to a storage
  credential in Unity Catalog, the request fails and Unity Catalog does not
  attempt to authenticate to your cloud tenant on the user’s behalf.
  
  Databricks recommends using external locations rather than using storage
  credentials directly.
  
  To create storage credentials, you must be a Databricks account admin. The
  account admin who creates the storage credential can delegate ownership to
  another user or group to manage permissions on it.`,
}

// start create command

var createReq catalog.CreateStorageCredential
var createJson jsonflag.JsonFlag

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]
		createReq.MetastoreId = args[1]

		response, err := w.StorageCredentials.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteStorageCredentialRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if there are dependent external locations or external tables.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete a credential.`,
	Long: `Delete a credential.
  
  Deletes a storage credential from the metastore. The caller must be an owner
  of the storage credential.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.StorageCredentials.StorageCredentialInfoNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Name of the storage credential")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have name of the storage credential")
		}
		deleteReq.Name = args[0]

		err = w.StorageCredentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetStorageCredentialRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get a credential.`,
	Long: `Get a credential.
  
  Gets a storage credential from the metastore. The caller must be a metastore
  admin, the owner of the storage credential, or have some permission on the
  storage credential.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.StorageCredentials.StorageCredentialInfoNameToIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Name of the storage credential")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have name of the storage credential")
		}
		getReq.Name = args[0]

		response, err := w.StorageCredentials.Get(ctx, getReq)
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
	Short: `List credentials.`,
	Long: `List credentials.
  
  Gets an array of storage credentials (as __StorageCredentialInfo__ objects).
  The array is limited to only those storage credentials the caller has
  permission to access. If the caller is a metastore admin, all storage
  credentials will be retrieved. There is no guarantee of a specific ordering of
  the elements in the array.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.StorageCredentials.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.UpdateStorageCredential
var updateJson jsonflag.JsonFlag

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
	Use:   "update",
	Short: `Update a credential.`,
	Long: `Update a credential.
  
  Updates a storage credential on the metastore. The caller must be the owner of
  the storage credential or a metastore admin. If the caller is a metastore
  admin, only the __owner__ credential can be changed.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.Name = args[0]

		response, err := w.StorageCredentials.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start validate command

var validateReq catalog.ValidateStorageCredential
var validateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(validateCmd)
	// TODO: short flags
	validateCmd.Flags().Var(&validateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_iam_role
	// TODO: complex arg: azure_service_principal
	validateCmd.Flags().StringVar(&validateReq.ExternalLocationName, "external-location-name", validateReq.ExternalLocationName, `The name of an existing external location to validate.`)
	// TODO: complex arg: gcp_service_account_key
	validateCmd.Flags().BoolVar(&validateReq.ReadOnly, "read-only", validateReq.ReadOnly, `Whether the storage credential is only usable for read operations.`)
	// TODO: any: storage_credential_name
	validateCmd.Flags().StringVar(&validateReq.Url, "url", validateReq.Url, `The external location url to validate.`)

}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: `Validate a storage credential.`,
	Long: `Validate a storage credential.
  
  Validates a storage credential. At least one of __external_location_name__ and
  __url__ need to be provided. If only one of them is provided, it will be used
  for validation. And if both are provided, the __url__ will be used for
  validation, and __external_location_name__ will be ignored when checking
  overlapping urls.
  
  Either the __storage_credential_name__ or the cloud-specific credential must
  be provided.
  
  The caller must be a metastore admin or the storage credential owner or have
  the **CREATE_EXTERNAL_LOCATION** privilege on the metastore and the storage
  credential.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = validateJson.Unmarshall(&validateReq)
		if err != nil {
			return err
		}

		response, err := w.StorageCredentials.Validate(ctx, validateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service StorageCredentials
