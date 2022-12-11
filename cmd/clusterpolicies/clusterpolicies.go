// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clusterpolicies

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/clusterpolicies"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "cluster-policies",
	Short: `Cluster policy limits the ability to configure clusters based on a set of rules.`,
	Long: `Cluster policy limits the ability to configure clusters based on a set of
  rules. The policy rules limit the attributes or attribute values available for
  cluster creation. Cluster policies have ACLs that limit their use to specific
  users and groups.
  
  Cluster policies let you limit users to create clusters with prescribed
  settings, simplify the user interface and enable more users to create their
  own clusters (by fixing and hiding some values), control cost by limiting per
  cluster maximum cost (by setting limits on attributes whose values contribute
  to hourly price).
  
  Cluster policy permissions limit which policies a user can select in the
  Policy drop-down when the user creates a cluster: - A user who has cluster
  create permission can select the Unrestricted policy and create
  fully-configurable clusters. - A user who has both cluster create permission
  and access to cluster policies can select the Unrestricted policy and policies
  they have access to. - A user that has access to only cluster policies, can
  select the policies they have access to.
  
  If no policies have been created in the workspace, the Policy drop-down does
  not display.
  
  Only admin users can create, edit, and delete policies. Admin users also have
  access to all policies.`,
}

// start create command

var createReq clusterpolicies.CreatePolicy

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

}

var createCmd = &cobra.Command{
	Use:   "create NAME DEFINITION",
	Short: `Create a new policy.`,
	Long: `Create a new policy.
  
  Creates a new policy with prescribed settings.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		createReq.Name = args[0]
		createReq.Definition = args[1]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.ClusterPolicies.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq clusterpolicies.DeletePolicy

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete POLICY_ID",
	Short: `Delete a cluster policy.`,
	Long: `Delete a cluster policy.
  
  Delete a policy for a cluster. Clusters governed by this policy can still run,
  but cannot be edited.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		deleteReq.PolicyId = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.ClusterPolicies.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start edit command

var editReq clusterpolicies.EditPolicy

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags

}

var editCmd = &cobra.Command{
	Use:   "edit POLICY_ID NAME DEFINITION",
	Short: `Update a cluster policy.`,
	Long: `Update a cluster policy.
  
  Update an existing policy for cluster. This operation may make some clusters
  governed by the previous policy invalid.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		editReq.PolicyId = args[0]
		editReq.Name = args[1]
		editReq.Definition = args[2]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.ClusterPolicies.Edit(ctx, editReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq clusterpolicies.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get POLICY_ID",
	Short: `Get entity.`,
	Long: `Get entity.
  
  Get a cluster policy entity. Creation and editing is available to admins only.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		getReq.PolicyId = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.ClusterPolicies.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get a cluster policy.`,
	Long: `Get a cluster policy.
  
  Returns a list of policies accessible by the requesting user.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.ClusterPolicies.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service ClusterPolicies

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
