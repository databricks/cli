// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package csp_enablement_account

import (
	"errors"
	"fmt"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "csp-enablement-account",
		Short: `*Public Preview* The compliance security profile settings at the account level control whether to enable it for new workspaces.`,
		Long: `This command is in Public Preview and may change without notice.

The compliance security profile settings at the account level control whether
  to enable it for new workspaces. By default, this account-level setting is
  disabled for new workspaces. After workspace creation, account admins can
  enable the compliance security profile individually for each workspace.

  This settings can be disabled so that new workspaces do not have compliance
  security profile enabled by default.`,
		RunE: root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq map[string]any
	getReq = map[string]any{}
	var getEtagRaw string

	cmd.Flags().StringVar(&getEtagRaw, "etag", getEtagRaw, `etag used for versioning.`)

	cmd.Use = "get"
	cmd.Short = `*Public Preview* Get the compliance security profile setting for new workspaces.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get the compliance security profile setting for new workspaces.

  Gets the compliance security profile setting for new workspaces.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("etag") || getEtagRaw != "" {
			rawQuery["etag"] = getEtagRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/accounts/%s/settings/types/shield_csp_enablement_ac/names/default", a.Config.AccountID)

		maps.Copy(rawQuery, getReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq map[string]any
	updateReq = map[string]any{}
	var updateAllowMissingRaw bool
	var updateFieldMaskRaw string

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update"
	cmd.Short = `*Public Preview* Update the compliance security profile setting for new workspaces.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update the compliance security profile setting for new workspaces.

  Updates the value of the compliance security profile setting for new
  workspaces.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

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
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("allow-missing") || updateAllowMissingRaw != false {
			updateReq["allow_missing"] = updateAllowMissingRaw
		}
		if cmd.Flags().Changed("field-mask") || updateFieldMaskRaw != "" {
			updateReq["field_mask"] = updateFieldMaskRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/accounts/%s/settings/types/shield_csp_enablement_ac/names/default", a.Config.AccountID)

		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, nil, rawQuery, updateReq, &response)
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

// end service CspEnablementAccount
