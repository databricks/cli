// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package disaster_recovery

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/disasterrecovery"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disaster-recovery",
		Short:   `Manage disaster recovery configurations and execute failover operations.`,
		Long:    `Manage disaster recovery configurations and execute failover operations.`,
		GroupID: "disasterrecovery",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateFailoverGroup())
	cmd.AddCommand(newCreateStableUrl())
	cmd.AddCommand(newDeleteFailoverGroup())
	cmd.AddCommand(newDeleteStableUrl())
	cmd.AddCommand(newFailoverFailoverGroup())
	cmd.AddCommand(newGetFailoverGroup())
	cmd.AddCommand(newGetStableUrl())
	cmd.AddCommand(newListFailoverGroups())
	cmd.AddCommand(newListStableUrls())
	cmd.AddCommand(newUpdateFailoverGroup())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-failover-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createFailoverGroupOverrides []func(
	*cobra.Command,
	*disasterrecovery.CreateFailoverGroupRequest,
)

func newCreateFailoverGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var createFailoverGroupReq disasterrecovery.CreateFailoverGroupRequest
	createFailoverGroupReq.FailoverGroup = disasterrecovery.FailoverGroup{}
	var createFailoverGroupJson flags.JsonFlag

	cmd.Flags().Var(&createFailoverGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createFailoverGroupReq.ValidateOnly, "validate-only", createFailoverGroupReq.ValidateOnly, `When true, validates the request without creating the failover group.`)
	cmd.Flags().StringVar(&createFailoverGroupReq.FailoverGroup.Etag, "etag", createFailoverGroupReq.FailoverGroup.Etag, `Opaque version string for optimistic locking.`)
	cmd.Flags().StringVar(&createFailoverGroupReq.FailoverGroup.Name, "name", createFailoverGroupReq.FailoverGroup.Name, `Fully qualified resource name in the format accounts/{account_id}/failover-groups/{failover_group_id}.`)
	// TODO: complex arg: unity_catalog_assets

	cmd.Use = "create-failover-group PARENT FAILOVER_GROUP_ID REGIONS WORKSPACE_SETS INITIAL_PRIMARY_REGION"
	cmd.Short = `Create a Failover Group.`
	cmd.Long = `Create a Failover Group.

  Create a new failover group.

  Arguments:
    PARENT: The parent resource. Format: accounts/{account_id}.
    FAILOVER_GROUP_ID: Client-provided identifier for the failover group. Used to construct the
      resource name as {parent}/failover-groups/{failover_group_id}.
    REGIONS: List of all regions participating in this failover group.
    WORKSPACE_SETS: Workspace sets, each containing workspaces that replicate to each other.
    INITIAL_PRIMARY_REGION: Initial primary region. Used only in Create requests to set the starting
      primary region. Not returned in responses.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT, FAILOVER_GROUP_ID as positional arguments. Provide 'regions', 'workspace_sets', 'initial_primary_region' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createFailoverGroupJson.Unmarshal(&createFailoverGroupReq.FailoverGroup)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createFailoverGroupReq.Parent = args[0]
		createFailoverGroupReq.FailoverGroupId = args[1]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createFailoverGroupReq.FailoverGroup.Regions)
			if err != nil {
				return fmt.Errorf("invalid REGIONS: %s", args[2])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &createFailoverGroupReq.FailoverGroup.WorkspaceSets)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_SETS: %s", args[3])
			}

		}
		if !cmd.Flags().Changed("json") {
			createFailoverGroupReq.FailoverGroup.InitialPrimaryRegion = args[4]
		}

		response, err := a.DisasterRecovery.CreateFailoverGroup(ctx, createFailoverGroupReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createFailoverGroupOverrides {
		fn(cmd, &createFailoverGroupReq)
	}

	return cmd
}

// start create-stable-url command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createStableUrlOverrides []func(
	*cobra.Command,
	*disasterrecovery.CreateStableUrlRequest,
)

func newCreateStableUrl() *cobra.Command {
	cmd := &cobra.Command{}

	var createStableUrlReq disasterrecovery.CreateStableUrlRequest
	createStableUrlReq.StableUrl = disasterrecovery.StableUrl{}
	var createStableUrlJson flags.JsonFlag

	cmd.Flags().Var(&createStableUrlJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createStableUrlReq.ValidateOnly, "validate-only", createStableUrlReq.ValidateOnly, `When true, validates the request without creating the stable URL.`)
	cmd.Flags().StringVar(&createStableUrlReq.StableUrl.Name, "name", createStableUrlReq.StableUrl.Name, `Fully qualified resource name.`)

	cmd.Use = "create-stable-url PARENT STABLE_URL_ID INITIAL_WORKSPACE_ID"
	cmd.Short = `Create a Stable URL.`
	cmd.Long = `Create a Stable URL.

  Create a new stable URL.

  Arguments:
    PARENT: The parent resource. Format: accounts/{account_id}.
    STABLE_URL_ID: Client-provided identifier for the stable URL. Used to construct the
      resource name as {parent}/stable-urls/{stable_url_id}.
    INITIAL_WORKSPACE_ID: The workspace this stable URL is initially bound to. Used only in Create
      requests to associate the stable URL with a workspace. Not returned in
      responses. Mirrors FailoverGroup.initial_primary_region semantics.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT, STABLE_URL_ID as positional arguments. Provide 'initial_workspace_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createStableUrlJson.Unmarshal(&createStableUrlReq.StableUrl)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createStableUrlReq.Parent = args[0]
		createStableUrlReq.StableUrlId = args[1]
		if !cmd.Flags().Changed("json") {
			createStableUrlReq.StableUrl.InitialWorkspaceId = args[2]
		}

		response, err := a.DisasterRecovery.CreateStableUrl(ctx, createStableUrlReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createStableUrlOverrides {
		fn(cmd, &createStableUrlReq)
	}

	return cmd
}

