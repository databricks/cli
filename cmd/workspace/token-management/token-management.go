// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package token_management

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token-management",
		Short: `Enables administrators to get all tokens and delete tokens for other users.`,
		Long: `Enables administrators to get all tokens and delete tokens for other users.
  Admins can either get every token, get a specific token by ID, or get all
  tokens for a particular user.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
	}

	cmd.AddCommand(newCreateOboToken())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())

	return cmd
}

// start create-obo-token command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOboTokenOverrides []func(
	*cobra.Command,
	*settings.CreateOboTokenRequest,
)

func newCreateOboToken() *cobra.Command {
	cmd := &cobra.Command{}

	var createOboTokenReq settings.CreateOboTokenRequest
	var createOboTokenJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createOboTokenJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createOboTokenReq.Comment, "comment", createOboTokenReq.Comment, `Comment that describes the purpose of the token.`)

	cmd.Use = "create-obo-token APPLICATION_ID LIFETIME_SECONDS"
	cmd.Short = `Create on-behalf token.`
	cmd.Long = `Create on-behalf token.
  
  Creates a token on behalf of a service principal.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
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
			err = createOboTokenJson.Unmarshal(&createOboTokenReq)
			if err != nil {
				return err
			}
		} else {
			createOboTokenReq.ApplicationId = args[0]
			_, err = fmt.Sscan(args[1], &createOboTokenReq.LifetimeSeconds)
			if err != nil {
				return fmt.Errorf("invalid LIFETIME_SECONDS: %s", args[1])
			}
		}

		response, err := w.TokenManagement.CreateOboToken(ctx, createOboTokenReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOboTokenOverrides {
		fn(cmd, &createOboTokenReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*settings.DeleteTokenManagementRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq settings.DeleteTokenManagementRequest

	// TODO: short flags

	cmd.Use = "delete TOKEN_ID"
	cmd.Short = `Delete a token.`
	cmd.Long = `Delete a token.
  
  Deletes a token, specified by its ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteReq.TokenId = args[0]

		err = w.TokenManagement.Delete(ctx, deleteReq)
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
	*settings.GetTokenManagementRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq settings.GetTokenManagementRequest

	// TODO: short flags

	cmd.Use = "get TOKEN_ID"
	cmd.Short = `Get token info.`
	cmd.Long = `Get token info.
  
  Gets information about a token, specified by its ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.TokenId = args[0]

		response, err := w.TokenManagement.Get(ctx, getReq)
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
	*settings.ListTokenManagementRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq settings.ListTokenManagementRequest
	var listJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&listReq.CreatedById, "created-by-id", listReq.CreatedById, `User ID of the user that created the token.`)
	cmd.Flags().StringVar(&listReq.CreatedByUsername, "created-by-username", listReq.CreatedByUsername, `Username of the user that created the token.`)

	cmd.Use = "list"
	cmd.Short = `List all tokens.`
	cmd.Long = `List all tokens.
  
  Lists all tokens associated with the specified workspace or user.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
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
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.TokenManagement.ListAll(ctx, listReq)
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
		fn(cmd, &listReq)
	}

	return cmd
}

// end service TokenManagement
