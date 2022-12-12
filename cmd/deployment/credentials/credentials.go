// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package credentials

import (
	"fmt"

	"github.com/databricks/bricks/lib/jsonflag"
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

// start create command

var createReq deployment.CreateCredentialRequest
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.CredentialsName = args[0]
		_, err = fmt.Sscan(args[1], &createReq.AwsCredentials)
		if err != nil {
			return fmt.Errorf("invalid AWS_CREDENTIALS: %s", args[1])
		}

		response, err := a.Credentials.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq deployment.DeleteCredentialRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete CREDENTIALS_ID",
	Short: `Delete credential configuration.`,
	Long: `Delete credential configuration.
  
  Deletes a Databricks credential configuration object for an account, both
  specified by ID. You cannot delete a credential that is associated with any
  workspace.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Credentials.CredentialCredentialsNameToCredentialsIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Databricks Account API credential configuration ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks account api credential configuration id")
		}
		deleteReq.CredentialsId = args[0]

		err = a.Credentials.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq deployment.GetCredentialRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get CREDENTIALS_ID",
	Short: `Get credential configuration.`,
	Long: `Get credential configuration.
  
  Gets a Databricks credential configuration object for an account, both
  specified by ID.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Credentials.CredentialCredentialsNameToCredentialsIdMap(ctx)
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Databricks Account API credential configuration ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks account api credential configuration id")
		}
		getReq.CredentialsId = args[0]

		response, err := a.Credentials.Get(ctx, getReq)
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
	Short: `Get all credential configurations.`,
	Long: `Get all credential configurations.
  
  Gets all Databricks credential configurations associated with an account
  specified by ID.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Credentials.List(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service Credentials
