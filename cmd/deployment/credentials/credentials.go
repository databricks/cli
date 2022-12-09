package credentials

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/deployment"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "credentials",
	Short: `These APIs manage credential configurations for this workspace.`,
	Long: `These APIs manage credential configurations for this workspace. Databricks
  needs access to a cross-account service IAM role in your AWS account so that
  Databricks can deploy clusters in the appropriate VPC for the new workspace. A
  credential configuration encapsulates this role information, and its ID is
  used when creating a new workspace.`,
}

var createReq deployment.CreateCredentialRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: complex arg: aws_credentials
	createCmd.Flags().StringVar(&createReq.CredentialsName, "credentials-name", "", `The human-readable name of the credential configuration object.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create credential configuration.`,
	Long: `Create credential configuration.
  
  Creates a Databricks credential configuration that represents cloud
  cross-account credentials for a specified account. Databricks uses this to set
  up network infrastructure properly to host Databricks clusters. For your AWS
  IAM role, you need to trust the External ID (the Databricks Account API
  account ID) in the returned credential object, and configure the required
  access policy.
  
  Save the response's credentials_id field, which is the ID for your new
  credential configuration object.
  
  For information about how to create a new workspace with this API, see [Create
  a new workspace using the Account API]
  
  [Create a new workspace using the Account API]: http://docs.databricks.com/administration-guide/account-api/new-workspace.html`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Credentials.Create(ctx, createReq)
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

var deleteReq deployment.DeleteCredentialRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.CredentialsId, "credentials-id", "", `Databricks Account API credential configuration ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete credential configuration.`,
	Long: `Delete credential configuration.
  
  Deletes a Databricks credential configuration object for an account, both
  specified by ID. You cannot delete a credential that is associated with any
  workspace.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.Credentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq deployment.GetCredentialRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.CredentialsId, "credentials-id", "", `Databricks Account API credential configuration ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get credential configuration.`,
	Long: `Get credential configuration.
  
  Gets a Databricks credential configuration object for an account, both
  specified by ID.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Credentials.Get(ctx, getReq)
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
	Short: `Get all credential configurations.`,
	Long: `Get all credential configurations.
  
  Gets all Databricks credential configurations associated with an account
  specified by ID.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Credentials.List(ctx)
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

// end service Credentials
