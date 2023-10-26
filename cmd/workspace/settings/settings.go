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
		Use:     "settings",
		Short:   `// TODO(yuyuan.tang) to add the description for the setting.`,
		Long:    `// TODO(yuyuan.tang) to add the description for the setting`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
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
	cmd.Short = `Delete the default namespace.`
	cmd.Long = `Delete the default namespace.
  
  Deletes the default namespace.`

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
	cmd.Short = `Get the default namespace.`
	cmd.Long = `Get the default namespace.
  
  Gets the default namespace.`

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

	cmd.Flags().BoolVar(&updateDefaultWorkspaceNamespaceReq.AllowMissing, "allow-missing", updateDefaultWorkspaceNamespaceReq.AllowMissing, `This should always be set to true for Settings RPCs.`)
	cmd.Flags().StringVar(&updateDefaultWorkspaceNamespaceReq.FieldMask, "field-mask", updateDefaultWorkspaceNamespaceReq.FieldMask, `Field mask required to be passed into the PATCH request.`)
	// TODO: complex arg: setting

	cmd.Use = "update-default-workspace-namespace"
	cmd.Short = `Updates the default namespace setting.`
	cmd.Long = `Updates the default namespace setting.
  
  Updates the default namespace setting for the workspace. A fresh etag needs to
  be provided in PATCH requests (as part the setting field). The etag can be
  retrieved by making a GET request before the PATCH request. Note that if the
  setting does not exist, GET will return a NOT_FOUND error and the etag will be
  present in the error response, which should be set in the PATCH request.`

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
