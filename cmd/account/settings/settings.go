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
	Short: `TBD.`,
	Long:  `TBD`,
	Annotations: map[string]string{
		"package": "settings",
	},

	// This service is being previewed; hide from help output.
	Hidden: true,
}

// start delete-personal-compute-setting command
var deletePersonalComputeSettingReq settings.DeletePersonalComputeSettingRequest
var deletePersonalComputeSettingJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deletePersonalComputeSettingCmd)
	// TODO: short flags
	deletePersonalComputeSettingCmd.Flags().Var(&deletePersonalComputeSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	deletePersonalComputeSettingCmd.Flags().StringVar(&deletePersonalComputeSettingReq.Etag, "etag", deletePersonalComputeSettingReq.Etag, `TBD.`)

}

var deletePersonalComputeSettingCmd = &cobra.Command{
	Use:   "delete-personal-compute-setting",
	Short: `Delete Personal Compute setting.`,
	Long: `Delete Personal Compute setting.
  
  TBD`,

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
			err = deletePersonalComputeSettingJson.Unmarshal(&deletePersonalComputeSettingReq)
			if err != nil {
				return err
			}
		} else {
		}

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
var readPersonalComputeSettingJson flags.JsonFlag

func init() {
	Cmd.AddCommand(readPersonalComputeSettingCmd)
	// TODO: short flags
	readPersonalComputeSettingCmd.Flags().Var(&readPersonalComputeSettingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	readPersonalComputeSettingCmd.Flags().StringVar(&readPersonalComputeSettingReq.Etag, "etag", readPersonalComputeSettingReq.Etag, `TBD.`)

}

var readPersonalComputeSettingCmd = &cobra.Command{
	Use:   "read-personal-compute-setting",
	Short: `Get Personal Compute setting.`,
	Long: `Get Personal Compute setting.
  
  TBD`,

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
			err = readPersonalComputeSettingJson.Unmarshal(&readPersonalComputeSettingReq)
			if err != nil {
				return err
			}
		} else {
		}

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

	updatePersonalComputeSettingCmd.Flags().BoolVar(&updatePersonalComputeSettingReq.AllowMissing, "allow-missing", updatePersonalComputeSettingReq.AllowMissing, `TBD.`)
	// TODO: complex arg: setting

}

var updatePersonalComputeSettingCmd = &cobra.Command{
	Use:   "update-personal-compute-setting",
	Short: `Update Personal Compute setting.`,
	Long: `Update Personal Compute setting.
  
  TBD`,

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
