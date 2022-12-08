package cluster_policies

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/clusterpolicies"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "cluster-policies",
	Short: `Cluster policy limits the ability to configure clusters based on a set of rules.`,
}

var createReq clusterpolicies.CreatePolicy

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Definition, "definition", "", `Policy definition document expressed in Databricks Cluster Policy Definition Language.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `Cluster Policy name requested by the user.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new policy.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.ClusterPolicies.Create(ctx, createReq)
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

var deleteReq clusterpolicies.DeletePolicy

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.PolicyId, "policy-id", "", `The ID of the policy to delete.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a cluster policy.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.ClusterPolicies.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var editReq clusterpolicies.EditPolicy

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags

	editCmd.Flags().StringVar(&editReq.Definition, "definition", "", `Policy definition document expressed in Databricks Cluster Policy Definition Language.`)
	editCmd.Flags().StringVar(&editReq.Name, "name", "", `Cluster Policy name requested by the user.`)
	editCmd.Flags().StringVar(&editReq.PolicyId, "policy-id", "", `The ID of the policy to update.`)

}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: `Update a cluster policy.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.ClusterPolicies.Edit(ctx, editReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq clusterpolicies.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.PolicyId, "policy-id", "", `Canonical unique identifier for the cluster policy.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get entity.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.ClusterPolicies.Get(ctx, getReq)
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

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get a cluster policy.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.ClusterPolicies.ListAll(ctx)
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
