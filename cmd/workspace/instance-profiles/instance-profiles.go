// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package instance_profiles

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance-profiles",
		Short: `The Instance Profiles API allows admins to add, list, and remove instance profiles that users can launch clusters with.`,
		Long: `The Instance Profiles API allows admins to add, list, and remove instance
  profiles that users can launch clusters with. Regular users can list the
  instance profiles available to them. See [Secure access to S3 buckets] using
  instance profiles for more information.
  
  [Secure access to S3 buckets]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/instance-profiles.html`,
		GroupID: "compute",
		Annotations: map[string]string{
			"package": "compute",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newAdd())
	cmd.AddCommand(newEdit())
	cmd.AddCommand(newList())
	cmd.AddCommand(newRemove())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start add command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var addOverrides []func(
	*cobra.Command,
	*compute.AddInstanceProfile,
)

func newAdd() *cobra.Command {
	cmd := &cobra.Command{}

	var addReq compute.AddInstanceProfile
	var addJson flags.JsonFlag

	cmd.Flags().Var(&addJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&addReq.IamRoleArn, "iam-role-arn", addReq.IamRoleArn, `The AWS IAM role ARN of the role associated with the instance profile.`)
	cmd.Flags().BoolVar(&addReq.IsMetaInstanceProfile, "is-meta-instance-profile", addReq.IsMetaInstanceProfile, `Boolean flag indicating whether the instance profile should only be used in credential passthrough scenarios.`)
	cmd.Flags().BoolVar(&addReq.SkipValidation, "skip-validation", addReq.SkipValidation, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)

	cmd.Use = "add INSTANCE_PROFILE_ARN"
	cmd.Short = `Register an instance profile.`
	cmd.Long = `Register an instance profile.
  
  Registers an instance profile in Databricks. In the UI, you can then give
  users the permission to use this instance profile when launching clusters.
  
  This API is only available to admin users.

  Arguments:
    INSTANCE_PROFILE_ARN: The AWS ARN of the instance profile to register with Databricks. This
      field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'instance_profile_arn' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := addJson.Unmarshal(&addReq)
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
			addReq.InstanceProfileArn = args[0]
		}

		err = w.InstanceProfiles.Add(ctx, addReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range addOverrides {
		fn(cmd, &addReq)
	}

	return cmd
}

// start edit command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var editOverrides []func(
	*cobra.Command,
	*compute.InstanceProfile,
)

func newEdit() *cobra.Command {
	cmd := &cobra.Command{}

	var editReq compute.InstanceProfile
	var editJson flags.JsonFlag

	cmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&editReq.IamRoleArn, "iam-role-arn", editReq.IamRoleArn, `The AWS IAM role ARN of the role associated with the instance profile.`)
	cmd.Flags().BoolVar(&editReq.IsMetaInstanceProfile, "is-meta-instance-profile", editReq.IsMetaInstanceProfile, `Boolean flag indicating whether the instance profile should only be used in credential passthrough scenarios.`)

	cmd.Use = "edit INSTANCE_PROFILE_ARN"
	cmd.Short = `Edit an instance profile.`
	cmd.Long = `Edit an instance profile.
  
  The only supported field to change is the optional IAM role ARN associated
  with the instance profile. It is required to specify the IAM role ARN if both
  of the following are true:
  
  * Your role name and instance profile name do not match. The name is the part
  after the last slash in each ARN. * You want to use the instance profile with
  [Databricks SQL Serverless].
  
  To understand where these fields are in the AWS console, see [Enable
  serverless SQL warehouses].
  
  This API is only available to admin users.
  
  [Databricks SQL Serverless]: https://docs.databricks.com/sql/admin/serverless.html
  [Enable serverless SQL warehouses]: https://docs.databricks.com/sql/admin/serverless.html

  Arguments:
    INSTANCE_PROFILE_ARN: The AWS ARN of the instance profile to register with Databricks. This
      field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'instance_profile_arn' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := editJson.Unmarshal(&editReq)
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
			editReq.InstanceProfileArn = args[0]
		}

		err = w.InstanceProfiles.Edit(ctx, editReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range editOverrides {
		fn(cmd, &editReq)
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
	cmd.Short = `List available instance profiles.`
	cmd.Long = `List available instance profiles.
  
  List the instance profiles that the calling user can use to launch a cluster.
  
  This API is available to all users.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response := w.InstanceProfiles.List(ctx)
		return cmdio.RenderIterator(ctx, response)
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

// start remove command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var removeOverrides []func(
	*cobra.Command,
	*compute.RemoveInstanceProfile,
)

func newRemove() *cobra.Command {
	cmd := &cobra.Command{}

	var removeReq compute.RemoveInstanceProfile
	var removeJson flags.JsonFlag

	cmd.Flags().Var(&removeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "remove INSTANCE_PROFILE_ARN"
	cmd.Short = `Remove the instance profile.`
	cmd.Long = `Remove the instance profile.
  
  Remove the instance profile with the provided ARN. Existing clusters with this
  instance profile will continue to function.
  
  This API is only accessible to admin users.

  Arguments:
    INSTANCE_PROFILE_ARN: The ARN of the instance profile to remove. This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'instance_profile_arn' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := removeJson.Unmarshal(&removeReq)
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
			removeReq.InstanceProfileArn = args[0]
		}

		err = w.InstanceProfiles.Remove(ctx, removeReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range removeOverrides {
		fn(cmd, &removeReq)
	}

	return cmd
}

// end service InstanceProfiles
