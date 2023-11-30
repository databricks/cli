// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
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
		Use:   "settings",
		Short: `The default namespace setting API allows users to configure the default namespace for a Databricks workspace.`,
		Long: `The default namespace setting API allows users to configure the default
  namespace for a Databricks workspace.
  
  Through this API, users can retrieve, set, or modify the default namespace
  used when queries do not reference a fully qualified three-level name. For
  example, if you use the API to set 'retail_prod' as the default catalog, then
  a query 'SELECT * FROM myTable' would reference the object
  'retail_prod.default.myTable' (the schema 'default' is always assumed).
  
  This setting requires a restart of clusters and SQL warehouses to take effect.
  Additionally, the default namespace only applies when using Unity
  Catalog-enabled compute.`,
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

// start delete-default-workspace-namespace command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDefaultWorkspaceNamespaceOverrides []func(
	*cobra.Command,
	*settings.DeleteDefaultWorkspaceNamespaceRequest,
)

func newDeleteDefaultWorkspaceNamespace() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDefaultWorkspaceNamespaceReq settings.DeleteDefaultWorkspaceNamespaceRequest

	// TODO: short flags

	cmd.Use = "delete-default-workspace-namespace ETAG"
	cmd.Short = `Delete the default namespace setting.`
	cmd.Long = `Delete the default namespace setting.
  
  Deletes the default namespace setting for the workspace. A fresh etag needs to
  be provided in DELETE requests (as a query parameter). The etag can be
  retrieved by making a GET request before the DELETE request. If the setting is
  updated/deleted concurrently, DELETE will fail with 409 and the request will
  need to be retried by using the fresh etag in the 409 response.

  Arguments:
    ETAG: etag used for versioning. The response is at least as fresh as the eTag
    provided. This is used for optimistic concurrency control as a way to help
    prevent simultaneous writes of a setting overwriting each other. It is
    strongly suggested that systems make use of the etag in the read -> delete
    pattern to perform setting deletions in order to avoid race conditions. That
    is, get an etag from a GET request, and pass it with the DELETE request to
    identify the rule set version you are deleting.
    `

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteDefaultWorkspaceNamespaceReq.Etag = args[0]

		response, err := w.Settings.DeleteDefaultWorkspaceNamespace(ctx, deleteDefaultWorkspaceNamespaceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDefaultWorkspaceNamespaceOverrides {
		fn(cmd, &deleteDefaultWorkspaceNamespaceReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteDefaultWorkspaceNamespace())
	})
}

// start read-default-workspace-namespace command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var readDefaultWorkspaceNamespaceOverrides []func(
	*cobra.Command,
	*settings.ReadDefaultWorkspaceNamespaceRequest,
)

func newReadDefaultWorkspaceNamespace() *cobra.Command {
	cmd := &cobra.Command{}

	var readDefaultWorkspaceNamespaceReq settings.ReadDefaultWorkspaceNamespaceRequest

	// TODO: short flags

	cmd.Use = "read-default-workspace-namespace ETAG"
	cmd.Short = `Get the default namespace setting.`
	cmd.Long = `Get the default namespace setting.
  
  Gets the default namespace setting.

  Arguments:
    ETAG: etag used for versioning. The response is at least as fresh as the eTag
    provided. This is used for optimistic concurrency control as a way to help
    prevent simultaneous writes of a setting overwriting each other. It is
    strongly suggested that systems make use of the etag in the read -> delete
    pattern to perform setting deletions in order to avoid race conditions. That
    is, get an etag from a GET request, and pass it with the DELETE request to
    identify the rule set version you are deleting.
    `

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		readDefaultWorkspaceNamespaceReq.Etag = args[0]

		response, err := w.Settings.ReadDefaultWorkspaceNamespace(ctx, readDefaultWorkspaceNamespaceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range readDefaultWorkspaceNamespaceOverrides {
		fn(cmd, &readDefaultWorkspaceNamespaceReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newReadDefaultWorkspaceNamespace())
	})
}

// start update-default-workspace-namespace command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDefaultWorkspaceNamespaceOverrides []func(
	*cobra.Command,
	*settings.UpdateDefaultWorkspaceNamespaceRequest,
)

func newUpdateDefaultWorkspaceNamespace() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDefaultWorkspaceNamespaceReq settings.UpdateDefaultWorkspaceNamespaceRequest
	var updateDefaultWorkspaceNamespaceJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateDefaultWorkspaceNamespaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateDefaultWorkspaceNamespaceReq.AllowMissing, "allow-missing", updateDefaultWorkspaceNamespaceReq.AllowMissing, `This should always be set to true for Settings API.`)
	cmd.Flags().StringVar(&updateDefaultWorkspaceNamespaceReq.FieldMask, "field-mask", updateDefaultWorkspaceNamespaceReq.FieldMask, `Field mask is required to be passed into the PATCH request.`)
	// TODO: complex arg: setting

	cmd.Use = "update-default-workspace-namespace"
	cmd.Short = `Update the default namespace setting.`
	cmd.Long = `Update the default namespace setting.
  
  Updates the default namespace setting for the workspace. A fresh etag needs to
  be provided in PATCH requests (as part of the setting field). The etag can be
  retrieved by making a GET request before the PATCH request. Note that if the
  setting does not exist, GET will return a NOT_FOUND error and the etag will be
  present in the error response, which should be set in the PATCH request. If
  the setting is updated concurrently, PATCH will fail with 409 and the request
  will need to be retried by using the fresh etag in the 409 response.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateDefaultWorkspaceNamespaceJson.Unmarshal(&updateDefaultWorkspaceNamespaceReq)
			if err != nil {
				return err
			}
		}

		response, err := w.Settings.UpdateDefaultWorkspaceNamespace(ctx, updateDefaultWorkspaceNamespaceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDefaultWorkspaceNamespaceOverrides {
		fn(cmd, &updateDefaultWorkspaceNamespaceReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdateDefaultWorkspaceNamespace())
	})
}

// end service Settings
