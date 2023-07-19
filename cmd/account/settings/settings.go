// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package settings

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
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
	Annotations: map[string]string{
		"package": "settings",
	},

	// This service is being previewed; hide from help output.
	Hidden: true,
}

// start delete-personal-compute-setting command
var deletePersonalComputeSettingReq settings.DeletePersonalComputeSettingRequest

func init() {
	Cmd.AddCommand(deletePersonalComputeSettingCmd)
	// TODO: short flags

}

var deletePersonalComputeSettingCmd = &cobra.Command{
	Use:   "delete-personal-compute-setting ETAG",
	Short: `Delete Personal Compute setting.`,
	Long: `Delete Personal Compute setting.
  
  Reverts back the Personal Compute setting value to default (ON)`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		deletePersonalComputeSettingReq.Etag = args[0]

		response, err := a.Settings.DeletePersonalComputeSetting(ctx, deletePersonalComputeSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start read-personal-compute-setting command
var readPersonalComputeSettingReq settings.ReadPersonalComputeSettingRequest

func init() {
	Cmd.AddCommand(readPersonalComputeSettingCmd)
	// TODO: short flags

}

var readPersonalComputeSettingCmd = &cobra.Command{
	Use:   "read-personal-compute-setting ETAG",
	Short: `Get Personal Compute setting.`,
	Long: `Get Personal Compute setting.
  
  Gets the value of the Personal Compute setting.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		readPersonalComputeSettingReq.Etag = args[0]

		response, err := a.Settings.ReadPersonalComputeSetting(ctx, readPersonalComputeSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start update-personal-compute-setting command
var updatePersonalComputeSettingReq settings.UpdatePersonalComputeSettingRequest
var updatePersonalComputeSettingJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updatePersonalComputeSettingCmd)
	// TODO: short flags
	updatePersonalComputeSettingCmd.Flags().Var(&updatePersonalComputeSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updatePersonalComputeSettingCmd.Flags().BoolVar(&updatePersonalComputeSettingReq.AllowMissing, "allow-missing", updatePersonalComputeSettingReq.AllowMissing, `This should always be set to true for Settings RPCs.`)
	// TODO: complex arg: setting

}

var updatePersonalComputeSettingCmd = &cobra.Command{
	Use:   "update-personal-compute-setting",
	Short: `Update Personal Compute setting.`,
	Long: `Update Personal Compute setting.
  
  Updates the value of the Personal Compute setting.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updatePersonalComputeSettingJson.Unmarshal(&updatePersonalComputeSettingReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := a.Settings.UpdatePersonalComputeSetting(ctx, updatePersonalComputeSettingReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service AccountSettings
