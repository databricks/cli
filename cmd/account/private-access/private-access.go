// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package private_access

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "provisioning",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*provisioning.UpsertPrivateAccessSettingsRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq provisioning.UpsertPrivateAccessSettingsRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: allowed_vpc_endpoint_ids
	cmd.Flags().Var(&createReq.PrivateAccessLevel, "private-access-level", `Supported values: [ACCOUNT, ENDPOINT]`)
	cmd.Flags().BoolVar(&createReq.PublicAccessEnabled, "public-access-enabled", createReq.PublicAccessEnabled, `Determines if the workspace can be accessed over public internet.`)

	cmd.Use = "create PRIVATE_ACCESS_SETTINGS_NAME REGION"
	cmd.Short = `Create private access settings.`
	cmd.Long = `Create private access settings.
  
  Creates a private access settings object, which specifies how your workspace
  is accessed over [AWS PrivateLink]. To use AWS PrivateLink, a workspace must
  have a private access settings object referenced by ID in the workspace's
  private_access_settings_id property.
  
  You can share one private access settings with multiple workspaces in a single
  account. However, private access settings are specific to AWS regions, so only
  workspaces in the same AWS region can use a given private access settings
  object.
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html

  Arguments:
    PRIVATE_ACCESS_SETTINGS_NAME: The human-readable name of the private access settings object.
    REGION: The cloud region for workspaces associated with this private access
      settings object.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'private_access_settings_name', 'region' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.PrivateAccessSettingsName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createReq.Region = args[1]
		}

		response, err := a.PrivateAccess.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*provisioning.DeletePrivateAccesRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq provisioning.DeletePrivateAccesRequest

	cmd.Use = "delete PRIVATE_ACCESS_SETTINGS_ID"
	cmd.Short = `Delete a private access settings object.`
	cmd.Long = `Delete a private access settings object.
  
  Deletes a private access settings object, which determines how your workspace
  is accessed over [AWS PrivateLink].
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].",
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html

  Arguments:
    PRIVATE_ACCESS_SETTINGS_ID: Databricks Account API private access settings ID.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PRIVATE_ACCESS_SETTINGS_ID argument specified. Loading names for Private Access drop-down."
			names, err := a.PrivateAccess.PrivateAccessSettingsPrivateAccessSettingsNameToPrivateAccessSettingsIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Private Access drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Databricks Account API private access settings ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks account api private access settings id")
		}
		deleteReq.PrivateAccessSettingsId = args[0]

		err = a.PrivateAccess.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
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
	*provisioning.GetPrivateAccesRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq provisioning.GetPrivateAccesRequest

	cmd.Use = "get PRIVATE_ACCESS_SETTINGS_ID"
	cmd.Short = `Get a private access settings object.`
	cmd.Long = `Get a private access settings object.
  
  Gets a private access settings object, which specifies how your workspace is
  accessed over [AWS PrivateLink].
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].",
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html

  Arguments:
    PRIVATE_ACCESS_SETTINGS_ID: Databricks Account API private access settings ID.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PRIVATE_ACCESS_SETTINGS_ID argument specified. Loading names for Private Access drop-down."
			names, err := a.PrivateAccess.PrivateAccessSettingsPrivateAccessSettingsNameToPrivateAccessSettingsIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Private Access drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Databricks Account API private access settings ID")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have databricks account api private access settings id")
		}
		getReq.PrivateAccessSettingsId = args[0]

		response, err := a.PrivateAccess.Get(ctx, getReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `Get all private access settings objects.`
	cmd.Long = `Get all private access settings objects.
  
  Gets a list of all private access settings objects for an account, specified
  by ID.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		response, err := a.PrivateAccess.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*provisioning.UpsertPrivateAccessSettingsRequest,
)

func newReplace() *cobra.Command {
	cmd := &cobra.Command{}

	var replaceReq provisioning.UpsertPrivateAccessSettingsRequest
	var replaceJson flags.JsonFlag

	cmd.Flags().Var(&replaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: allowed_vpc_endpoint_ids
	cmd.Flags().Var(&replaceReq.PrivateAccessLevel, "private-access-level", `Supported values: [ACCOUNT, ENDPOINT]`)
	cmd.Flags().BoolVar(&replaceReq.PublicAccessEnabled, "public-access-enabled", replaceReq.PublicAccessEnabled, `Determines if the workspace can be accessed over public internet.`)

	cmd.Use = "replace PRIVATE_ACCESS_SETTINGS_ID PRIVATE_ACCESS_SETTINGS_NAME REGION"
	cmd.Short = `Replace private access settings.`
	cmd.Long = `Replace private access settings.
  
  Updates an existing private access settings object, which specifies how your
  workspace is accessed over [AWS PrivateLink]. To use AWS PrivateLink, a
  workspace must have a private access settings object referenced by ID in the
  workspace's private_access_settings_id property.
  
  This operation completely overwrites your existing private access settings
  object attached to your workspaces. All workspaces attached to the private
  access settings are affected by any change. If public_access_enabled,
  private_access_level, or allowed_vpc_endpoint_ids are updated, effects of
  these changes might take several minutes to propagate to the workspace API.
  
  You can share one private access settings object with multiple workspaces in a
  single account. However, private access settings are specific to AWS regions,
  so only workspaces in the same AWS region can use a given private access
  settings object.
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html

  Arguments:
    PRIVATE_ACCESS_SETTINGS_ID: Databricks Account API private access settings ID.
    PRIVATE_ACCESS_SETTINGS_NAME: The human-readable name of the private access settings object.
    REGION: The cloud region for workspaces associated with this private access
      settings object.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PRIVATE_ACCESS_SETTINGS_ID as positional arguments. Provide 'private_access_settings_name', 'region' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := replaceJson.Unmarshal(&replaceReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		replaceReq.PrivateAccessSettingsId = args[0]
		if !cmd.Flags().Changed("json") {
			replaceReq.PrivateAccessSettingsName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			replaceReq.Region = args[2]
		}

		err = a.PrivateAccess.Replace(ctx, replaceReq)
		if err != nil {
			return err
		}
		return nil
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
