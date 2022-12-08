package instance_profiles

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "instance-profiles",
	Short: `The Instance Profiles API allows admins to add, list, and remove instance profiles that users can launch clusters with.`,
}

var addReq clusters.AddInstanceProfile

func init() {
	Cmd.AddCommand(addCmd)
	// TODO: short flags

	addCmd.Flags().StringVar(&addReq.IamRoleArn, "iam-role-arn", "", `The AWS IAM role ARN of the role associated with the instance profile.`)
	addCmd.Flags().StringVar(&addReq.InstanceProfileArn, "instance-profile-arn", "", `The AWS ARN of the instance profile to register with Databricks.`)
	addCmd.Flags().BoolVar(&addReq.IsMetaInstanceProfile, "is-meta-instance-profile", false, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)
	addCmd.Flags().BoolVar(&addReq.SkipValidation, "skip-validation", false, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)

}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: `Register an instance profile.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.InstanceProfiles.Add(ctx, addReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var editReq clusters.InstanceProfile

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags

	editCmd.Flags().StringVar(&editReq.IamRoleArn, "iam-role-arn", "", `The AWS IAM role ARN of the role associated with the instance profile.`)
	editCmd.Flags().StringVar(&editReq.InstanceProfileArn, "instance-profile-arn", "", `The AWS ARN of the instance profile to register with Databricks.`)
	editCmd.Flags().BoolVar(&editReq.IsMetaInstanceProfile, "is-meta-instance-profile", false, `By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.`)

}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: `Edit an instance profile.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.InstanceProfiles.Edit(ctx, editReq)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List available instance profiles.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.InstanceProfiles.ListAll(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var removeReq clusters.RemoveInstanceProfile

func init() {
	Cmd.AddCommand(removeCmd)
	// TODO: short flags

	removeCmd.Flags().StringVar(&removeReq.InstanceProfileArn, "instance-profile-arn", "", `The ARN of the instance profile to remove.`)

}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: `Remove the instance profile.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.InstanceProfiles.Remove(ctx, removeReq)
		if err != nil {
			return err
		}

		return nil
	},
}