// start delete-failover-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteFailoverGroupOverrides []func(
	*cobra.Command,
	*disasterrecovery.DeleteFailoverGroupRequest,
)

func newDeleteFailoverGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteFailoverGroupReq disasterrecovery.DeleteFailoverGroupRequest

	cmd.Flags().StringVar(&deleteFailoverGroupReq.Etag, "etag", deleteFailoverGroupReq.Etag, `Opaque version string for optimistic locking.`)

	cmd.Use = "delete-failover-group NAME"
	cmd.Short = `Delete a Failover Group.`
	cmd.Long = `Delete a Failover Group.

  Delete a failover group.

  Arguments:
    NAME: The fully qualified resource name of the failover group to delete. Format:
      accounts/{account_id}/failover-groups/{failover_group_id}.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deleteFailoverGroupReq.Name = args[0]

		err = a.DisasterRecovery.DeleteFailoverGroup(ctx, deleteFailoverGroupReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteFailoverGroupOverrides {
		fn(cmd, &deleteFailoverGroupReq)
	}

	return cmd
}

// start delete-stable-url command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteStableUrlOverrides []func(
	*cobra.Command,
	*disasterrecovery.DeleteStableUrlRequest,
)

func newDeleteStableUrl() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteStableUrlReq disasterrecovery.DeleteStableUrlRequest

	cmd.Use = "delete-stable-url NAME"
	cmd.Short = `Delete a Stable URL.`
	cmd.Long = `Delete a Stable URL.

  Delete a stable URL.

  Arguments:
    NAME: The fully qualified resource name. Format:
      accounts/{account_id}/stable-urls/{stable_url_id}.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deleteStableUrlReq.Name = args[0]

		err = a.DisasterRecovery.DeleteStableUrl(ctx, deleteStableUrlReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteStableUrlOverrides {
		fn(cmd, &deleteStableUrlReq)
	}

	return cmd
}

// start failover-failover-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var failoverFailoverGroupOverrides []func(
	*cobra.Command,
	*disasterrecovery.FailoverFailoverGroupRequest,
)

func newFailoverFailoverGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var failoverFailoverGroupReq disasterrecovery.FailoverFailoverGroupRequest
	var failoverFailoverGroupJson flags.JsonFlag

	cmd.Flags().Var(&failoverFailoverGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&failoverFailoverGroupReq.Etag, "etag", failoverFailoverGroupReq.Etag, `Opaque version string for optimistic locking.`)

	cmd.Use = "failover-failover-group NAME TARGET_PRIMARY_REGION FAILOVER_TYPE"
	cmd.Short = `Failover a Failover Group to a new primary region.`
	cmd.Long = `Failover a Failover Group to a new primary region.

  Initiate a failover to a new primary region.

  Arguments:
    NAME: The fully qualified resource name of the failover group to failover.
      Format: accounts/{account_id}/failover-groups/{failover_group_id}.
    TARGET_PRIMARY_REGION: The target primary region. Must be one of the derived regions and
      different from the current effective_primary_region. Serves as an
      idempotency check.
    FAILOVER_TYPE: The type of failover to perform.
      Supported values: [FORCED]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME as positional arguments. Provide 'target_primary_region', 'failover_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := failoverFailoverGroupJson.Unmarshal(&failoverFailoverGroupReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		failoverFailoverGroupReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			failoverFailoverGroupReq.TargetPrimaryRegion = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &failoverFailoverGroupReq.FailoverType)
			if err != nil {
				return fmt.Errorf("invalid FAILOVER_TYPE: %s", args[2])
			}

		}

		response, err := a.DisasterRecovery.FailoverFailoverGroup(ctx, failoverFailoverGroupReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range failoverFailoverGroupOverrides {
		fn(cmd, &failoverFailoverGroupReq)
	}

	return cmd
}

// start get-failover-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getFailoverGroupOverrides []func(
	*cobra.Command,
	*disasterrecovery.GetFailoverGroupRequest,
)

func newGetFailoverGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var getFailoverGroupReq disasterrecovery.GetFailoverGroupRequest

	cmd.Use = "get-failover-group NAME"
	cmd.Short = `Get a Failover Group.`
	cmd.Long = `Get a Failover Group.

  Get a failover group.

  Arguments:
    NAME: The fully qualified resource name of the failover group. Format:
      accounts/{account_id}/failover-groups/{failover_group_id}.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getFailoverGroupReq.Name = args[0]

		response, err := a.DisasterRecovery.GetFailoverGroup(ctx, getFailoverGroupReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getFailoverGroupOverrides {
		fn(cmd, &getFailoverGroupReq)
	}

	return cmd
}

// start get-stable-url command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getStableUrlOverrides []func(
	*cobra.Command,
	*disasterrecovery.GetStableUrlRequest,
)

