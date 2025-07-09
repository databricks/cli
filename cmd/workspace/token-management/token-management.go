// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package token_management

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

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
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateOboToken())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newList())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newUpdatePermissions())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

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

	cmd.Flags().Var(&createOboTokenJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createOboTokenReq.Comment, "comment", createOboTokenReq.Comment, `Comment that describes the purpose of the token.`)
	cmd.Flags().Int64Var(&createOboTokenReq.LifetimeSeconds, "lifetime-seconds", createOboTokenReq.LifetimeSeconds, `The number of seconds before the token expires.`)

	cmd.Use = "create-obo-token APPLICATION_ID"
	cmd.Short = `Create on-behalf token.`
	cmd.Long = `Create on-behalf token.
  
  Creates a token on behalf of a service principal.

  Arguments:
    APPLICATION_ID: Application ID of the service principal.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'application_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createOboTokenJson.Unmarshal(&createOboTokenReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No APPLICATION_ID argument specified. Loading names for Token Management drop-down."
				names, err := w.TokenManagement.TokenInfoCommentToTokenIdMap(ctx, settings.ListTokenManagementRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Token Management drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Application ID of the service principal")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have application id of the service principal")
			}
			createOboTokenReq.ApplicationId = args[0]
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

	cmd.Use = "delete TOKEN_ID"
	cmd.Short = `Delete a token.`
	cmd.Long = `Delete a token.
  
  Deletes a token, specified by its ID.

  Arguments:
    TOKEN_ID: The ID of the token to revoke.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No TOKEN_ID argument specified. Loading names for Token Management drop-down."
			names, err := w.TokenManagement.TokenInfoCommentToTokenIdMap(ctx, settings.ListTokenManagementRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Token Management drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The ID of the token to revoke")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id of the token to revoke")
		}
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

	cmd.Use = "get TOKEN_ID"
	cmd.Short = `Get token info.`
	cmd.Long = `Get token info.
  
  Gets information about a token, specified by its ID.

  Arguments:
    TOKEN_ID: The ID of the token to get.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No TOKEN_ID argument specified. Loading names for Token Management drop-down."
			names, err := w.TokenManagement.TokenInfoCommentToTokenIdMap(ctx, settings.ListTokenManagementRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Token Management drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The ID of the token to get")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id of the token to get")
		}
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

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-permission-levels"
	cmd.Short = `Get token permission levels.`
	cmd.Long = `Get token permission levels.
  
  Gets the permission levels that a user can have on an object.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.TokenManagement.GetPermissionLevels(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionLevelsOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-permissions"
	cmd.Short = `Get token permissions.`
	cmd.Long = `Get token permissions.
  
  Gets the permissions of all tokens. Tokens can inherit permissions from their
  root object.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.TokenManagement.GetPermissions(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionsOverrides {
		fn(cmd)
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

	cmd.Flags().Int64Var(&listReq.CreatedById, "created-by-id", listReq.CreatedById, `User ID of the user that created the token.`)
	cmd.Flags().StringVar(&listReq.CreatedByUsername, "created-by-username", listReq.CreatedByUsername, `Username of the user that created the token.`)

	cmd.Use = "list"
	cmd.Short = `List all tokens.`
	cmd.Long = `List all tokens.
  
  Lists all tokens associated with the specified workspace or user.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.TokenManagement.List(ctx, listReq)
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

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*settings.TokenPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq settings.TokenPermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions"
	cmd.Short = `Set token permissions.`
	cmd.Long = `Set token permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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

		response, err := w.TokenManagement.SetPermissions(ctx, setPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setPermissionsOverrides {
		fn(cmd, &setPermissionsReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*settings.TokenPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq settings.TokenPermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions"
	cmd.Short = `Update token permissions.`
	cmd.Long = `Update token permissions.
  
  Updates the permissions on all tokens. Tokens can inherit permissions from
  their root object.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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

		response, err := w.TokenManagement.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
	}

	return cmd
}

// end service TokenManagement
