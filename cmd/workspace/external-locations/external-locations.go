// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package external_locations

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucexternallocations "github.com/databricks/cli/libs/sdktypes/uc/externallocations/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "external-locations",
		Short: `An external location is an object that combines a cloud storage path with a storage credential that authorizes access to the cloud storage path.`,
		Long: `An external location is an object that combines a cloud storage path with a
  storage credential that authorizes access to the cloud storage path. Each
  external location is subject to Unity Catalog access-control policies that
  control which users and groups can access the credential. If a user does not
  have access to an external location in Unity Catalog, the request fails and
  Unity Catalog does not attempt to authenticate to your cloud tenant on the
  user’s behalf.

  Databricks recommends using external locations rather than using storage
  credentials directly.

  To create external locations, you must be a metastore admin or a user with the
  **CREATE_EXTERNAL_LOCATION** privilege.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
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
	*ucexternallocations.CreateExternalLocationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq ucexternallocations.CreateExternalLocationRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.Comment, "comment", `User-provided free-form text description.`)
	// TODO: complex arg: effective_file_event_queue
	flags.BoolPtrVar(cmd.Flags(), &createReq.EnableFileEvents, "enable-file-events", `Whether to enable file events on this external location.`)
	// TODO: complex arg: encryption_details
	flags.BoolPtrVar(cmd.Flags(), &createReq.Fallback, "fallback", `Indicates whether fallback mode is enabled for this external location.`)
	// TODO: complex arg: file_event_queue
	flags.BoolPtrVar(cmd.Flags(), &createReq.ReadOnly, "read-only", `Indicates whether the external location is read-only.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.SkipValidation, "skip-validation", `Skips validation of the storage credential associated with the external location.`)

	cmd.Use = "create NAME URL CREDENTIAL_NAME"
	cmd.Short = `Create an external location.`
	cmd.Long = `Create an external location.

  Creates a new external location entry in the metastore. The caller must be a
  metastore admin or have the **CREATE_EXTERNAL_LOCATION** privilege on both the
  metastore and the associated storage credential.

  Arguments:
    NAME: Name of the external location.
    URL: Path URL of the external location.
    CREDENTIAL_NAME: Name of the storage credential used with this location.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'url', 'credential_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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
		if !cmd.Flags().Changed("json") {
			createReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createReq.Url = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createReq.CredentialName = new(args[2])
		}

		sdkReq, err := ucexternallocations.CreateExternalLocationRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.ExternalLocations.CreateExternalLocation(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucexternallocations.ExternalLocationInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*ucexternallocations.DeleteExternalLocationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq ucexternallocations.DeleteExternalLocationRequest

	flags.BoolPtrVar(cmd.Flags(), &deleteReq.Force, "force", `Force deletion even if there are dependent external tables or mounts.`)

	cmd.Use = "delete NAME"
	cmd.Short = `Delete an external location.`
	cmd.Long = `Delete an external location.

  Deletes the specified external location from the metastore. The caller must be
  the owner of the external location.

  Arguments:
    NAME: Name of the external location.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteReq.NameArg = new(args[0])

		sdkReq, err := ucexternallocations.DeleteExternalLocationRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := w.ExternalLocations.DeleteExternalLocation(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*ucexternallocations.GetExternalLocationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq ucexternallocations.GetExternalLocationRequest

	flags.BoolPtrVar(cmd.Flags(), &getReq.IncludeBrowse, "include-browse", `Whether to include external locations in the response for which the principal can only access selective metadata for.`)

	cmd.Use = "get NAME"
	cmd.Short = `Get an external location.`
	cmd.Long = `Get an external location.

  Gets an external location from the metastore. The caller must be either a
  metastore admin, the owner of the external location, or a user that has some
  privilege on the external location.

  Arguments:
    NAME: Name of the external location.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.NameArg = new(args[0])

		sdkReq, err := ucexternallocations.GetExternalLocationRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.ExternalLocations.GetExternalLocation(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucexternallocations.ExternalLocationInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*ucexternallocations.ListExternalLocationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq ucexternallocations.ListExternalLocationsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeBrowse, "include-browse", `Whether to include external locations in the response for which the principal can only access selective metadata for.`)
	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeUnbound, "include-unbound", `Whether to include external locations not bound to the workspace.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of external locations to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List external locations.`
	cmd.Long = `List external locations.

  Gets an array of external locations (__ExternalLocationInfo__ objects) from
  the metastore. The caller must be a metastore admin, the owner of the external
  location, or a user that has some privilege on the external location. There is
  no guarantee of a specific ordering of the elements in the array.

  NOTE: we recommend using max_results=0 to use the paginated version of this
  API. Unpaginated calls will be deprecated soon.

  PAGINATION BEHAVIOR: When using pagination (max_results >= 0), a page may
  contain zero results while still providing a next_page_token. Clients must
  continue reading pages until next_page_token is absent, which is the only
  indication that the end of results has been reached.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := ucexternallocations.ListExternalLocationsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.ExternalLocations.ListExternalLocationsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucexternallocations.ExternalLocationInfoFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*ucexternallocations.UpdateExternalLocationRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucexternallocations.UpdateExternalLocationRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.Comment, "comment", `User-provided free-form text description.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.CredentialName, "credential-name", `Name of the storage credential used with this location.`)
	// TODO: complex arg: effective_file_event_queue
	flags.BoolPtrVar(cmd.Flags(), &updateReq.EnableFileEvents, "enable-file-events", `Whether to enable file events on this external location.`)
	// TODO: complex arg: encryption_details
	flags.BoolPtrVar(cmd.Flags(), &updateReq.Fallback, "fallback", `Indicates whether fallback mode is enabled for this external location.`)
	// TODO: complex arg: file_event_queue
	flags.BoolPtrVar(cmd.Flags(), &updateReq.Force, "force", `Force update even if changing url invalidates dependent external tables or mounts.`)
	flags.EnumVar(cmd.Flags(), &updateReq.IsolationMode, "isolation-mode", `Supported values: [ISOLATION_MODE_ISOLATED, ISOLATION_MODE_OPEN]`, "IsolationMode", "ISOLATION_MODE_ISOLATED", "ISOLATION_MODE_OPEN")
	flags.StringPtrVar(cmd.Flags(), &updateReq.NewName, "new-name", `New name for the external location.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Owner, "owner", `The owner of the external location.`)
	flags.BoolPtrVar(cmd.Flags(), &updateReq.ReadOnly, "read-only", `Indicates whether the external location is read-only.`)
	flags.BoolPtrVar(cmd.Flags(), &updateReq.SkipValidation, "skip-validation", `Skips validation of the storage credential associated with the external location.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Url, "url", `Path URL of the external location.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update an external location.`
	cmd.Long = `Update an external location.

  Updates an external location in the metastore. The caller must be the owner of
  the external location, or be a metastore admin. In the second case, the admin
  can only update the name of the external location.

  Arguments:
    NAME: Name of the external location.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
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
		updateReq.NameArg = new(args[0])

		sdkReq, err := ucexternallocations.UpdateExternalLocationRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.ExternalLocations.UpdateExternalLocation(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucexternallocations.ExternalLocationInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// end service ExternalLocations
