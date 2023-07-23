// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package users

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "users",
	Short: `User identities recognized by Databricks and represented by email addresses.`,
	Long: `User identities recognized by Databricks and represented by email addresses.
  
  Databricks recommends using SCIM provisioning to sync users and groups
  automatically from your identity provider to your Databricks workspace. SCIM
  streamlines onboarding a new employee or team by using your identity provider
  to create users and groups in Databricks workspace and give them the proper
  level of access. When a user leaves your organization or no longer needs
  access to Databricks workspace, admins can terminate the user in your identity
  provider and that user’s account will also be removed from Databricks
  workspace. This ensures a consistent offboarding process and prevents
  unauthorized users from accessing sensitive data.`,
	Annotations: map[string]string{
		"package": "iam",
	},
}

// start create command
var createReq iam.User
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().BoolVar(&createReq.Active, "active", createReq.Active, `If this user is active.`)
	createCmd.Flags().StringVar(&createReq.DisplayName, "display-name", createReq.DisplayName, `String that represents a concatenation of given and family names.`)
	// TODO: array: emails
	// TODO: array: entitlements
	createCmd.Flags().StringVar(&createReq.ExternalId, "external-id", createReq.ExternalId, ``)
	// TODO: array: groups
	createCmd.Flags().StringVar(&createReq.Id, "id", createReq.Id, `Databricks user ID.`)
	// TODO: complex arg: name
	// TODO: array: roles
	createCmd.Flags().StringVar(&createReq.UserName, "user-name", createReq.UserName, `Email address of the Databricks user.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new user.`,
	Long: `Create a new user.
  
  Creates a new user in the Databricks workspace. This new user will also be
  added to the Databricks account.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Users.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start delete command
var deleteReq iam.DeleteUserRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete ID",
	Short: `Delete a user.`,
	Long: `Delete a user.
  
  Deletes a user. Deleting a user from a Databricks workspace also removes
  objects associated with the user.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Users drop-down."
			names, err := w.Users.UserUserNameToIdMap(ctx, iam.ListUsersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Users drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Unique ID for a user in the Databricks workspace")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a user in the databricks workspace")
		}
		deleteReq.Id = args[0]

		err = w.Users.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command
var getReq iam.GetUserRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get ID",
	Short: `Get user details.`,
	Long: `Get user details.
  
  Gets information for a specific user in Databricks workspace.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Users drop-down."
			names, err := w.Users.UserUserNameToIdMap(ctx, iam.ListUsersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Users drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Unique ID for a user in the Databricks workspace")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a user in the databricks workspace")
		}
		getReq.Id = args[0]

		response, err := w.Users.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command
var listReq iam.ListUsersRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().StringVar(&listReq.Attributes, "attributes", listReq.Attributes, `Comma-separated list of attributes to return in response.`)
	listCmd.Flags().IntVar(&listReq.Count, "count", listReq.Count, `Desired number of results per page.`)
	listCmd.Flags().StringVar(&listReq.ExcludedAttributes, "excluded-attributes", listReq.ExcludedAttributes, `Comma-separated list of attributes to exclude in response.`)
	listCmd.Flags().StringVar(&listReq.Filter, "filter", listReq.Filter, `Query by which the results have to be filtered.`)
	listCmd.Flags().StringVar(&listReq.SortBy, "sort-by", listReq.SortBy, `Attribute to sort the results.`)
	listCmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order to sort the results.`)
	listCmd.Flags().IntVar(&listReq.StartIndex, "start-index", listReq.StartIndex, `Specifies the index of the first result.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List users.`,
	Long: `List users.
  
  Gets details for all the users associated with a Databricks workspace.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Users.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start patch command
var patchReq iam.PartialUpdate
var patchJson flags.JsonFlag

func init() {
	Cmd.AddCommand(patchCmd)
	// TODO: short flags
	patchCmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: Operations
	// TODO: array: schema

}

var patchCmd = &cobra.Command{
	Use:   "patch ID",
	Short: `Update user details.`,
	Long: `Update user details.
  
  Partially updates a user resource by applying the supplied operations on
  specific user attributes.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = patchJson.Unmarshal(&patchReq)
			if err != nil {
				return err
			}
		}
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Users drop-down."
			names, err := w.Users.UserUserNameToIdMap(ctx, iam.ListUsersRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Users drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Unique ID for a user in the Databricks workspace")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a user in the databricks workspace")
		}
		patchReq.Id = args[0]

		err = w.Users.Patch(ctx, patchReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start update command
var updateReq iam.User
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().BoolVar(&updateReq.Active, "active", updateReq.Active, `If this user is active.`)
	updateCmd.Flags().StringVar(&updateReq.DisplayName, "display-name", updateReq.DisplayName, `String that represents a concatenation of given and family names.`)
	// TODO: array: emails
	// TODO: array: entitlements
	updateCmd.Flags().StringVar(&updateReq.ExternalId, "external-id", updateReq.ExternalId, ``)
	// TODO: array: groups
	updateCmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Databricks user ID.`)
	// TODO: complex arg: name
	// TODO: array: roles
	updateCmd.Flags().StringVar(&updateReq.UserName, "user-name", updateReq.UserName, `Email address of the Databricks user.`)

}

var updateCmd = &cobra.Command{
	Use:   "update ID",
	Short: `Replace a user.`,
	Long: `Replace a user.
  
  Replaces a user's information with the data supplied in request.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No ID argument specified. Loading names for Users drop-down."
				names, err := w.Users.UserUserNameToIdMap(ctx, iam.ListUsersRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Users drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Databricks user ID")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have databricks user id")
			}
			updateReq.Id = args[0]
		}

		err = w.Users.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service Users
