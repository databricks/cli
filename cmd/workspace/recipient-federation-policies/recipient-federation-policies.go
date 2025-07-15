// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package recipient_federation_policies

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recipient-federation-policies",
		Short: `The Recipient Federation Policies APIs are only applicable in the open sharing model where the recipient object has the authentication type of OIDC_RECIPIENT, enabling data sharing from Databricks to non-Databricks recipients.`,
		Long: `The Recipient Federation Policies APIs are only applicable in the open sharing
  model where the recipient object has the authentication type of
  OIDC_RECIPIENT, enabling data sharing from Databricks to non-Databricks
  recipients. OIDC Token Federation enables secure, secret-less authentication
  for accessing Delta Sharing servers. Users and applications authenticate using
  short-lived OIDC tokens issued by their own Identity Provider (IdP), such as
  Azure Entra ID or Okta, without the need for managing static credentials or
  client secrets. A federation policy defines how non-Databricks recipients
  authenticate using OIDC tokens. It validates the OIDC claims in federated
  tokens and is set at the recipient level. The caller must be the owner of the
  recipient to create or manage a federation policy. Federation policies support
  the following scenarios: - User-to-Machine (U2M) flow: A user accesses Delta
  Shares using their own identity, such as connecting through PowerBI Delta
  Sharing Client. - Machine-to-Machine (M2M) flow: An application accesses Delta
  Shares using its own identity, typically for automation tasks like nightly
  jobs through Python Delta Sharing Client. OIDC Token Federation enables
  fine-grained access control, supports Multi-Factor Authentication (MFA), and
  enhances security by minimizing the risk of credential leakage through the use
  of short-lived, expiring tokens. It is designed for strong identity
  governance, secure cross-platform data sharing, and reduced operational
  overhead for credential management.
  
  For more information, see
  https://www.databricks.com/blog/announcing-oidc-token-federation-enhanced-delta-sharing-security
  and https://docs.databricks.com/en/delta-sharing/create-recipient-oidc-fed`,
		GroupID: "sharing",
		Annotations: map[string]string{
			"package": "sharing",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGetFederationPolicy())
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
	*sharing.CreateFederationPolicyRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sharing.CreateFederationPolicyRequest
	createReq.Policy = sharing.FederationPolicy{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Policy.Comment, "comment", createReq.Policy.Comment, `Description of the policy.`)
	cmd.Flags().StringVar(&createReq.Policy.Name, "name", createReq.Policy.Name, `Name of the federation policy.`)
	// TODO: complex arg: oidc_policy

	cmd.Use = "create RECIPIENT_NAME"
	cmd.Short = `Create recipient federation policy.`
	cmd.Long = `Create recipient federation policy.
  
  Create a federation policy for an OIDC_FEDERATION recipient for sharing data
  from Databricks to non-Databricks recipients. The caller must be the owner of
  the recipient. When sharing data from Databricks to non-Databricks clients,
  you can define a federation policy to authenticate non-Databricks recipients.
  The federation policy validates OIDC claims in federated tokens and is defined
  at the recipient level. This enables secretless sharing clients to
  authenticate using OIDC tokens.
  
  Supported scenarios for federation policies: 1. **User-to-Machine (U2M) flow**
  (e.g., PowerBI): A user accesses a resource using their own identity. 2.
  **Machine-to-Machine (M2M) flow** (e.g., OAuth App): An OAuth App accesses a
  resource using its own identity, typically for tasks like running nightly
  jobs.
  
  For an overview, refer to: - Blog post: Overview of feature:
  https://www.databricks.com/blog/announcing-oidc-token-federation-enhanced-delta-sharing-security
  
  For detailed configuration guides based on your use case: - Creating a
  Federation Policy as a provider:
  https://docs.databricks.com/en/delta-sharing/create-recipient-oidc-fed -
  Configuration and usage for Machine-to-Machine (M2M) applications (e.g.,
  Python Delta Sharing Client):
  https://docs.databricks.com/aws/en/delta-sharing/sharing-over-oidc-m2m -
  Configuration and usage for User-to-Machine (U2M) applications (e.g.,
  PowerBI):
  https://docs.databricks.com/aws/en/delta-sharing/sharing-over-oidc-u2m

  Arguments:
    RECIPIENT_NAME: Name of the recipient. This is the name of the recipient for which the
      policy is being created.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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
		createReq.RecipientName = args[0]

		response, err := w.RecipientFederationPolicies.Create(ctx, createReq)
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
	*sharing.DeleteFederationPolicyRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sharing.DeleteFederationPolicyRequest

	cmd.Use = "delete RECIPIENT_NAME NAME"
	cmd.Short = `Delete recipient federation policy.`
	cmd.Long = `Delete recipient federation policy.
  
  Deletes an existing federation policy for an OIDC_FEDERATION recipient. The
  caller must be the owner of the recipient.

  Arguments:
    RECIPIENT_NAME: Name of the recipient. This is the name of the recipient for which the
      policy is being deleted.
    NAME: Name of the policy. This is the name of the policy to be deleted.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.RecipientName = args[0]
		deleteReq.Name = args[1]

		err = w.RecipientFederationPolicies.Delete(ctx, deleteReq)
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

