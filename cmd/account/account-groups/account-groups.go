// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package account_groups

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/iam"
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

// start create command

var createReq iam.Group
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.DisplayName, "display-name", createReq.DisplayName, `String that represents a human-readable group name.`)
	// TODO: array: entitlements
	createCmd.Flags().StringVar(&createReq.ExternalId, "external-id", createReq.ExternalId, ``)
	// TODO: array: groups
	createCmd.Flags().StringVar(&createReq.Id, "id", createReq.Id, `Databricks group ID.`)
	// TODO: array: members
	// TODO: array: roles

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new group.`,
	Long: `Create a new group.
  
  Creates a group in the Databricks Account with a unique name, using the
  supplied group details.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Id = args[0]

		response, err := a.AccountGroups.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq iam.DeleteAccountGroupRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete ID",
	Short: `Delete a group.`,
	Long: `Delete a group.
  
  Deletes a group from the Databricks Account.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Groups.GroupDisplayNameToIdMap(ctx, iam.ListAccountGroupsRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Unique ID for a group in the Databricks Account")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a group in the databricks account")
		}
		deleteReq.Id = args[0]

		err = a.AccountGroups.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq iam.GetAccountGroupRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get ID",
	Short: `Get group details.`,
	Long: `Get group details.
  
  Gets the information for a specific group in the Databricks Account.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		if len(args) == 0 {
			names, err := a.Groups.GroupDisplayNameToIdMap(ctx, iam.ListAccountGroupsRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Unique ID for a group in the Databricks Account")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a group in the databricks account")
		}
		getReq.Id = args[0]

		response, err := a.AccountGroups.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq iam.ListAccountGroupsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

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
	Short: `List group details.`,
	Long: `List group details.
  
  Gets all details of the groups associated with the Databricks Account.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)

		response, err := a.AccountGroups.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start patch command

var patchReq iam.PartialUpdate
var patchJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(patchCmd)
	// TODO: short flags
	patchCmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: operations

}

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: `Update group details.`,
	Long: `Update group details.
  
  Partially updates the details of a group.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err = patchJson.Unmarshall(&patchReq)
		if err != nil {
			return err
		}
		patchReq.Id = args[0]

		err = a.AccountGroups.Patch(ctx, patchReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq iam.Group
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.DisplayName, "display-name", updateReq.DisplayName, `String that represents a human-readable group name.`)
	// TODO: array: entitlements
	updateCmd.Flags().StringVar(&updateReq.ExternalId, "external-id", updateReq.ExternalId, ``)
	// TODO: array: groups
	updateCmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Databricks group ID.`)
	// TODO: array: members
	// TODO: array: roles

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Replace a group.`,
	Long: `Replace a group.
  
  Updates the details of a group by replacing the entire group entity.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient, // FIXME: accounts client
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := sdk.AccountClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.Id = args[0]

		err = a.AccountGroups.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service AccountGroups
