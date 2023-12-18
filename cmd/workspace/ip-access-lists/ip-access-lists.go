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

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip-access-lists",
		Short: `IP Access List enables admins to configure IP access lists.`,
		Long: `IP Access List enables admins to configure IP access lists.
  
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
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*settings.CreateIpAccessList,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq settings.CreateIpAccessList
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create access list.`
	cmd.Long = `Create access list.
  
  Creates an IP access list for this workspace.
  
  A list can be an allow list or a block list. See the top of this file for a
  description of how the server treats allow lists and block lists at runtime.
  
  When creating or updating an IP access list:
  
  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the new list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.
  
  It can take a few minutes for the changes to take effect. **Note**: Your new
  IP access list has no effect until you enable the feature. See
  :method:workspaceconf/setStatus`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.IpAccessLists.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreate())
	})
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*settings.DeleteIpAccessListRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq settings.DeleteIpAccessListRequest

	// TODO: short flags

	cmd.Use = "delete IP_ACCESS_LIST_ID"
	cmd.Short = `Delete access list.`
	cmd.Long = `Delete access list.
  
  Deletes an IP access list, specified by its list ID.

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list to modify.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No IP_ACCESS_LIST_ID argument specified. Loading names for Ip Access Lists drop-down."
			names, err := w.IpAccessLists.IpAccessListInfoLabelToListIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Ip Access Lists drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The ID for the corresponding IP access list to modify")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id for the corresponding ip access list to modify")
		}
		deleteReq.IpAccessListId = args[0]

		err = w.IpAccessLists.Delete(ctx, deleteReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDelete())
	})
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*settings.GetIpAccessListRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq settings.GetIpAccessListRequest

	// TODO: short flags

	cmd.Use = "get IP_ACCESS_LIST_ID"
	cmd.Short = `Get access list.`
	cmd.Long = `Get access list.
  
  Gets an IP access list, specified by its list ID.

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list to modify.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No IP_ACCESS_LIST_ID argument specified. Loading names for Ip Access Lists drop-down."
			names, err := w.IpAccessLists.IpAccessListInfoLabelToListIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Ip Access Lists drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The ID for the corresponding IP access list to modify")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the id for the corresponding ip access list to modify")
		}
		getReq.IpAccessListId = args[0]

		response, err := w.IpAccessLists.Get(ctx, getReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGet())
	})
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `Get access lists.`
	cmd.Long = `Get access lists.
  
  Gets all IP access lists for the specified workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.IpAccessLists.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newList())
	})
}

// start replace command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var replaceOverrides []func(
	*cobra.Command,
	*settings.ReplaceIpAccessList,
)

func newReplace() *cobra.Command {
	cmd := &cobra.Command{}

	var replaceReq settings.ReplaceIpAccessList
	var replaceJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&replaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&replaceReq.ListId, "list-id", replaceReq.ListId, `Universally unique identifier (UUID) of the IP access list.`)

	cmd.Use = "replace"
	cmd.Short = `Replace access list.`
	cmd.Long = `Replace access list.
  
  Replaces an IP access list, specified by its ID.
  
  A list can include allow lists and block lists. See the top of this file for a
  description of how the server treats allow lists and block lists at run time.
  When replacing an IP access list: * For all allow lists and block lists
  combined, the API supports a maximum of 1000 IP/CIDR values, where one CIDR
  counts as a single value. Attempts to exceed that number return error 400 with
  error_code value QUOTA_EXCEEDED. * If the resulting list would block the
  calling user's current IP, error 400 is returned with error_code value
  INVALID_STATE. It can take a few minutes for the changes to take effect.
  Note that your resulting IP access list has no effect until you enable the
  feature. See :method:workspaceconf/setStatus.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = replaceJson.Unmarshal(&replaceReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = w.IpAccessLists.Replace(ctx, replaceReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range replaceOverrides {
		fn(cmd, &replaceReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newReplace())
	})
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*settings.UpdateIpAccessList,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq settings.UpdateIpAccessList
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.ListId, "list-id", updateReq.ListId, `Universally unique identifier (UUID) of the IP access list.`)

	cmd.Use = "update"
	cmd.Short = `Update access list.`
	cmd.Long = `Update access list.
  
  Updates an existing IP access list, specified by its ID.
  
  A list can include allow lists and block lists. See the top of this file for a
  description of how the server treats allow lists and block lists at run time.
  
  When updating an IP access list:
  
  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the updated list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.
  
  It can take a few minutes for the changes to take effect. Note that your
  resulting IP access list has no effect until you enable the feature. See
  :method:workspaceconf/setStatus.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = w.IpAccessLists.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdate())
	})
}

// end service IpAccessLists
