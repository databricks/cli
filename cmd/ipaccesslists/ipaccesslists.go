package ipaccesslists

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/ipaccesslists"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "ip-access-lists",
	Short: `The IP Access List API enables Databricks admins to configure IP access lists for a workspace.`,
	Long: `The IP Access List API enables Databricks admins to configure IP access lists
  for a workspace.
  
  IP access lists affect web application access and REST API access to this
  workspace only. If the feature is disabled for a workspace, all access is
  allowed for this workspace. There is support for allow lists (inclusion) and
  block lists (exclusion).
  
  When a connection is attempted: 1. **First, all block lists are checked.** If
  the connection IP address matches any block list, the connection is rejected.
  2. **If the connection was not rejected by block lists**, the IP address is
  compared with the allow lists.
  
  If there is at least one allow list for the workspace, the connection is
  allowed only if the IP address matches an allow list. If there are no allow
  lists for the workspace, all IP addresses are allowed.
  
  For all allow lists and block lists combined, the workspace supports a maximum
  of 1000 IP/CIDR values, where one CIDR counts as a single value.
  
  After changes to the IP access list feature, it can take a few minutes for
  changes to take effect.`,
}

var createReq ipaccesslists.CreateIpAccessList

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: array: ip_addresses
	createCmd.Flags().StringVar(&createReq.Label, "label", "", `Label for the IP access list.`)
	createCmd.Flags().Var(&createReq.ListType, "list-type", `This describes an enum.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create access list.`,
	Long: `Create access list.
  
  Creates an IP access list for this workspace. A list can be an allow list or a
  block list. See the top of this file for a description of how the server
  treats allow lists and block lists at runtime.
  
  When creating or updating an IP access list:
  
  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the new list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.
  
  It can take a few minutes for the changes to take effect. **Note**: Your new
  IP access list has no effect until you enable the feature. See
  [/workspace-conf](#operation/set-status).`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.IpAccessLists.Create(ctx, createReq)
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

var deleteReq ipaccesslists.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.IpAccessListId, "ip-access-list-id", "", `The ID for the corresponding IP access list to modify.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete access list.`,
	Long: `Delete access list.
  
  Deletes an IP access list, specified by its list ID.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.IpAccessLists.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq ipaccesslists.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.IpAccessListId, "ip-access-list-id", "", `The ID for the corresponding IP access list to modify.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get access list.`,
	Long: `Get access list.
  
  Gets an IP access list, specified by its list ID.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.IpAccessLists.Get(ctx, getReq)
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

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get access lists.`,
	Long: `Get access lists.
  
  Gets all IP access lists for the specified workspace.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.IpAccessLists.ListAll(ctx)
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

var replaceReq ipaccesslists.ReplaceIpAccessList

func init() {
	Cmd.AddCommand(replaceCmd)
	// TODO: short flags

	replaceCmd.Flags().BoolVar(&replaceReq.Enabled, "enabled", false, `Specifies whether this IP access list is enabled.`)
	replaceCmd.Flags().StringVar(&replaceReq.IpAccessListId, "ip-access-list-id", "", `The ID for the corresponding IP access list to modify.`)
	// TODO: array: ip_addresses
	replaceCmd.Flags().StringVar(&replaceReq.Label, "label", "", `Label for the IP access list.`)
	replaceCmd.Flags().StringVar(&replaceReq.ListId, "list-id", "", `Universally unique identifier(UUID) of the IP access list.`)
	replaceCmd.Flags().Var(&replaceReq.ListType, "list-type", `This describes an enum.`)

}

var replaceCmd = &cobra.Command{
	Use:   "replace",
	Short: `Replace access list.`,
	Long: `Replace access list.
  
  Replaces an IP access list, specified by its ID. A list can include allow
  lists and block lists. See the top of this file for a description of how the
  server treats allow lists and block lists at run time. When replacing an IP
  access list: * For all allow lists and block lists combined, the API supports
  a maximum of 1000 IP/CIDR values, where one CIDR counts as a single value.
  Attempts to exceed that number return error 400 with error_code value
  QUOTA_EXCEEDED. * If the resulting list would block the calling user's
  current IP, error 400 is returned with error_code value INVALID_STATE. It
  can take a few minutes for the changes to take effect. Note that your
  resulting IP access list has no effect until you enable the feature. See
  :method:workspaceconf/setStatus.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.IpAccessLists.Replace(ctx, replaceReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateReq ipaccesslists.UpdateIpAccessList

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().BoolVar(&updateReq.Enabled, "enabled", false, `Specifies whether this IP access list is enabled.`)
	updateCmd.Flags().StringVar(&updateReq.IpAccessListId, "ip-access-list-id", "", `The ID for the corresponding IP access list to modify.`)
	// TODO: array: ip_addresses
	updateCmd.Flags().StringVar(&updateReq.Label, "label", "", `Label for the IP access list.`)
	updateCmd.Flags().StringVar(&updateReq.ListId, "list-id", "", `Universally unique identifier(UUID) of the IP access list.`)
	updateCmd.Flags().Var(&updateReq.ListType, "list-type", `This describes an enum.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update access list.`,
	Long: `Update access list.
  
  Updates an existing IP access list, specified by its ID. A list can include
  allow lists and block lists. See the top of this file for a description of how
  the server treats allow lists and block lists at run time.
  
  When updating an IP access list:
  
  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the updated list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.
  
  It can take a few minutes for the changes to take effect. Note that your
  resulting IP access list has no effect until you enable the feature. See
  :method:workspaceconf/setStatus.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.IpAccessLists.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}

// end service IpAccessLists

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
