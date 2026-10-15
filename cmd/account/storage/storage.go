// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package storage

import (
	"errors"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	storageconfigurations "github.com/databricks/cli/libs/sdktypes/storageconfigurations/v1"
	sdkstorageconfigurations "github.com/databricks/sdk-go/storageconfigurations/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: `These APIs manage storage configurations for this workspace.`,
		Long: `These APIs manage storage configurations for this workspace. A root storage S3
  bucket in your account is required to store objects like cluster logs,
  notebook revisions, and job results. You can also use the root storage S3
  bucket for storage of non-production DBFS data. A storage configuration
  encapsulates this bucket information, and its ID is used when creating a new
  workspace.`,
		GroupID: "provisioning",
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
	*storageconfigurations.CreateStorageConfigurationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq storageconfigurations.CreateStorageConfigurationRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.RoleArn, "role-arn", `Optional IAM role that is used to access the workspace catalog which is created during workspace creation for UC by Default.`)

	cmd.Use = "create"
	cmd.Short = `Create a storage configuration.`
	cmd.Long = `Create a storage configuration.

  Creates a Databricks storage configuration for an account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

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
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := storageconfigurations.CreateStorageConfigurationRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.StorageConfigurations.CreateStorageConfigurationPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := storageconfigurations.StorageConfigurationFromSDK(response)
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
	*sdkstorageconfigurations.DeleteStorageConfigurationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkstorageconfigurations.DeleteStorageConfigurationRequest

	cmd.Use = "delete STORAGE_CONFIGURATION_ID"
	cmd.Short = `Delete a storage configuration.`
	cmd.Long = `Delete a storage configuration.

  Deletes a Databricks storage configuration. You cannot delete a storage
  configuration that is associated with any workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteReq.StorageConfigurationId = new(args[0])

		sdkReq := &deleteReq
		response, err := a.StorageConfigurations.DeleteStorageConfigurationPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := storageconfigurations.StorageConfigurationFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkstorageconfigurations.GetStorageConfigurationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkstorageconfigurations.GetStorageConfigurationRequest

	cmd.Use = "get STORAGE_CONFIGURATION_ID"
	cmd.Short = `Get a storage configuration.`
	cmd.Long = `Get a storage configuration.

  Gets a Databricks storage configuration for an account, both specified by ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getReq.StorageConfigurationId = new(args[0])

		sdkReq := &getReq
		response, err := a.StorageConfigurations.GetStorageConfigurationPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := storageconfigurations.StorageConfigurationFromSDK(response)
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
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `List storage configurations.`
	cmd.Long = `List storage configurations.

  Lists Databricks storage configurations for an account, specified by ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq := &sdkstorageconfigurations.ListStorageConfigurationRequest{}
		response, err := a.StorageConfigurations.ListStorageConfigurationPublic(ctx, *sdkReq)
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
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// end service Storage
