package account_groups

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "account-groups",
	Short: `Groups simplify identity management, making it easier to assign access to Databricks Account, data, and other securable objects.`,
	Long: `Groups simplify identity management, making it easier to assign access to
  Databricks Account, data, and other securable objects.
  
  It is best practice to assign access to workspaces and access-control policies
  in Unity Catalog to groups, instead of to users individually. All Databricks
  Account identities can be assigned as members of groups, and members inherit
  permissions that are assigned to their group.`,
}

var createReq scim.Group

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.DisplayName, "display-name", "", `String that represents a human-readable group name.`)
	// TODO: array: entitlements
	createCmd.Flags().StringVar(&createReq.ExternalId, "external-id", "", ``)
	// TODO: array: groups
	createCmd.Flags().StringVar(&createReq.Id, "id", "", `Databricks group ID.`)
	// TODO: array: members
	// TODO: array: roles

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new group.`,
	Long: `Create a new group.
  
  Creates a group in the Databricks Account with a unique name, using the
  supplied group details.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Groups.Create(ctx, createReq)
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

var deleteReq scim.DeleteGroupRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", "", `Unique ID for a group in the Databricks Account.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a group.`,
	Long: `Delete a group.
  
  Deletes a group from the Databricks Account.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.Groups.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq scim.GetGroupRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Id, "id", "", `Unique ID for a group in the Databricks Account.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get group details.`,
	Long: `Get group details.
  
  Gets the information for a specific group in the Databricks Account.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Groups.Get(ctx, getReq)
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

var listReq scim.ListGroupsRequest

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
	Short: `List group details.`,
	Long: `List group details.
  
  Gets all details of the groups associated with the Databricks Account.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		response, err := a.Groups.ListAll(ctx, listReq)
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
	Short: `Update group details.`,
	Long: `Update group details.
  
  Partially updates the details of a group.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.Groups.Patch(ctx, patchReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateReq scim.Group

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.DisplayName, "display-name", "", `String that represents a human-readable group name.`)
	// TODO: array: entitlements
	updateCmd.Flags().StringVar(&updateReq.ExternalId, "external-id", "", ``)
	// TODO: array: groups
	updateCmd.Flags().StringVar(&updateReq.Id, "id", "", `Databricks group ID.`)
	// TODO: array: members
	// TODO: array: roles

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Replace a group.`,
	Long: `Replace a group.
  
  Updates the details of a group by replacing the entire group entity.`,

	PreRunE: sdk.PreAccountClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err := a.Groups.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}

// end service AccountGroups