// start get-federation-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getFederationPolicyOverrides []func(
	*cobra.Command,
	*sharing.GetFederationPolicyRequest,
)

func newGetFederationPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var getFederationPolicyReq sharing.GetFederationPolicyRequest

	cmd.Use = "get-federation-policy RECIPIENT_NAME NAME"
	cmd.Short = `Get recipient federation policy.`
	cmd.Long = `Get recipient federation policy.
  
  Reads an existing federation policy for an OIDC_FEDERATION recipient for
  sharing data from Databricks to non-Databricks recipients. The caller must
  have read access to the recipient.

  Arguments:
    RECIPIENT_NAME: Name of the recipient. This is the name of the recipient for which the
      policy is being retrieved.
    NAME: Name of the policy. This is the name of the policy to be retrieved.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getFederationPolicyReq.RecipientName = args[0]
		getFederationPolicyReq.Name = args[1]

		response, err := w.RecipientFederationPolicies.GetFederationPolicy(ctx, getFederationPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getFederationPolicyOverrides {
		fn(cmd, &getFederationPolicyReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*sharing.ListFederationPoliciesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sharing.ListFederationPoliciesRequest

	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, ``)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, ``)

	cmd.Use = "list RECIPIENT_NAME"
	cmd.Short = `List recipient federation policies.`
	cmd.Long = `List recipient federation policies.
  
  Lists federation policies for an OIDC_FEDERATION recipient for sharing data
  from Databricks to non-Databricks recipients. The caller must have read access
  to the recipient.

  Arguments:
    RECIPIENT_NAME: Name of the recipient. This is the name of the recipient for which the
      policies are being listed.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.RecipientName = args[0]

		response := w.RecipientFederationPolicies.List(ctx, listReq)
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
	*sharing.UpdateFederationPolicyRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sharing.UpdateFederationPolicyRequest
	updateReq.Policy = sharing.FederationPolicy{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.UpdateMask, "update-mask", updateReq.UpdateMask, `The field mask specifies which fields of the policy to update.`)
	cmd.Flags().StringVar(&updateReq.Policy.Comment, "comment", updateReq.Policy.Comment, `Description of the policy.`)
	cmd.Flags().StringVar(&updateReq.Policy.Name, "name", updateReq.Policy.Name, `Name of the federation policy.`)
	// TODO: complex arg: oidc_policy

	cmd.Use = "update RECIPIENT_NAME NAME"
	cmd.Short = `Update recipient federation policy.`
	cmd.Long = `Update recipient federation policy.
  
  Updates an existing federation policy for an OIDC_RECIPIENT. The caller must
  be the owner of the recipient.

  Arguments:
    RECIPIENT_NAME: Name of the recipient. This is the name of the recipient for which the
      policy is being updated.
    NAME: Name of the policy. This is the name of the current name of the policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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
		updateReq.RecipientName = args[0]
		updateReq.Name = args[1]

		response, err := w.RecipientFederationPolicies.Update(ctx, updateReq)
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

// end service RecipientFederationPolicies
