// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"github.com/spf13/cobra"

	automatic_cluster_update "github.com/databricks/cli/cmd/workspace/automatic-cluster-update"
	csp_enablement "github.com/databricks/cli/cmd/workspace/csp-enablement"
	default_namespace "github.com/databricks/cli/cmd/workspace/default-namespace"
	esm_enablement "github.com/databricks/cli/cmd/workspace/esm-enablement"
	restrict_workspace_admins "github.com/databricks/cli/cmd/workspace/restrict-workspace-admins"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Short:   `Workspace Settings API allows users to manage settings at the workspace level.`,
		Long:    `Workspace Settings API allows users to manage settings at the workspace level.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
	}

	// Add subservices
	cmd.AddCommand(automatic_cluster_update.New())
	cmd.AddCommand(csp_enablement.New())
	cmd.AddCommand(default_namespace.New())
	cmd.AddCommand(esm_enablement.New())
	cmd.AddCommand(restrict_workspace_admins.New())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

<<<<<<< HEAD
=======
// start delete-default-namespace-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDefaultNamespaceSettingOverrides []func(
	*cobra.Command,
	*settings.DeleteDefaultNamespaceSettingRequest,
)

func newDeleteDefaultNamespaceSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDefaultNamespaceSettingReq settings.DeleteDefaultNamespaceSettingRequest

	// TODO: short flags

	cmd.Flags().StringVar(&deleteDefaultNamespaceSettingReq.Etag, "etag", deleteDefaultNamespaceSettingReq.Etag, `etag used for versioning.`)

	cmd.Use = "delete-default-namespace-setting"
	cmd.Short = `Delete the default namespace setting.`
	cmd.Long = `Delete the default namespace setting.
  
  Deletes the default namespace setting for the workspace. A fresh etag needs to
  be provided in DELETE requests (as a query parameter). The etag can be
  retrieved by making a GET request before the DELETE request. If the setting is
  updated/deleted concurrently, DELETE will fail with 409 and the request will
  need to be retried by using the fresh etag in the 409 response.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.Settings.DeleteDefaultNamespaceSetting(ctx, deleteDefaultNamespaceSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDefaultNamespaceSettingOverrides {
		fn(cmd, &deleteDefaultNamespaceSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteDefaultNamespaceSetting())
	})
}

// start delete-restrict-workspace-admins-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRestrictWorkspaceAdminsSettingOverrides []func(
	*cobra.Command,
	*settings.DeleteRestrictWorkspaceAdminsSettingRequest,
)

func newDeleteRestrictWorkspaceAdminsSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteRestrictWorkspaceAdminsSettingReq settings.DeleteRestrictWorkspaceAdminsSettingRequest

	// TODO: short flags

	cmd.Flags().StringVar(&deleteRestrictWorkspaceAdminsSettingReq.Etag, "etag", deleteRestrictWorkspaceAdminsSettingReq.Etag, `etag used for versioning.`)

	cmd.Use = "delete-restrict-workspace-admins-setting"
	cmd.Short = `Delete the restrict workspace admins setting.`
	cmd.Long = `Delete the restrict workspace admins setting.
  
  Reverts the restrict workspace admins setting status for the workspace. A
  fresh etag needs to be provided in DELETE requests (as a query parameter). The
  etag can be retrieved by making a GET request before the DELETE request. If
  the setting is updated/deleted concurrently, DELETE will fail with 409 and the
  request will need to be retried by using the fresh etag in the 409 response.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.Settings.DeleteRestrictWorkspaceAdminsSetting(ctx, deleteRestrictWorkspaceAdminsSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteRestrictWorkspaceAdminsSettingOverrides {
		fn(cmd, &deleteRestrictWorkspaceAdminsSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteRestrictWorkspaceAdminsSetting())
	})
}

// start get-default-namespace-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDefaultNamespaceSettingOverrides []func(
	*cobra.Command,
	*settings.GetDefaultNamespaceSettingRequest,
)

func newGetDefaultNamespaceSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var getDefaultNamespaceSettingReq settings.GetDefaultNamespaceSettingRequest

	// TODO: short flags

	cmd.Flags().StringVar(&getDefaultNamespaceSettingReq.Etag, "etag", getDefaultNamespaceSettingReq.Etag, `etag used for versioning.`)

	cmd.Use = "get-default-namespace-setting"
	cmd.Short = `Get the default namespace setting.`
	cmd.Long = `Get the default namespace setting.
  
  Gets the default namespace setting.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.Settings.GetDefaultNamespaceSetting(ctx, getDefaultNamespaceSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDefaultNamespaceSettingOverrides {
		fn(cmd, &getDefaultNamespaceSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetDefaultNamespaceSetting())
	})
}

