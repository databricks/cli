// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package private_access

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "private-access",
	Short: `These APIs manage private access settings for this account.`,
	Long:  `These APIs manage private access settings for this account.`,
	Annotations: map[string]string{
		"package": "provisioning",
	},
}

// start create command
var createReq provisioning.UpsertPrivateAccessSettingsRequest
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: allowed_vpc_endpoint_ids
	createCmd.Flags().Var(&createReq.PrivateAccessLevel, "private-access-level", `The private access level controls which VPC endpoints can connect to the UI or API of any workspace that attaches this private access settings object.`)
	createCmd.Flags().BoolVar(&createReq.PublicAccessEnabled, "public-access-enabled", createReq.PublicAccessEnabled, `Determines if the workspace can be accessed over public internet.`)

}

var createCmd = &cobra.Command{
	Use:   "create PRIVATE_ACCESS_SETTINGS_NAME REGION",
	Short: `Create private access settings.`,
	Long: `Create private access settings.
  
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
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
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
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.PrivateAccessSettingsName = args[0]
			createReq.Region = args[1]
		}

		response, err := a.PrivateAccess.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start delete command
var deleteReq provisioning.DeletePrivateAccesRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete PRIVATE_ACCESS_SETTINGS_ID",
	Short: `Delete a private access settings object.`,
	Long: `Delete a private access settings object.
  
  Deletes a private access settings object, which determines how your workspace
  is accessed over [AWS PrivateLink].
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

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
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command
var getReq provisioning.GetPrivateAccesRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get PRIVATE_ACCESS_SETTINGS_ID",
	Short: `Get a private access settings object.`,
	Long: `Get a private access settings object.
  
  Gets a private access settings object, which specifies how your workspace is
  accessed over [AWS PrivateLink].
  
  Before configuring PrivateLink, read the [Databricks article about
  PrivateLink].
  
  [AWS PrivateLink]: https://aws.amazon.com/privatelink
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

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
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all private access settings objects.`,
	Long: `Get all private access settings objects.
  
  Gets a list of all private access settings objects for an account, specified
  by ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.PrivateAccess.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start replace command
var replaceReq provisioning.UpsertPrivateAccessSettingsRequest
var replaceJson flags.JsonFlag

func init() {
	Cmd.AddCommand(replaceCmd)
	// TODO: short flags
	replaceCmd.Flags().Var(&replaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: allowed_vpc_endpoint_ids
	replaceCmd.Flags().Var(&replaceReq.PrivateAccessLevel, "private-access-level", `The private access level controls which VPC endpoints can connect to the UI or API of any workspace that attaches this private access settings object.`)
	replaceCmd.Flags().BoolVar(&replaceReq.PublicAccessEnabled, "public-access-enabled", replaceReq.PublicAccessEnabled, `Determines if the workspace can be accessed over public internet.`)

}

var replaceCmd = &cobra.Command{
	Use:   "replace PRIVATE_ACCESS_SETTINGS_NAME REGION PRIVATE_ACCESS_SETTINGS_ID",
	Short: `Replace private access settings.`,
	Long: `Replace private access settings.
  
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
  [Databricks article about PrivateLink]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(3)
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = replaceJson.Unmarshal(&replaceReq)
			if err != nil {
				return err
			}
		}
		replaceReq.PrivateAccessSettingsName = args[0]
		replaceReq.Region = args[1]
		replaceReq.PrivateAccessSettingsId = args[2]

		err = a.PrivateAccess.Replace(ctx, replaceReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service PrivateAccess
