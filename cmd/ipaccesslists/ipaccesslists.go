// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ipaccesslists

import (
	"github.com/databricks/bricks/lib/jsonflag"
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

// start create command

var createReq ipaccesslists.CreateIpAccessList
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
  :method:workspaceconf/setStatus`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.IpAccessLists.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq ipaccesslists.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete IP_ACCESS_LIST_ID",
	Short: `Delete access list.`,
	Long: `Delete access list.
  
  Deletes an IP access list, specified by its list ID.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		deleteReq.IpAccessListId = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.IpAccessLists.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq ipaccesslists.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get IP_ACCESS_LIST_ID",
	Short: `Get access list.`,
	Long: `Get access list.
  
  Gets an IP access list, specified by its list ID.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		getReq.IpAccessListId = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.IpAccessLists.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get access lists.`,
	Long: `Get access lists.
  
  Gets all IP access lists for the specified workspace.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.IpAccessLists.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start replace command

var replaceReq ipaccesslists.ReplaceIpAccessList
var replaceJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(replaceCmd)
	// TODO: short flags
	replaceCmd.Flags().Var(&replaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	replaceCmd.Flags().StringVar(&replaceReq.ListId, "list-id", replaceReq.ListId, `Universally unique identifier(UUID) of the IP access list.`)

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

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = replaceJson.Unmarshall(&replaceReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.IpAccessLists.Replace(ctx, replaceReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq ipaccesslists.UpdateIpAccessList
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.ListId, "list-id", updateReq.ListId, `Universally unique identifier(UUID) of the IP access list.`)

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

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.IpAccessLists.Update(ctx, updateReq)
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
