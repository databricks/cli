// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package federation_policy

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		Use:   "federation-policy",
		Short: `These APIs manage account federation policies.`,
		Long: `These APIs manage account federation policies.
  
  Account federation policies allow users and service principals in your
  Databricks account to securely access Databricks APIs using tokens from your
  trusted identity providers (IdPs).
  
  With token federation, your users and service principals can exchange tokens
  from your IdP for Databricks OAuth tokens, which can be used to access
  Databricks APIs. Token federation eliminates the need to manage Databricks
  secrets, and allows you to centralize management of token issuance policies in
  your IdP. Databricks token federation is typically used in combination with
  [SCIM], so users in your IdP are synchronized into your Databricks account.
  
  Token federation is configured in your Databricks account using an account
  federation policy. An account federation policy specifies: * which IdP, or
  issuer, your Databricks account should accept tokens from * how to determine
  which Databricks user, or subject, a token is issued for
  
  To configure a federation policy, you provide the following: * The required
  token __issuer__, as specified in the “iss” claim of your tokens. The
  issuer is an https URL that identifies your IdP. * The allowed token
  __audiences__, as specified in the “aud” claim of your tokens. This
  identifier is intended to represent the recipient of the token. As long as the
  audience in the token matches at least one audience in the policy, the token
  is considered a match. If unspecified, the default value is your Databricks
  account id. * The __subject claim__, which indicates which token claim
  contains the Databricks username of the user the token was issued for. If
  unspecified, the default value is “sub”. * Optionally, the public keys
  used to validate the signature of your tokens, in JWKS format. If unspecified
  (recommended), Databricks automatically fetches the public keys from your
  issuer’s well known endpoint. Databricks strongly recommends relying on your
  issuer’s well known endpoint for discovering public keys.
  
  An example federation policy is:  issuer: "https://idp.mycompany.com/oidc"
  audiences: ["databricks"] subject_claim: "sub" 
  
  An example JWT token body that matches this policy and could be used to
  authenticate to Databricks as user username@mycompany.com is:  { "iss":
  "https://idp.mycompany.com/oidc", "aud": "databricks", "sub":
  "username@mycompany.com" } 
  
  You may also need to configure your IdP to generate tokens for your users to
  exchange with Databricks, if your users do not already have the ability to
  generate tokens that are compatible with your federation policy.
  
  You do not need to configure an OAuth application in Databricks to use token
  federation.
  
  [SCIM]: https://docs.databricks.com/admin/users-groups/scim/index.html`,
		GroupID: "oauth2",
		Annotations: map[string]string{
			"package": "oauth2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

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
	*oauth2.CreateAccountFederationPolicyRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq oauth2.CreateAccountFederationPolicyRequest
	createReq.Policy = oauth2.FederationPolicy{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.PolicyId, "policy-id", createReq.PolicyId, `The identifier for the federation policy.`)
	cmd.Flags().StringVar(&createReq.Policy.Description, "description", createReq.Policy.Description, `Description of the federation policy.`)
	cmd.Flags().StringVar(&createReq.Policy.Name, "name", createReq.Policy.Name, `Resource name for the federation policy.`)
	// TODO: complex arg: oidc_policy

	cmd.Use = "create"
	cmd.Short = `Create account federation policy.`
	cmd.Long = `Create account federation policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.Policy)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}

		response, err := a.FederationPolicy.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*oauth2.DeleteAccountFederationPolicyRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq oauth2.DeleteAccountFederationPolicyRequest

	cmd.Use = "delete POLICY_ID"
	cmd.Short = `Delete account federation policy.`
	cmd.Long = `Delete account federation policy.

  Arguments:
    POLICY_ID: The identifier for the federation policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deleteReq.PolicyId = args[0]

		err = a.FederationPolicy.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*oauth2.GetAccountFederationPolicyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq oauth2.GetAccountFederationPolicyRequest

	cmd.Use = "get POLICY_ID"
	cmd.Short = `Get account federation policy.`
	cmd.Long = `Get account federation policy.

  Arguments:
    POLICY_ID: The identifier for the federation policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getReq.PolicyId = args[0]

		response, err := a.FederationPolicy.Get(ctx, getReq)
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
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*oauth2.ListAccountFederationPoliciesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq oauth2.ListAccountFederationPoliciesRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, ``)

	cmd.Use = "list"
	cmd.Short = `List account federation policies.`
	cmd.Long = `List account federation policies.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response := a.FederationPolicy.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*oauth2.UpdateAccountFederationPolicyRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq oauth2.UpdateAccountFederationPolicyRequest
	updateReq.Policy = oauth2.FederationPolicy{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.UpdateMask, "update-mask", updateReq.UpdateMask, `The field mask specifies which fields of the policy to update.`)
	cmd.Flags().StringVar(&updateReq.Policy.Description, "description", updateReq.Policy.Description, `Description of the federation policy.`)
	cmd.Flags().StringVar(&updateReq.Policy.Name, "name", updateReq.Policy.Name, `Resource name for the federation policy.`)
	// TODO: complex arg: oidc_policy

	cmd.Use = "update POLICY_ID"
	cmd.Short = `Update account federation policy.`
	cmd.Long = `Update account federation policy.

  Arguments:
    POLICY_ID: The identifier for the federation policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq.Policy)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.PolicyId = args[0]

		response, err := a.FederationPolicy.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service AccountFederationPolicy
