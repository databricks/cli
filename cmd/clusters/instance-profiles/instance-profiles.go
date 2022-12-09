package instance_profiles

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/clusters"
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

var addReq clusters.AddInstanceProfile

func init() {
	Cmd.AddCommand(addCmd)
	// TODO: short flags

	addCmd.Flags().StringVar(&addReq.IamRoleArn, "iam-role-arn", addReq.IamRoleArn, `The AWS IAM role ARN of the role associated with the instance profile.`)
	addCmd.Flags().StringVar(&addReq.InstanceProfileArn, "instance-profile-arn", addReq.InstanceProfileArn, `The AWS ARN of the instance profile to register with Databricks.`)
	addCmd.Flags().BoolVar(&addReq.IsMetaInstanceProfile, "is-meta-instance-profile", addReq.IsMetaInstanceProfile, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)
	addCmd.Flags().BoolVar(&addReq.SkipValidation, "skip-validation", addReq.SkipValidation, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)

}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: `Register an instance profile.`,
	Long: `Register an instance profile.
  
  In the UI, you can select the instance profile when launching clusters. This
  API is only available to admin users.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.InstanceProfiles.Add(ctx, addReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start edit command

var editReq clusters.InstanceProfile

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags

	editCmd.Flags().StringVar(&editReq.IamRoleArn, "iam-role-arn", editReq.IamRoleArn, `The AWS IAM role ARN of the role associated with the instance profile.`)
	editCmd.Flags().StringVar(&editReq.InstanceProfileArn, "instance-profile-arn", editReq.InstanceProfileArn, `The AWS ARN of the instance profile to register with Databricks.`)
	editCmd.Flags().BoolVar(&editReq.IsMetaInstanceProfile, "is-meta-instance-profile", editReq.IsMetaInstanceProfile, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)

}

var editCmd = &cobra.Command{
	Use:   "edit",
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

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.InstanceProfiles.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start remove command

var removeReq clusters.RemoveInstanceProfile

func init() {
	Cmd.AddCommand(removeCmd)
	// TODO: short flags

	removeCmd.Flags().StringVar(&removeReq.InstanceProfileArn, "instance-profile-arn", removeReq.InstanceProfileArn, `The ARN of the instance profile to remove.`)

}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: `Remove the instance profile.`,
	Long: `Remove the instance profile.
  
  Remove the instance profile with the provided ARN. Existing clusters with this
  instance profile will continue to function.
  
  This API is only accessible to admin users.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.InstanceProfiles.Remove(ctx, removeReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service InstanceProfiles
