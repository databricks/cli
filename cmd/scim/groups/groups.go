package groups

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "groups",
	Short: `Groups simplify identity management, making it easier to assign access to Databricks Workspace, data, and other securable objects.`, // TODO: fix FirstSentence logic and append dot to summary
}

var createReq scim.Group

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.DisplayName, "display-name", "", `String that represents a human-readable group name.`)
	// TODO: complex arg: entitlements
	createCmd.Flags().StringVar(&createReq.ExternalId, "external-id", "", ``)
	// TODO: complex arg: groups
	createCmd.Flags().StringVar(&createReq.Id, "id", "", `Databricks group ID.`)
	// TODO: complex arg: members
	// TODO: complex arg: roles

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new group Creates a group in the Databricks Workspace with a unique name, using the supplied group details.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Groups.Create(ctx, createReq)
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

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", "", `Unique ID for a group in the Databricks Workspace.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a group Deletes a group from the Databricks Workspace.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Groups.Delete(ctx, deleteReq)
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

	getCmd.Flags().StringVar(&getReq.Id, "id", "", `Unique ID for a group in the Databricks Workspace.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get group details Gets the information for a specific group in the Databricks Workspace.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Groups.Get(ctx, getReq)
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
	// TODO: complex arg: sortOrder
	listCmd.Flags().IntVar(&listReq.StartIndex, "start-index", 0, `Specifies the index of the first result.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List group details Gets all details of the groups associated with the Databricks Workspace.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Groups.ListAll(ctx, listReq)
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
	// TODO: complex arg: operations

}

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: `Update group details Partially updates the details of a group.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Groups.Patch(ctx, patchReq)
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
	// TODO: complex arg: entitlements
	updateCmd.Flags().StringVar(&updateReq.ExternalId, "external-id", "", ``)
	// TODO: complex arg: groups
	updateCmd.Flags().StringVar(&updateReq.Id, "id", "", `Databricks group ID.`)
	// TODO: complex arg: members
	// TODO: complex arg: roles

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Replace a group Updates the details of a group by replacing the entire group entity.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Groups.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
