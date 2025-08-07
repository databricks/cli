// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ip_access_lists

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newReplace())
	cmd.AddCommand(newUpdate())

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

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: ip_addresses

	cmd.Use = "create LABEL LIST_TYPE"
	cmd.Short = `Create access list.`
	cmd.Long = `Create access list.
  
  Creates an IP access list for the account.
  
  A list can be an allow list or a block list. See the top of this file for a
  description of how the server treats allow lists and block lists at runtime.
  
  When creating or updating an IP access list:
  
  * For all allow lists and block lists combined, the API supports a maximum of
  1000 IP/CIDR values, where one CIDR counts as a single value. Attempts to
  exceed that number return error 400 with error_code value QUOTA_EXCEEDED.
  * If the new list would block the calling user's current IP, error 400 is
  returned with error_code value INVALID_STATE.
  
  It can take a few minutes for the changes to take effect.

  Arguments:
    LABEL: Label for the IP access list. This **cannot** be empty.
    LIST_TYPE:  
      Supported values: [ALLOW, BLOCK]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'label', 'list_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.Label = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createReq.ListType)
			if err != nil {
				return fmt.Errorf("invalid LIST_TYPE: %s", args[1])
			}
		}

		response, err := a.IpAccessLists.Create(ctx, createReq)
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

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*settings.DeleteAccountIpAccessListRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq settings.DeleteAccountIpAccessListRequest

	cmd.Use = "delete IP_ACCESS_LIST_ID"
	cmd.Short = `Delete access list.`
	cmd.Long = `Delete access list.
  
  Deletes an IP access list, specified by its list ID.

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

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

		err = a.IpAccessLists.Delete(ctx, deleteReq)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*settings.GetAccountIpAccessListRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq settings.GetAccountIpAccessListRequest

	cmd.Use = "get IP_ACCESS_LIST_ID"
	cmd.Short = `Get IP access list.`
	cmd.Long = `Get IP access list.
  
  Gets an IP access list, specified by its list ID.

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

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

		response, err := a.IpAccessLists.Get(ctx, getReq)
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
  
  Gets all IP access lists for the specified account.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		response := a.IpAccessLists.List(ctx)
		return cmdio.RenderIterator(ctx, response)
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

	cmd.Flags().Var(&replaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: ip_addresses

	cmd.Use = "replace IP_ACCESS_LIST_ID LABEL LIST_TYPE ENABLED"
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

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list
    LABEL: Label for the IP access list. This **cannot** be empty.
    LIST_TYPE:  
      Supported values: [ALLOW, BLOCK]
    ENABLED: Specifies whether this IP access list is enabled.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only IP_ACCESS_LIST_ID as positional arguments. Provide 'label', 'list_type', 'enabled' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := replaceJson.Unmarshal(&replaceReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		replaceReq.IpAccessListId = args[0]
		if !cmd.Flags().Changed("json") {
			replaceReq.Label = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &replaceReq.ListType)
			if err != nil {
				return fmt.Errorf("invalid LIST_TYPE: %s", args[2])
			}
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &replaceReq.Enabled)
			if err != nil {
				return fmt.Errorf("invalid ENABLED: %s", args[3])
			}
		}

		err = a.IpAccessLists.Replace(ctx, replaceReq)
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

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateReq.Enabled, "enabled", updateReq.Enabled, `Specifies whether this IP access list is enabled.`)
	// TODO: array: ip_addresses
	cmd.Flags().StringVar(&updateReq.Label, "label", updateReq.Label, `Label for the IP access list.`)
	cmd.Flags().Var(&updateReq.ListType, "list-type", `Supported values: [ALLOW, BLOCK]`)

	cmd.Use = "update IP_ACCESS_LIST_ID"
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
  
  It can take a few minutes for the changes to take effect.

  Arguments:
    IP_ACCESS_LIST_ID: The ID for the corresponding IP access list`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
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
		updateReq.IpAccessListId = args[0]

		err = a.IpAccessLists.Update(ctx, updateReq)
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

// end service AccountIpAccessLists