// start get-restrict-workspace-admins-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRestrictWorkspaceAdminsSettingOverrides []func(
	*cobra.Command,
	*settings.GetRestrictWorkspaceAdminsSettingRequest,
)

func newGetRestrictWorkspaceAdminsSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var getRestrictWorkspaceAdminsSettingReq settings.GetRestrictWorkspaceAdminsSettingRequest

	// TODO: short flags

	cmd.Flags().StringVar(&getRestrictWorkspaceAdminsSettingReq.Etag, "etag", getRestrictWorkspaceAdminsSettingReq.Etag, `etag used for versioning.`)

	cmd.Use = "get-restrict-workspace-admins-setting"
	cmd.Short = `Get the restrict workspace admins setting.`
	cmd.Long = `Get the restrict workspace admins setting.
  
  Gets the restrict workspace admins setting.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.Settings.GetRestrictWorkspaceAdminsSetting(ctx, getRestrictWorkspaceAdminsSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRestrictWorkspaceAdminsSettingOverrides {
		fn(cmd, &getRestrictWorkspaceAdminsSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetRestrictWorkspaceAdminsSetting())
	})
}

// start update-default-namespace-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDefaultNamespaceSettingOverrides []func(
	*cobra.Command,
	*settings.UpdateDefaultNamespaceSettingRequest,
)

func newUpdateDefaultNamespaceSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDefaultNamespaceSettingReq settings.UpdateDefaultNamespaceSettingRequest
	var updateDefaultNamespaceSettingJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateDefaultNamespaceSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-default-namespace-setting"
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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateDefaultNamespaceSettingJson.Unmarshal(&updateDefaultNamespaceSettingReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.Settings.UpdateDefaultNamespaceSetting(ctx, updateDefaultNamespaceSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDefaultNamespaceSettingOverrides {
		fn(cmd, &updateDefaultNamespaceSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdateDefaultNamespaceSetting())
	})
}

// start update-restrict-workspace-admins-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateRestrictWorkspaceAdminsSettingOverrides []func(
	*cobra.Command,
	*settings.UpdateRestrictWorkspaceAdminsSettingRequest,
)

func newUpdateRestrictWorkspaceAdminsSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var updateRestrictWorkspaceAdminsSettingReq settings.UpdateRestrictWorkspaceAdminsSettingRequest
	var updateRestrictWorkspaceAdminsSettingJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateRestrictWorkspaceAdminsSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-restrict-workspace-admins-setting"
	cmd.Short = `Update the restrict workspace admins setting.`
	cmd.Long = `Update the restrict workspace admins setting.
  
  Updates the restrict workspace admins setting for the workspace. A fresh etag
  needs to be provided in PATCH requests (as part of the setting field). The
  etag can be retrieved by making a GET request before the PATCH request. If the
  setting is updated concurrently, PATCH will fail with 409 and the request will
  need to be retried by using the fresh etag in the 409 response.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateRestrictWorkspaceAdminsSettingJson.Unmarshal(&updateRestrictWorkspaceAdminsSettingReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.Settings.UpdateRestrictWorkspaceAdminsSetting(ctx, updateRestrictWorkspaceAdminsSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateRestrictWorkspaceAdminsSettingOverrides {
		fn(cmd, &updateRestrictWorkspaceAdminsSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdateRestrictWorkspaceAdminsSetting())
	})
}

>>>>>>> b0523371 (Add usage string when command fails with incorrect arguments)
// end service Settings
