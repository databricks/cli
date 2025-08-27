// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package service_principal_federation_policy

import (
	"fmt"

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
		Use:   "service-principal-federation-policy",
		Short: `These APIs manage service principal federation policies.`,
		Long: `These APIs manage service principal federation policies.
  
  Service principal federation, also known as Workload Identity Federation,
  allows your automated workloads running outside of Databricks to securely
  access Databricks APIs without the need for Databricks secrets. With Workload
  Identity Federation, your application (or workload) authenticates to
  Databricks as a Databricks service principal, using tokens provided by the
  workload runtime.
  
  Databricks strongly recommends using Workload Identity Federation to
  authenticate to Databricks from automated workloads, over alternatives such as
  OAuth client secrets or Personal Access Tokens, whenever possible. Workload
  Identity Federation is supported by many popular services, including Github
  Actions, Azure DevOps, GitLab, Terraform Cloud, and Kubernetes clusters, among
  others.
  
  Workload identity federation is configured in your Databricks account using a
  service principal federation policy. A service principal federation policy
  specifies: * which IdP, or issuer, the service principal is allowed to
  authenticate from * which workload identity, or subject, is allowed to
  authenticate as the Databricks service principal
  
  To configure a federation policy, you provide the following: * The required
  token __issuer__, as specified in the “iss” claim of workload identity
  tokens. The issuer is an https URL that identifies the workload identity
  provider. * The required token __subject__, as specified in the “sub”
  claim of workload identity tokens. The subject uniquely identifies the
  workload in the workload runtime environment. * The allowed token
  __audiences__, as specified in the “aud” claim of workload identity
  tokens. The audience is intended to represent the recipient of the token. As
  long as the audience in the token matches at least one audience in the policy,
  the token is considered a match. If unspecified, the default value is your
  Databricks account id. * Optionally, the public keys used to validate the
  signature of the workload identity tokens, in JWKS format. If unspecified
  (recommended), Databricks automatically fetches the public keys from the
  issuer’s well known endpoint. Databricks strongly recommends relying on the
  issuer’s well known endpoint for discovering public keys.
  
  An example service principal federation policy, for a Github Actions workload,
  is:  issuer: "https://token.actions.githubusercontent.com" audiences:
  ["https://github.com/my-github-org"] subject:
  "repo:my-github-org/my-repo:environment:prod" 
  
  An example JWT token body that matches this policy and could be used to
  authenticate to Databricks is:  { "iss":
  "https://token.actions.githubusercontent.com", "aud":
  "https://github.com/my-github-org", "sub":
  "repo:my-github-org/my-repo:environment:prod" } 
  
  You may also need to configure the workload runtime to generate tokens for
  your workloads.
  
  You do not need to configure an OAuth application in Databricks to use token
  federation.`,
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
	*oauth2.CreateServicePrincipalFederationPolicyRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq oauth2.CreateServicePrincipalFederationPolicyRequest
	createReq.Policy = oauth2.FederationPolicy{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.PolicyId, "policy-id", createReq.PolicyId, `The identifier for the federation policy.`)
	cmd.Flags().StringVar(&createReq.Policy.Description, "description", createReq.Policy.Description, `Description of the federation policy.`)
	cmd.Flags().StringVar(&createReq.Policy.Name, "name", createReq.Policy.Name, `Resource name for the federation policy.`)
	// TODO: complex arg: oidc_policy

	cmd.Use = "create SERVICE_PRINCIPAL_ID"
	cmd.Short = `Create service principal federation policy.`
	cmd.Long = `Create service principal federation policy.
  
  Create account federation policy.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal id for the federation policy.`

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
		_, err = fmt.Sscan(args[0], &createReq.ServicePrincipalId)
		if err != nil {
			return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
		}

		response, err := a.ServicePrincipalFederationPolicy.Create(ctx, createReq)
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
	*oauth2.DeleteServicePrincipalFederationPolicyRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq oauth2.DeleteServicePrincipalFederationPolicyRequest

	cmd.Use = "delete SERVICE_PRINCIPAL_ID POLICY_ID"
	cmd.Short = `Delete service principal federation policy.`
	cmd.Long = `Delete service principal federation policy.
  
  Delete account federation policy.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal id for the federation policy.
    POLICY_ID: The identifier for the federation policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteReq.ServicePrincipalId)
		if err != nil {
			return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
		}
		deleteReq.PolicyId = args[1]

		err = a.ServicePrincipalFederationPolicy.Delete(ctx, deleteReq)
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
	*oauth2.GetServicePrincipalFederationPolicyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq oauth2.GetServicePrincipalFederationPolicyRequest

	cmd.Use = "get SERVICE_PRINCIPAL_ID POLICY_ID"
	cmd.Short = `Get service principal federation policy.`
	cmd.Long = `Get service principal federation policy.
  
  Get account federation policy.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal id for the federation policy.
    POLICY_ID: The identifier for the federation policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &getReq.ServicePrincipalId)
		if err != nil {
			return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
		}
		getReq.PolicyId = args[1]

		response, err := a.ServicePrincipalFederationPolicy.Get(ctx, getReq)
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
	*oauth2.ListServicePrincipalFederationPoliciesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq oauth2.ListServicePrincipalFederationPoliciesRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, ``)

	cmd.Use = "list SERVICE_PRINCIPAL_ID"
	cmd.Short = `List service principal federation policies.`
	cmd.Long = `List service principal federation policies.
  
  List account federation policies.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal id for the federation policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		_, err = fmt.Sscan(args[0], &listReq.ServicePrincipalId)
		if err != nil {
			return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
		}

		response := a.ServicePrincipalFederationPolicy.List(ctx, listReq)
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
	*oauth2.UpdateServicePrincipalFederationPolicyRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq oauth2.UpdateServicePrincipalFederationPolicyRequest
	updateReq.Policy = oauth2.FederationPolicy{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.UpdateMask, "update-mask", updateReq.UpdateMask, `The field mask specifies which fields of the policy to update.`)
	cmd.Flags().StringVar(&updateReq.Policy.Description, "description", updateReq.Policy.Description, `Description of the federation policy.`)
	cmd.Flags().StringVar(&updateReq.Policy.Name, "name", updateReq.Policy.Name, `Resource name for the federation policy.`)
	// TODO: complex arg: oidc_policy

	cmd.Use = "update SERVICE_PRINCIPAL_ID POLICY_ID"
	cmd.Short = `Update service principal federation policy.`
	cmd.Long = `Update service principal federation policy.
  
  Update account federation policy.

  Arguments:
    SERVICE_PRINCIPAL_ID: The service principal id for the federation policy.
    POLICY_ID: The identifier for the federation policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
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
		_, err = fmt.Sscan(args[0], &updateReq.ServicePrincipalId)
		if err != nil {
			return fmt.Errorf("invalid SERVICE_PRINCIPAL_ID: %s", args[0])
		}
		updateReq.PolicyId = args[1]

		response, err := a.ServicePrincipalFederationPolicy.Update(ctx, updateReq)
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

// end service ServicePrincipalFederationPolicy
