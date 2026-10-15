// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package git_credentials

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	gitcredentials "github.com/databricks/cli/libs/sdktypes/gitcredentials/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git-credentials",
		Short: `Registers personal access token for Databricks to do operations on behalf of the user.`,
		Long: `Registers personal access token for Databricks to do operations on behalf of
  the user.

  See [more info].

  [more info]: https://docs.databricks.com/repos/get-access-tokens-from-git-provider.html`,
		GroupID: "workspace",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*gitcredentials.CreateCredentialsRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq gitcredentials.CreateCredentialsRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.GitEmail, "git-email", `The authenticating email associated with your Git provider user account.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.GitUsername, "git-username", `The username provided with your Git provider account and associated with the credential.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.IsDefaultForProvider, "is-default-for-provider", `if the credential is the default for the given provider.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Name, "name", `the name of the git credential, used for identification and ease of lookup.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.PersonalAccessToken, "personal-access-token", `The personal access token used to authenticate to the corresponding Git provider.`)
	flags.Int64PtrVar(cmd.Flags(), &createReq.PrincipalId, "principal-id", `The ID of the service principal whose credentials will be modified.`)

	cmd.Use = "create GIT_PROVIDER"
	cmd.Short = `Create a credential entry.`
	cmd.Long = `Create a credential entry.

  Creates a Git credential entry for the user. Use the PATCH endpoint to update
  existing credentials, or the DELETE endpoint to delete existing credentials.

  Arguments:
    GIT_PROVIDER: Git provider. This field is case-insensitive. The available Git providers
      are gitHub, bitbucketCloud, gitLab, azureDevOpsServices (Azure
      DevOps Services, including Microsoft Entra ID authentication),
      gitHubEnterprise, bitbucketServer (Bitbucket Data Center),
      gitLabEnterpriseEdition (GitLab Self-Managed), and awsCodeCommit.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'git_provider' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.GitProvider = new(args[0])
		}

		sdkReq, err := gitcredentials.CreateCredentialsRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.GitCredentials.CreateCredentials(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := gitcredentials.CreateCredentialsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*gitcredentials.DeleteCredentialsRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq gitcredentials.DeleteCredentialsRequest

	flags.Int64PtrVar(cmd.Flags(), &deleteReq.PrincipalId, "principal-id", `The ID of the service principal whose credentials will be modified.`)

	cmd.Use = "delete CREDENTIAL_ID"
	cmd.Short = `Delete a credential.`
	cmd.Long = `Delete a credential.

  Deletes the specified Git credential.

  Arguments:
    CREDENTIAL_ID: The ID for the corresponding credential to access.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		var deleteCredentialIdArg int64
		_, err = fmt.Sscan(args[0], &deleteCredentialIdArg)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}
		deleteReq.Id = &deleteCredentialIdArg

		sdkReq, err := gitcredentials.DeleteCredentialsRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := w.GitCredentials.DeleteCredentials(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*gitcredentials.GetCredentialsRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq gitcredentials.GetCredentialsRequest

	flags.Int64PtrVar(cmd.Flags(), &getReq.PrincipalId, "principal-id", `The ID of the service principal whose credentials will be modified.`)

	cmd.Use = "get CREDENTIAL_ID"
	cmd.Short = `Get a credential entry.`
	cmd.Long = `Get a credential entry.

  Gets the Git credential with the specified credential ID.

  Arguments:
    CREDENTIAL_ID: The ID for the corresponding credential to access.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		var getCredentialIdArg int64
		_, err = fmt.Sscan(args[0], &getCredentialIdArg)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}
		getReq.Id = &getCredentialIdArg

		sdkReq, err := gitcredentials.GetCredentialsRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.GitCredentials.GetCredentials(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := gitcredentials.GetCredentialsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*gitcredentials.ListCredentialsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq gitcredentials.ListCredentialsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.Int64PtrVar(cmd.Flags(), &listReq.PrincipalId, "principal-id", `The ID of the service principal whose credentials will be listed.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list"
	cmd.Short = `Get Git credentials.`
	cmd.Long = `Get Git credentials.

  Lists the calling user's Git credentials.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := gitcredentials.ListCredentialsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		response, err := w.GitCredentials.ListCredentials(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		out, err := gitcredentials.ListCredentialsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*gitcredentials.UpdateCredentialsRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq gitcredentials.UpdateCredentialsRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.GitEmail, "git-email", `The authenticating email associated with your Git provider user account.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.GitUsername, "git-username", `The username provided with your Git provider account and associated with the credential.`)
	flags.BoolPtrVar(cmd.Flags(), &updateReq.IsDefaultForProvider, "is-default-for-provider", `if the credential is the default for the given provider.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Name, "name", `the name of the git credential, used for identification and ease of lookup.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.PersonalAccessToken, "personal-access-token", `The personal access token used to authenticate to the corresponding Git provider.`)
	flags.Int64PtrVar(cmd.Flags(), &updateReq.PrincipalId, "principal-id", `The ID of the service principal whose credentials will be modified.`)

	cmd.Use = "update CREDENTIAL_ID GIT_PROVIDER"
	cmd.Short = `Update a credential.`
	cmd.Long = `Update a credential.

  Updates the specified Git credential.

  Arguments:
    CREDENTIAL_ID: The ID for the corresponding credential to access.
    GIT_PROVIDER: Git provider. This field is case-insensitive. The available Git providers
      are gitHub, bitbucketCloud, gitLab, azureDevOpsServices (Azure
      DevOps Services, including Microsoft Entra ID authentication),
      gitHubEnterprise, bitbucketServer (Bitbucket Data Center),
      gitLabEnterpriseEdition (GitLab Self-Managed), and awsCodeCommit.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only CREDENTIAL_ID as positional arguments. Provide 'git_provider' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		var updateCredentialIdArg int64
		_, err = fmt.Sscan(args[0], &updateCredentialIdArg)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}
		updateReq.Id = &updateCredentialIdArg
		if !cmd.Flags().Changed("json") {
			updateReq.GitProvider = new(args[1])
		}

		sdkReq, err := gitcredentials.UpdateCredentialsRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.GitCredentials.UpdateCredentials(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
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

// end service GitCredentials
