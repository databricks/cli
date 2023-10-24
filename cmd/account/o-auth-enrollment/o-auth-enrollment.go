// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package o_auth_enrollment

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "o-auth-enrollment",
		Short: `These APIs enable administrators to enroll OAuth for their accounts, which is required for adding/using any OAuth published/custom application integration.`,
		Long: `These APIs enable administrators to enroll OAuth for their accounts, which is
  required for adding/using any OAuth published/custom application integration.
  
  **Note:** Your account must be on the E2 version to use these APIs, this is
  because OAuth is only supported on the E2 version.`,
		GroupID: "oauth2",
		Annotations: map[string]string{
			"package": "oauth2",
		},
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*oauth2.CreateOAuthEnrollment,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq oauth2.CreateOAuthEnrollment
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createReq.EnableAllPublishedApps, "enable-all-published-apps", createReq.EnableAllPublishedApps, `If true, enable OAuth for all the published applications in the account.`)

	cmd.Use = "create"
	cmd.Short = `Create OAuth Enrollment request.`
	cmd.Long = `Create OAuth Enrollment request.
  
  Create an OAuth Enrollment request to enroll OAuth for this account and
  optionally enable the OAuth integration for all the partner applications in
  the account.
  
  The parter applications are: - Power BI - Tableau Desktop - Databricks CLI
  
  The enrollment is executed asynchronously, so the API will return 204
  immediately. The actual enrollment take a few minutes, you can check the
  status via API :method:OAuthEnrollment/get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		}

		err = a.OAuthEnrollment.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreate())
	})
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get"
	cmd.Short = `Get OAuth enrollment status.`
	cmd.Long = `Get OAuth enrollment status.
  
  Gets the OAuth enrollment status for this Account.
  
  You can only add/use the OAuth published/custom application integrations when
  OAuth enrollment status is enabled.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.OAuthEnrollment.Get(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGet())
	})
}

// end service OAuthEnrollment
