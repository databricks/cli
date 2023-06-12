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
}

// end service AccountSettings
