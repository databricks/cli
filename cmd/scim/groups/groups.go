// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package groups

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "groups",
	Short: `Groups simplify identity management, making it easier to assign access to Databricks Workspace, data, and other securable objects.`,
	Long: `Groups simplify identity management, making it easier to assign access to
  Databricks Workspace, data, and other securable objects.
  
  It is best practice to assign access to workspaces and access-control policies
  in Unity Catalog to groups, instead of to users individually. All Databricks
  Workspace identities can be assigned as members of groups, and members inherit
  permissions that are assigned to their group.`,
}

// start create command

var createReq scim.Group
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
  
  Creates a group in the Databricks Workspace with a unique name, using the
  supplied group details.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Groups.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq scim.DeleteGroupRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete ID",
	Short: `Delete a group.`,
	Long: `Delete a group.
  
  Deletes a group from the Databricks Workspace.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		deleteReq.Id = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Groups.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq scim.GetGroupRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get ID",
	Short: `Get group details.`,
	Long: `Get group details.
  
  Gets the information for a specific group in the Databricks Workspace.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		getReq.Id = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Groups.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq scim.ListGroupsRequest
var listJson jsonflag.JsonFlag

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
	Short: `List group details.`,
	Long: `List group details.
  
  Gets all details of the groups associated with the Databricks Workspace.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = listJson.Unmarshall(&listReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Groups.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start patch command

var patchReq scim.PartialUpdate
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
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = patchJson.Unmarshall(&patchReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Groups.Patch(ctx, patchReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq scim.Group
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
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Groups.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Groups
