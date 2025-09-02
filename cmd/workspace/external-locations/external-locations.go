// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package external_locations

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*catalog.CreateExternalLocation,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateExternalLocation
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().BoolVar(&createReq.EnableFileEvents, "enable-file-events", createReq.EnableFileEvents, `Whether to enable file events on this external location.`)
	// TODO: complex arg: encryption_details
	cmd.Flags().BoolVar(&createReq.Fallback, "fallback", createReq.Fallback, `Indicates whether fallback mode is enabled for this external location.`)
	// TODO: complex arg: file_event_queue
	cmd.Flags().BoolVar(&createReq.ReadOnly, "read-only", createReq.ReadOnly, `Indicates whether the external location is read-only.`)
	cmd.Flags().BoolVar(&createReq.SkipValidation, "skip-validation", createReq.SkipValidation, `Skips validation of the storage credential associated with the external location.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'url', 'credential_name' in your JSON input")
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
			createReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createReq.Url = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createReq.CredentialName = args[2]
		}

		response, err := w.ExternalLocations.Create(ctx, createReq)
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
	*catalog.DeleteExternalLocationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteExternalLocationRequest

	cmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if there are dependent external tables or mounts.`)

	cmd.Use = "delete NAME"
	cmd.Short = `Delete an external location.`
	cmd.Long = `Delete an external location.
  
  Deletes the specified external location from the metastore. The caller must be
  the owner of the external location.

  Arguments:
    NAME: Name of the external location.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Name = args[0]

		err = w.ExternalLocations.Delete(ctx, deleteReq)
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
	*catalog.GetExternalLocationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetExternalLocationRequest

	cmd.Flags().BoolVar(&getReq.IncludeBrowse, "include-browse", getReq.IncludeBrowse, `Whether to include external locations in the response for which the principal can only access selective metadata for.`)

	cmd.Use = "get NAME"
	cmd.Short = `Get an external location.`
	cmd.Long = `Get an external location.
  
  Gets an external location from the metastore. The caller must be either a
  metastore admin, the owner of the external location, or a user that has some
  privilege on the external location.

  Arguments:
    NAME: Name of the external location.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Name = args[0]

		response, err := w.ExternalLocations.Get(ctx, getReq)
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
	*catalog.ListExternalLocationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListExternalLocationsRequest

	cmd.Flags().BoolVar(&listReq.IncludeBrowse, "include-browse", listReq.IncludeBrowse, `Whether to include external locations in the response for which the principal can only access selective metadata for.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of external locations to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List external locations.`
	cmd.Long = `List external locations.
  
  Gets an array of external locations (__ExternalLocationInfo__ objects) from
  the metastore. The caller must be a metastore admin, the owner of the external
  location, or a user that has some privilege on the external location. There is
  no guarantee of a specific ordering of the elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ExternalLocations.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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
	*catalog.UpdateExternalLocation,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateExternalLocation
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().StringVar(&updateReq.CredentialName, "credential-name", updateReq.CredentialName, `Name of the storage credential used with this location.`)
	cmd.Flags().BoolVar(&updateReq.EnableFileEvents, "enable-file-events", updateReq.EnableFileEvents, `Whether to enable file events on this external location.`)
	// TODO: complex arg: encryption_details
	cmd.Flags().BoolVar(&updateReq.Fallback, "fallback", updateReq.Fallback, `Indicates whether fallback mode is enabled for this external location.`)
	// TODO: complex arg: file_event_queue
	cmd.Flags().BoolVar(&updateReq.Force, "force", updateReq.Force, `Force update even if changing url invalidates dependent external tables or mounts.`)
	cmd.Flags().Var(&updateReq.IsolationMode, "isolation-mode", `Supported values: [ISOLATION_MODE_ISOLATED, ISOLATION_MODE_OPEN]`)
	cmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `New name for the external location.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The owner of the external location.`)
	cmd.Flags().BoolVar(&updateReq.ReadOnly, "read-only", updateReq.ReadOnly, `Indicates whether the external location is read-only.`)
	cmd.Flags().BoolVar(&updateReq.SkipValidation, "skip-validation", updateReq.SkipValidation, `Skips validation of the storage credential associated with the external location.`)
	cmd.Flags().StringVar(&updateReq.Url, "url", updateReq.Url, `Path URL of the external location.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update an external location.`
	cmd.Long = `Update an external location.
  
  Updates an external location in the metastore. The caller must be the owner of
  the external location, or be a metastore admin. In the second case, the admin
  can only update the name of the external location.

  Arguments:
    NAME: Name of the external location.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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
		updateReq.Name = args[0]

		response, err := w.ExternalLocations.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
