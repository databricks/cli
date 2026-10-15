// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package private_access

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	networking "github.com/databricks/cli/libs/sdktypes/networking/v1"
	sdknetworking "github.com/databricks/sdk-go/networking/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "private-access",
		Short:   `These APIs manage private access settings for this account.`,
		Long:    `These APIs manage private access settings for this account.`,
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
	cmd.AddCommand(newReplace())

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
	*networking.CreatePrivateAccessSettingsRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq networking.CreatePrivateAccessSettingsRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: allowed_vpc_endpoint_ids
	flags.EnumVar(cmd.Flags(), &createReq.PrivateAccessLevel, "private-access-level", `The private access level controls which VPC endpoints can connect to the UI or API of any workspace that attaches this private access settings object. Supported values: [ACCOUNT, ENDPOINT]`, "PrivateAccessLevel", "ACCOUNT", "ENDPOINT")
	flags.StringPtrVar(cmd.Flags(), &createReq.PrivateAccessSettingsName, "private-access-settings-name", `The human-readable name of the private access settings object.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.PublicAccessEnabled, "public-access-enabled", `Determines if the workspace can be accessed over public internet.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Region, "region", `The AWS region for workspaces attached to this private access settings object.`)

	cmd.Use = "create"
	cmd.Short = `Create private access settings.`
	cmd.Long = `Create private access settings.

  Creates a private access settings configuration, which represents network
  access restrictions for workspace resources. Private access settings configure
  whether workspaces can be accessed from the public internet or only from
  private endpoints.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

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
		}

		sdkReq, err := networking.CreatePrivateAccessSettingsRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.Networking.CreatePrivateAccessSettingsPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.PrivateAccessSettingsFromSDK(response)
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
	*sdknetworking.DeletePrivateAccessSettingsRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdknetworking.DeletePrivateAccessSettingsRequest

	cmd.Use = "delete PRIVATE_ACCESS_SETTINGS_ID"
	cmd.Short = `Delete private access settings.`
	cmd.Long = `Delete private access settings.

  Deletes a Databricks private access settings configuration, both specified by
  ID.`

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

		deleteReq.PrivateAccessSettingsId = new(args[0])

		sdkReq := &deleteReq
		response, err := a.Networking.DeletePrivateAccessSettingsPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.PrivateAccessSettingsFromSDK(response)
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
	*sdknetworking.GetPrivateAccessSettingsRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdknetworking.GetPrivateAccessSettingsRequest

	cmd.Use = "get PRIVATE_ACCESS_SETTINGS_ID"
	cmd.Short = `Get private access settings.`
	cmd.Long = `Get private access settings.

  Gets a Databricks private access settings configuration, both specified by ID.`

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

		getReq.PrivateAccessSettingsId = new(args[0])

		sdkReq := &getReq
		response, err := a.Networking.GetPrivateAccessSettingsPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.PrivateAccessSettingsFromSDK(response)
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
	cmd.Short = `List private access settings.`
	cmd.Long = `List private access settings.

  Lists Databricks private access settings for an account.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq := &sdknetworking.ListPrivateAccessSettingsRequest{}

		response, err := a.Networking.ListPrivateAccessSettingsPublic(ctx, *sdkReq)
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

// start replace command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var replaceOverrides []func(
	*cobra.Command,
	*networking.UpdatePrivateAccessSettingsRequest,
)

func newReplace() *cobra.Command {
	cmd := &cobra.Command{}
	var replaceReq networking.UpdatePrivateAccessSettingsRequest
	replaceReq.CustomerFacingPrivateAccessSettings = &networking.PrivateAccessSettings{}

	var replaceJson flags.JsonFlag

	cmd.Flags().Var(&replaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: allowed_vpc_endpoint_ids
	flags.EnumVar(cmd.Flags(), &replaceReq.CustomerFacingPrivateAccessSettings.PrivateAccessLevel, "private-access-level", `The private access level controls which VPC endpoints can connect to the UI or API of any workspace that attaches this private access settings object. Supported values: [ACCOUNT, ENDPOINT]`, "PrivateAccessLevel", "ACCOUNT", "ENDPOINT")
	flags.StringPtrVar(cmd.Flags(), &replaceReq.CustomerFacingPrivateAccessSettings.PrivateAccessSettingsName, "private-access-settings-name", `The human-readable name of the private access settings object.`)
	flags.BoolPtrVar(cmd.Flags(), &replaceReq.CustomerFacingPrivateAccessSettings.PublicAccessEnabled, "public-access-enabled", `Determines if the workspace can be accessed over public internet.`)
	flags.StringPtrVar(cmd.Flags(), &replaceReq.CustomerFacingPrivateAccessSettings.Region, "region", `The AWS region for workspaces attached to this private access settings object.`)

	cmd.Use = "replace PRIVATE_ACCESS_SETTINGS_ID"
	cmd.Short = `Update private access settings.`
	cmd.Long = `Update private access settings.

  Updates an existing private access settings object, which specifies how your
  workspace is accessed over AWS PrivateLink. To use AWS PrivateLink, a
  workspace must have a private access settings object referenced by ID in the
  workspace's private_access_settings_id property. This operation completely
  overwrites your existing private access settings object attached to your
  workspaces. All workspaces attached to the private access settings are
  affected by any change. If public_access_enabled, private_access_level, or
  allowed_vpc_endpoint_ids are updated, effects of these changes might take
  several minutes to propagate to the workspace API. You can share one private
  access settings object with multiple workspaces in a single account. However,
  private access settings are specific to AWS regions, so only workspaces in the
  same AWS region can use a given private access settings object. Before
  configuring PrivateLink, read the Databricks article about PrivateLink.

  Arguments:
    PRIVATE_ACCESS_SETTINGS_ID: Databricks private access settings ID.`

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

		if cmd.Flags().Changed("json") {
			diags := replaceJson.Unmarshal(&replaceReq.CustomerFacingPrivateAccessSettings)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		replaceReq.CustomerFacingPrivateAccessSettings.PrivateAccessSettingsId = new(args[0])

		sdkReq, err := networking.UpdatePrivateAccessSettingsRequestToSDK(&replaceReq)
		if err != nil {
			return err
		}
		response, err := a.Networking.UpdatePrivateAccessSettingsPublic(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.PrivateAccessSettingsFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range replaceOverrides {
		fn(cmd, &replaceReq)
	}

	return cmd
}

// end service PrivateAccess
