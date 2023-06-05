// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ip_access_lists

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "ip-access-lists",
	Short: `The Accounts IP Access List API enables account admins to configure IP access lists for access to the account console.`,
	Long: `The Accounts IP Access List API enables account admins to configure IP access
  lists for access to the account console.
  
  Account IP Access Lists affect web application access and REST API access to
  the account console and account APIs. If the feature is disabled for the
  account, all access is allowed for this account. There is support for allow
  lists (inclusion) and block lists (exclusion).
  
  When a connection is attempted: 1. **First, all block lists are checked.** If
  the connection IP address matches any block list, the connection is rejected.
  2. **If the connection was not rejected by block lists**, the IP address is
  compared with the allow lists.
  
  If there is at least one allow list for the account, the connection is allowed
  only if the IP address matches an allow list. If there are no allow lists for
  the account, all IP addresses are allowed.
  
  For all allow lists and block lists combined, the account supports a maximum
  of 1000 IP/CIDR values, where one CIDR counts as a single value.
  
  After changes to the account-level IP access lists, it can take a few minutes
  for changes to take effect.`,
}

// start create command

var createReq settings.CreateIpAccessList
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create access list.`,
	Long: `Create access list.
  
  Creates an IP access list for the account.
  
  A list can be an allow list or a block list. See the top of this file for a
  description of how the server treats allow lists and block lists at runtime.
  
  When creating or updating an IP access list:
  
  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the new list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.
  
  It can take a few minutes for the changes to take effect.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.Label = args[0]
			_, err = fmt.Sscan(args[1], &createReq.ListType)
			if err != nil {
				return fmt.Errorf("invalid LIST_TYPE: %s", args[1])
			}
			_, err = fmt.Sscan(args[2], &createReq.IpAddresses)
			if err != nil {
				return fmt.Errorf("invalid IP_ADDRESSES: %s", args[2])
			}
		}

		response, err := a.IpAccessLists.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq settings.DeleteAccountIpAccessListRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete IP_ACCESS_LIST_ID",
	Short: `Delete access list.`,
	Long: `Delete access list.
  
  Deletes an IP access list, specified by its list ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No IP_ACCESS_LIST_ID argument specified. Loading names for Account Ip Access Lists drop-down."
				names, err := a.IpAccessLists.IpAccessListInfoLabelToListIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Account Ip Access Lists drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID for the corresponding IP access list")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the id for the corresponding ip access list")
			}
			deleteReq.IpAccessListId = args[0]
		}

		err = a.IpAccessLists.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq settings.GetAccountIpAccessListRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get IP_ACCESS_LIST_ID",
	Short: `Get IP access list.`,
	Long: `Get IP access list.
  
  Gets an IP access list, specified by its list ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No IP_ACCESS_LIST_ID argument specified. Loading names for Account Ip Access Lists drop-down."
				names, err := a.IpAccessLists.IpAccessListInfoLabelToListIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Account Ip Access Lists drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID for the corresponding IP access list")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the id for the corresponding ip access list")
			}
			getReq.IpAccessListId = args[0]
		}

		response, err := a.IpAccessLists.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
  
  Gets all IP access lists for the specified account.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.IpAccessLists.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start replace command

var replaceReq settings.ReplaceIpAccessList
var replaceJson flags.JsonFlag

func init() {
	Cmd.AddCommand(replaceCmd)
	// TODO: short flags
	replaceCmd.Flags().Var(&replaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	replaceCmd.Flags().StringVar(&replaceReq.ListId, "list-id", replaceReq.ListId, `Universally unique identifier (UUID) of the IP access list.`)

}

var replaceCmd = &cobra.Command{
	Use:   "replace",
	Short: `Replace access list.`,
	Long: `Replace access list.
  
  Replaces an IP access list, specified by its ID.
  
  A list can include allow lists and block lists. See the top of this file for a
  description of how the server treats allow lists and block lists at run time.
  When replacing an IP access list: * For all allow lists and block lists
  combined, the API supports a maximum of 1000 IP/CIDR values, where one CIDR
  counts as a single value. Attempts to exceed that number return error 400 with
  error_code value QUOTA_EXCEEDED. * If the resulting list would block the
  calling user's current IP, error 400 is returned with error_code value
  INVALID_STATE. It can take a few minutes for the changes to take effect.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = replaceJson.Unmarshal(&replaceReq)
			if err != nil {
				return err
			}
		} else {
			replaceReq.Label = args[0]
			_, err = fmt.Sscan(args[1], &replaceReq.ListType)
			if err != nil {
				return fmt.Errorf("invalid LIST_TYPE: %s", args[1])
			}
			_, err = fmt.Sscan(args[2], &replaceReq.IpAddresses)
			if err != nil {
				return fmt.Errorf("invalid IP_ADDRESSES: %s", args[2])
			}
			_, err = fmt.Sscan(args[3], &replaceReq.Enabled)
			if err != nil {
				return fmt.Errorf("invalid ENABLED: %s", args[3])
			}
			replaceReq.IpAccessListId = args[4]
		}

		err = a.IpAccessLists.Replace(ctx, replaceReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq settings.UpdateIpAccessList
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.ListId, "list-id", updateReq.ListId, `Universally unique identifier (UUID) of the IP access list.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update access list.`,
	Long: `Update access list.
  
  Updates an existing IP access list, specified by its ID.
  
  A list can include allow lists and block lists. See the top of this file for a
  description of how the server treats allow lists and block lists at run time.
  
  When updating an IP access list:
  
  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the updated list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.
  
  It can take a few minutes for the changes to take effect.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			updateReq.Label = args[0]
			_, err = fmt.Sscan(args[1], &updateReq.ListType)
			if err != nil {
				return fmt.Errorf("invalid LIST_TYPE: %s", args[1])
			}
			_, err = fmt.Sscan(args[2], &updateReq.IpAddresses)
			if err != nil {
				return fmt.Errorf("invalid IP_ADDRESSES: %s", args[2])
			}
			_, err = fmt.Sscan(args[3], &updateReq.Enabled)
			if err != nil {
				return fmt.Errorf("invalid ENABLED: %s", args[3])
			}
			updateReq.IpAccessListId = args[4]
		}

		err = a.IpAccessLists.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service AccountIpAccessLists
