// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

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
		Use:   "settings",
		Short: `The Personal Compute enablement setting lets you control which users can use the Personal Compute default policy to create compute resources.`,
		Long: `The Personal Compute enablement setting lets you control which users can use
  the Personal Compute default policy to create compute resources. By default
  all users in all workspaces have access (ON), but you can change the setting
  to instead let individual workspaces configure access control (DELEGATE).
  
  There is only one instance of this setting per account. Since this setting has
  a default value, this setting is present on all accounts even though it's
  never set on a given account. Deletion reverts the value of the setting back
  to the default value.`,
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

// start delete-personal-compute-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deletePersonalComputeSettingOverrides []func(
	*cobra.Command,
	*settings.DeletePersonalComputeSettingRequest,
)

func newDeletePersonalComputeSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var deletePersonalComputeSettingReq settings.DeletePersonalComputeSettingRequest

	// TODO: short flags

	cmd.Flags().StringVar(&deletePersonalComputeSettingReq.Etag, "etag", deletePersonalComputeSettingReq.Etag, `etag used for versioning.`)

	cmd.Use = "delete-personal-compute-setting"
	cmd.Short = `Delete Personal Compute setting.`
	cmd.Long = `Delete Personal Compute setting.
  
  Reverts back the Personal Compute setting value to default (ON)`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		response, err := a.Settings.DeletePersonalComputeSetting(ctx, deletePersonalComputeSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deletePersonalComputeSettingOverrides {
		fn(cmd, &deletePersonalComputeSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeletePersonalComputeSetting())
	})
}

// start get-personal-compute-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPersonalComputeSettingOverrides []func(
	*cobra.Command,
	*settings.GetPersonalComputeSettingRequest,
)

func newGetPersonalComputeSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var getPersonalComputeSettingReq settings.GetPersonalComputeSettingRequest

	// TODO: short flags

	cmd.Flags().StringVar(&getPersonalComputeSettingReq.Etag, "etag", getPersonalComputeSettingReq.Etag, `etag used for versioning.`)

	cmd.Use = "get-personal-compute-setting"
	cmd.Short = `Get Personal Compute setting.`
	cmd.Long = `Get Personal Compute setting.
  
  Gets the value of the Personal Compute setting.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		response, err := a.Settings.GetPersonalComputeSetting(ctx, getPersonalComputeSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPersonalComputeSettingOverrides {
		fn(cmd, &getPersonalComputeSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetPersonalComputeSetting())
	})
}

// start update-personal-compute-setting command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePersonalComputeSettingOverrides []func(
	*cobra.Command,
	*settings.UpdatePersonalComputeSettingRequest,
)

func newUpdatePersonalComputeSetting() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePersonalComputeSettingReq settings.UpdatePersonalComputeSettingRequest
	var updatePersonalComputeSettingJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updatePersonalComputeSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-personal-compute-setting"
	cmd.Short = `Update Personal Compute setting.`
	cmd.Long = `Update Personal Compute setting.
  
  Updates the value of the Personal Compute setting.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updatePersonalComputeSettingJson.Unmarshal(&updatePersonalComputeSettingReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := a.Settings.UpdatePersonalComputeSetting(ctx, updatePersonalComputeSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePersonalComputeSettingOverrides {
		fn(cmd, &updatePersonalComputeSettingReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdatePersonalComputeSetting())
	})
}

// end service AccountSettings