func newGetStableUrl() *cobra.Command {
	cmd := &cobra.Command{}

	var getStableUrlReq disasterrecovery.GetStableUrlRequest

	cmd.Use = "get-stable-url NAME"
	cmd.Short = `Get a Stable URL.`
	cmd.Long = `Get a Stable URL.

  Get a stable URL.

  Arguments:
    NAME: The fully qualified resource name. Format:
      accounts/{account_id}/stable-urls/{stable_url_id}.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getStableUrlReq.Name = args[0]

		response, err := a.DisasterRecovery.GetStableUrl(ctx, getStableUrlReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getStableUrlOverrides {
		fn(cmd, &getStableUrlReq)
	}

	return cmd
}

// start list-failover-groups command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listFailoverGroupsOverrides []func(
	*cobra.Command,
	*disasterrecovery.ListFailoverGroupsRequest,
)

func newListFailoverGroups() *cobra.Command {
	cmd := &cobra.Command{}

	var listFailoverGroupsReq disasterrecovery.ListFailoverGroupsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listFailoverGroupsLimit int

	cmd.Flags().IntVar(&listFailoverGroupsReq.PageSize, "page-size", listFailoverGroupsReq.PageSize, `Maximum number of failover groups to return per page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listFailoverGroupsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listFailoverGroupsReq.PageToken, "page-token", listFailoverGroupsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-failover-groups PARENT"
	cmd.Short = `List Failover Groups.`
	cmd.Long = `List Failover Groups.

  List failover groups.

  Arguments:
    PARENT: The parent resource. Format: accounts/{account_id}.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		listFailoverGroupsReq.Parent = args[0]

		response := a.DisasterRecovery.ListFailoverGroups(ctx, listFailoverGroupsReq)
		if listFailoverGroupsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listFailoverGroupsLimit)
		}
		if listFailoverGroupsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listFailoverGroupsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listFailoverGroupsOverrides {
		fn(cmd, &listFailoverGroupsReq)
	}

	return cmd
}

// start list-stable-urls command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listStableUrlsOverrides []func(
	*cobra.Command,
	*disasterrecovery.ListStableUrlsRequest,
)

func newListStableUrls() *cobra.Command {
	cmd := &cobra.Command{}

	var listStableUrlsReq disasterrecovery.ListStableUrlsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listStableUrlsLimit int

	cmd.Flags().IntVar(&listStableUrlsReq.PageSize, "page-size", listStableUrlsReq.PageSize, `Maximum number of stable URLs to return per page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listStableUrlsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listStableUrlsReq.PageToken, "page-token", listStableUrlsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-stable-urls PARENT"
	cmd.Short = `List Stable URLs.`
	cmd.Long = `List Stable URLs.

  List stable URLs for an account.

  Arguments:
    PARENT: The parent resource. Format: accounts/{account_id}.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		listStableUrlsReq.Parent = args[0]

		response := a.DisasterRecovery.ListStableUrls(ctx, listStableUrlsReq)
		if listStableUrlsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listStableUrlsLimit)
		}
		if listStableUrlsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listStableUrlsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listStableUrlsOverrides {
		fn(cmd, &listStableUrlsReq)
	}

	return cmd
}

// start update-failover-group command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateFailoverGroupOverrides []func(
	*cobra.Command,
	*disasterrecovery.UpdateFailoverGroupRequest,
)

func newUpdateFailoverGroup() *cobra.Command {
	cmd := &cobra.Command{}

	var updateFailoverGroupReq disasterrecovery.UpdateFailoverGroupRequest
	updateFailoverGroupReq.FailoverGroup = disasterrecovery.FailoverGroup{}
	var updateFailoverGroupJson flags.JsonFlag

	cmd.Flags().Var(&updateFailoverGroupJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateFailoverGroupReq.FailoverGroup.Etag, "etag", updateFailoverGroupReq.FailoverGroup.Etag, `Opaque version string for optimistic locking.`)
	cmd.Flags().StringVar(&updateFailoverGroupReq.FailoverGroup.Name, "name", updateFailoverGroupReq.FailoverGroup.Name, `Fully qualified resource name in the format accounts/{account_id}/failover-groups/{failover_group_id}.`)
	// TODO: complex arg: unity_catalog_assets

	cmd.Use = "update-failover-group NAME UPDATE_MASK REGIONS WORKSPACE_SETS INITIAL_PRIMARY_REGION"
	cmd.Short = `Update a Failover Group.`
	cmd.Long = `Update a Failover Group.

  Update a failover group.

  Arguments:
    NAME: Fully qualified resource name in the format
      accounts/{account_id}/failover-groups/{failover_group_id}.
    UPDATE_MASK: Comma-separated list of fields to update.
    REGIONS: List of all regions participating in this failover group.
    WORKSPACE_SETS: Workspace sets, each containing workspaces that replicate to each other.
    INITIAL_PRIMARY_REGION: Initial primary region. Used only in Create requests to set the starting
      primary region. Not returned in responses.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'regions', 'workspace_sets', 'initial_primary_region' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateFailoverGroupJson.Unmarshal(&updateFailoverGroupReq.FailoverGroup)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateFailoverGroupReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateFailoverGroupReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateFailoverGroupReq.FailoverGroup.Regions)
			if err != nil {
				return fmt.Errorf("invalid REGIONS: %s", args[2])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateFailoverGroupReq.FailoverGroup.WorkspaceSets)
			if err != nil {
				return fmt.Errorf("invalid WORKSPACE_SETS: %s", args[3])
			}

		}
		if !cmd.Flags().Changed("json") {
			updateFailoverGroupReq.FailoverGroup.InitialPrimaryRegion = args[4]
		}

		response, err := a.DisasterRecovery.UpdateFailoverGroup(ctx, updateFailoverGroupReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateFailoverGroupOverrides {
		fn(cmd, &updateFailoverGroupReq)
	}

	return cmd
}

// end service DisasterRecovery
