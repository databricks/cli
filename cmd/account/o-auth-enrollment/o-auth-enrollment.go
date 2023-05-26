// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package o_auth_enrollment

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "o-auth-enrollment",
	Short: `These APIs enable administrators to enroll OAuth for their accounts, which is required for adding/using any OAuth published/custom application integration.`,
	Long: `These APIs enable administrators to enroll OAuth for their accounts, which is
  required for adding/using any OAuth published/custom application integration.
  
  **Note:** Your account must be on the E2 version to use these APIs, this is
  because OAuth is only supported on the E2 version.`,
}

// start create command

var createReq oauth2.CreateOAuthEnrollment
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().BoolVar(&createReq.EnableAllPublishedApps, "enable-all-published-apps", createReq.EnableAllPublishedApps, `If true, enable OAuth for all the published applications in the account.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create OAuth Enrollment request.`,
	Long: `Create OAuth Enrollment request.
  
  Create an OAuth Enrollment request to enroll OAuth for this account and
  optionally enable the OAuth integration for all the partner applications in
  the account.
  
  The parter applications are: - Power BI - Tableau Desktop - Databricks CLI
  
  The enrollment is executed asynchronously, so the API will return 204
  immediately. The actual enrollment take a few minutes, you can check the
  status via API :method:OAuthEnrollment/get.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
		}

		err = a.OAuthEnrollment.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

func init() {
	Cmd.AddCommand(getCmd)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get OAuth enrollment status.`,
	Long: `Get OAuth enrollment status.
  
  Gets the OAuth enrollment status for this Account.
  
  You can only add/use the OAuth published/custom application integrations when
  OAuth enrollment status is enabled.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.OAuthEnrollment.Get(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service OAuthEnrollment
