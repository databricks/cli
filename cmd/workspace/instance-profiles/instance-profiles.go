// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package instance_profiles

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "instance-profiles",
	Short: `The Instance Profiles API allows admins to add, list, and remove instance profiles that users can launch clusters with.`,
	Long: `The Instance Profiles API allows admins to add, list, and remove instance
  profiles that users can launch clusters with. Regular users can list the
  instance profiles available to them. See [Secure access to S3 buckets] using
  instance profiles for more information.
  
  [Secure access to S3 buckets]: https://docs.databricks.com/administration-guide/cloud-configurations/aws/instance-profiles.html`,
}

// start add command

var addReq compute.AddInstanceProfile
var addJson flags.JsonFlag

func init() {
	Cmd.AddCommand(addCmd)
	// TODO: short flags
	addCmd.Flags().Var(&addJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	addCmd.Flags().StringVar(&addReq.IamRoleArn, "iam-role-arn", addReq.IamRoleArn, `The AWS IAM role ARN of the role associated with the instance profile.`)
	addCmd.Flags().BoolVar(&addReq.IsMetaInstanceProfile, "is-meta-instance-profile", addReq.IsMetaInstanceProfile, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)
	addCmd.Flags().BoolVar(&addReq.SkipValidation, "skip-validation", addReq.SkipValidation, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)

}

var addCmd = &cobra.Command{
	Use:   "add INSTANCE_PROFILE_ARN",
	Short: `Register an instance profile.`,
	Long: `Register an instance profile.
  
  In the UI, you can select the instance profile when launching clusters. This
  API is only available to admin users.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = addJson.Unmarshal(&addReq)
			if err != nil {
				return err
			}
		} else {
			addReq.InstanceProfileArn = args[0]
		}

		err = w.InstanceProfiles.Add(ctx, addReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start edit command

var editReq compute.InstanceProfile
var editJson flags.JsonFlag

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags
	editCmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	editCmd.Flags().StringVar(&editReq.IamRoleArn, "iam-role-arn", editReq.IamRoleArn, `The AWS IAM role ARN of the role associated with the instance profile.`)
	editCmd.Flags().BoolVar(&editReq.IsMetaInstanceProfile, "is-meta-instance-profile", editReq.IsMetaInstanceProfile, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)

}

var editCmd = &cobra.Command{
	Use:   "edit INSTANCE_PROFILE_ARN",
	Short: `Edit an instance profile.`,
	Long: `Edit an instance profile.
  
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
  [Enable serverless SQL warehouses]: https://docs.databricks.com/sql/admin/serverless.html`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = editJson.Unmarshal(&editReq)
			if err != nil {
				return err
			}
		} else {
			editReq.InstanceProfileArn = args[0]
		}

		err = w.InstanceProfiles.Edit(ctx, editReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List available instance profiles.`,
	Long: `List available instance profiles.
  
  List the instance profiles that the calling user can use to launch a cluster.
  
  This API is available to all users.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.InstanceProfiles.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start remove command

var removeReq compute.RemoveInstanceProfile
var removeJson flags.JsonFlag

func init() {
	Cmd.AddCommand(removeCmd)
	// TODO: short flags
	removeCmd.Flags().Var(&removeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var removeCmd = &cobra.Command{
	Use:   "remove INSTANCE_PROFILE_ARN",
	Short: `Remove the instance profile.`,
	Long: `Remove the instance profile.
  
  Remove the instance profile with the provided ARN. Existing clusters with this
  instance profile will continue to function.
  
  This API is only accessible to admin users.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = removeJson.Unmarshal(&removeReq)
			if err != nil {
				return err
			}
		} else {
			removeReq.InstanceProfileArn = args[0]
		}

		err = w.InstanceProfiles.Remove(ctx, removeReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service InstanceProfiles
