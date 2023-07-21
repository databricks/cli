// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package git_credentials

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "workspace",
		},
	}

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
	*workspace.CreateCredentials,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq workspace.CreateCredentials
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.GitUsername, "git-username", createReq.GitUsername, `Git username.`)
	cmd.Flags().StringVar(&createReq.PersonalAccessToken, "personal-access-token", createReq.PersonalAccessToken, `The personal access token used to authenticate to the corresponding Git provider.`)

	cmd.Use = "create GIT_PROVIDER"
	cmd.Short = `Create a credential entry.`
	cmd.Long = `Create a credential entry.
  
  Creates a Git credential entry for the user. Only one Git credential per user
  is supported, so any attempts to create credentials if an entry already exists
  will fail. Use the PATCH endpoint to update existing credentials, or the
  DELETE endpoint to delete existing credentials.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.GitProvider = args[0]
		}

		response, err := w.GitCredentials.Create(ctx, createReq)
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
	*workspace.DeleteGitCredentialRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq workspace.DeleteGitCredentialRequest

	// TODO: short flags

	cmd.Use = "delete CREDENTIAL_ID"
	cmd.Short = `Delete a credential.`
	cmd.Long = `Delete a credential.
  
  Deletes the specified Git credential.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}

		err = w.GitCredentials.Delete(ctx, deleteReq)
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
	*workspace.GetGitCredentialRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq workspace.GetGitCredentialRequest

	// TODO: short flags

	cmd.Use = "get CREDENTIAL_ID"
	cmd.Short = `Get a credential entry.`
	cmd.Long = `Get a credential entry.
  
  Gets the Git credential with the specified credential ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}

		response, err := w.GitCredentials.Get(ctx, getReq)
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
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `Get Git credentials.`
	cmd.Long = `Get Git credentials.
  
  Lists the calling user's Git credentials. One credential per user is
  supported.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.GitCredentials.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*workspace.UpdateCredentials,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq workspace.UpdateCredentials

	// TODO: short flags

	cmd.Flags().StringVar(&updateReq.GitProvider, "git-provider", updateReq.GitProvider, `Git provider.`)
	cmd.Flags().StringVar(&updateReq.GitUsername, "git-username", updateReq.GitUsername, `Git username.`)
	cmd.Flags().StringVar(&updateReq.PersonalAccessToken, "personal-access-token", updateReq.PersonalAccessToken, `The personal access token used to authenticate to the corresponding Git provider.`)

	cmd.Use = "update CREDENTIAL_ID"
	cmd.Short = `Update a credential.`
	cmd.Long = `Update a credential.
  
  Updates the specified Git credential.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &updateReq.CredentialId)
		if err != nil {
			return fmt.Errorf("invalid CREDENTIAL_ID: %s", args[0])
		}

		err = w.GitCredentials.Update(ctx, updateReq)
		if err != nil {
			return err
		}
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
