package encryption_keys

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "encryption-keys",
	Short: `These APIs manage encryption key configurations for this workspace (optional).`,
	Long: `These APIs manage encryption key configurations for this workspace (optional).
  A key configuration encapsulates the AWS KMS key information and some
  information about how the key configuration can be used. There are two
  possible uses for key configurations:
  
  * Managed services: A key configuration can be used to encrypt a workspace's
  notebook and secret data in the control plane, as well as Databricks SQL
  queries and query history. * Storage: A key configuration can be used to
  encrypt a workspace's DBFS and EBS data in the data plane.
  
  In both of these cases, the key configuration's ID is used when creating a new
  workspace. This Preview feature is available if your account is on the E2
  version of the platform. Updating a running workspace with workspace storage
  encryption requires that the workspace is on the E2 version of the platform.
  If you have an older workspace, it might not be on the E2 version of the
  platform. If you are not sure, contact your Databricks reprsentative.`,
}

var createReq deployment.CreateCustomerManagedKeyRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: complex arg: aws_key_info
	// TODO: array: use_cases

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create encryption key configuration.`,
	Long: `Create encryption key configuration.
  
  Creates a customer-managed key configuration object for an account, specified
  by ID. This operation uploads a reference to a customer-managed key to
  Databricks. If the key is assigned as a workspace's customer-managed key for
  managed services, Databricks uses the key to encrypt the workspaces notebooks
  and secrets in the control plane, in addition to Databricks SQL queries and
  query history. If it is specified as a workspace's customer-managed key for
  workspace storage, the key encrypts the workspace's root S3 bucket (which
  contains the workspace's root DBFS and system data) and, optionally, cluster
  EBS volume data.
  
  **Important**: Customer-managed keys are supported only for some deployment
  types, subscription types, and AWS regions.
  
  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.EncryptionKeys.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteReq deployment.DeleteEncryptionKeyRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.CustomerManagedKeyId, "customer-managed-key-id", deleteReq.CustomerManagedKeyId, `Databricks encryption key configuration ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete encryption key configuration.`,
	Long: `Delete encryption key configuration.
  
  Deletes a customer-managed key configuration object for an account. You cannot
  delete a configuration that is associated with a running workspace.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.EncryptionKeys.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var getReq deployment.GetEncryptionKeyRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.CustomerManagedKeyId, "customer-managed-key-id", getReq.CustomerManagedKeyId, `Databricks encryption key configuration ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get encryption key configuration.`,
	Long: `Get encryption key configuration.
  
  Gets a customer-managed key configuration object for an account, specified by
  ID. This operation uploads a reference to a customer-managed key to
  Databricks. If assigned as a workspace's customer-managed key for managed
  services, Databricks uses the key to encrypt the workspaces notebooks and
  secrets in the control plane, in addition to Databricks SQL queries and query
  history. If it is specified as a workspace's customer-managed key for storage,
  the key encrypts the workspace's root S3 bucket (which contains the
  workspace's root DBFS and system data) and, optionally, cluster EBS volume
  data.
  
  **Important**: Customer-managed keys are supported only for some deployment
  types, subscription types, and AWS regions.
  
  This operation is available only if your account is on the E2 version of the
  platform.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.EncryptionKeys.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all encryption key configurations.`,
	Long: `Get all encryption key configurations.
  
  Gets all customer-managed key configuration objects for an account. If the key
  is specified as a workspace's managed services customer-managed key,
  Databricks uses the key to encrypt the workspace's notebooks and secrets in
  the control plane, in addition to Databricks SQL queries and query history. If
  the key is specified as a workspace's storage customer-managed key, the key is
  used to encrypt the workspace's root S3 bucket and optionally can encrypt
  cluster EBS volumes data in the data plane.
  
  **Important**: Customer-managed keys are supported only for some deployment
  types, subscription types, and AWS regions.
  
  This operation is available only if your account is on the E2 version of the
  platform.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.EncryptionKeys.List(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service EncryptionKeys
