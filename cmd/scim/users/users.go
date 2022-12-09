package users

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "users",
	Short: `User identities recognized by Databricks and represented by email addresses.`,
	Long: `User identities recognized by Databricks and represented by email addresses.
  
  Databricks recommends using SCIM provisioning to sync users and groups
  automatically from your identity provider to your Databricks Workspace. SCIM
  streamlines onboarding a new employee or team by using your identity provider
  to create users and groups in Databricks Workspace and give them the proper
  level of access. When a user leaves your organization or no longer needs
  access to Databricks Workspace, admins can terminate the user in your identity
  provider and that user’s account will also be removed from Databricks
  Workspace. This ensures a consistent offboarding process and prevents
  unauthorized users from accessing sensitive data.`,
}

var createReq scim.User

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().BoolVar(&createReq.Active, "active", false, `If this user is active.`)
	createCmd.Flags().StringVar(&createReq.DisplayName, "display-name", "", `String that represents a concatenation of given and family names.`)
	// TODO: array: emails
	// TODO: array: entitlements
	createCmd.Flags().StringVar(&createReq.ExternalId, "external-id", "", ``)
	// TODO: array: groups
	createCmd.Flags().StringVar(&createReq.Id, "id", "", `Databricks user ID.`)
	// TODO: complex arg: name
	// TODO: array: roles
	createCmd.Flags().StringVar(&createReq.UserName, "user-name", "", `Email address of the Databricks user.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new user.`,
	Long: `Create a new user.
  
  Creates a new user in the Databricks Workspace. This new user will also be
  added to the Databricks account.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Users.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq scim.DeleteUserRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", "", `Unique ID for a user in the Databricks Workspace.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a user.`,
	Long: `Delete a user.
  
  Deletes a user. Deleting a user from a Databricks Workspace also removes
  objects associated with the user.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Users.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq scim.GetUserRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Id, "id", "", `Unique ID for a user in the Databricks Workspace.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get user details.`,
	Long: `Get user details.
  
  Gets information for a specific user in Databricks Workspace.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Users.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var listReq scim.ListUsersRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.Attributes, "attributes", "", `Comma-separated list of attributes to return in response.`)
	listCmd.Flags().IntVar(&listReq.Count, "count", 0, `Desired number of results per page.`)
	listCmd.Flags().StringVar(&listReq.ExcludedAttributes, "excluded-attributes", "", `Comma-separated list of attributes to exclude in response.`)
	listCmd.Flags().StringVar(&listReq.Filter, "filter", "", `Query by which the results have to be filtered.`)
	listCmd.Flags().StringVar(&listReq.SortBy, "sort-by", "", `Attribute to sort the results.`)
	listCmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order to sort the results.`)
	listCmd.Flags().IntVar(&listReq.StartIndex, "start-index", 0, `Specifies the index of the first result.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List users.`,
	Long: `List users.
  
  Gets details for all the users associated with a Databricks Workspace.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Users.ListAll(ctx, listReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var patchReq scim.PartialUpdate

func init() {
	Cmd.AddCommand(patchCmd)
	// TODO: short flags

	patchCmd.Flags().StringVar(&patchReq.Id, "id", "", `Unique ID for a group in the Databricks Account.`)
	// TODO: array: operations

}

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: `Update user details.`,
	Long: `Update user details.
  
  Partially updates a user resource by applying the supplied operations on
  specific user attributes.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Users.Patch(ctx, patchReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateReq scim.User

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().BoolVar(&updateReq.Active, "active", false, `If this user is active.`)
	updateCmd.Flags().StringVar(&updateReq.DisplayName, "display-name", "", `String that represents a concatenation of given and family names.`)
	// TODO: array: emails
	// TODO: array: entitlements
	updateCmd.Flags().StringVar(&updateReq.ExternalId, "external-id", "", ``)
	// TODO: array: groups
	updateCmd.Flags().StringVar(&updateReq.Id, "id", "", `Databricks user ID.`)
	// TODO: complex arg: name
	// TODO: array: roles
	updateCmd.Flags().StringVar(&updateReq.UserName, "user-name", "", `Email address of the Databricks user.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Replace a user.`,
	Long: `Replace a user.
  
  Replaces a user's information with the data supplied in request.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Users.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}

// end service Users
