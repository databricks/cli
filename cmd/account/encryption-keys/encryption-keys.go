// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package encryption_keys

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
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
  platform. If you are not sure, contact your Databricks representative.`,
	Annotations: map[string]string{
		"package": "provisioning",
	},
}

// start create command

var createReq provisioning.CreateCustomerManagedKeyRequest
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_key_info
	// TODO: complex arg: gcp_key_info

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
  types, subscription types, and AWS regions that currently support creation of
  Databricks workspaces.
  
  This operation is available only if your account is on the E2 version of the
  platform or on a select custom plan that allows multiple workspaces per
  account.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := a.EncryptionKeys.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start delete command

var deleteReq provisioning.DeleteEncryptionKeyRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete CUSTOMER_MANAGED_KEY_ID",
	Short: `Delete encryption key configuration.`,
	Long: `Delete encryption key configuration.
  
  Deletes a customer-managed key configuration object for an account. You cannot
  delete a configuration that is associated with a running workspace.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		}
		deleteReq.CustomerManagedKeyId = args[0]

		err = a.EncryptionKeys.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command

var getReq provisioning.GetEncryptionKeyRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get CUSTOMER_MANAGED_KEY_ID",
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
  platform.",`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		}
		getReq.CustomerManagedKeyId = args[0]

		response, err := a.EncryptionKeys.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command

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

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.EncryptionKeys.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service EncryptionKeys
