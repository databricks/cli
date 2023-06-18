// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package cluster_policies

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/compute"
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
	Annotations: map[string]string{
		"package": "compute",
	},
}

// start create command

var createReq compute.CreatePolicy
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Definition, "definition", createReq.Definition, `Policy definition document expressed in Databricks Cluster Policy Definition Language.`)
	createCmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `Additional human-readable description of the cluster policy.`)
	createCmd.Flags().Int64Var(&createReq.MaxClustersPerUser, "max-clusters-per-user", createReq.MaxClustersPerUser, `Max number of clusters per user that can be active using this policy.`)
	createCmd.Flags().StringVar(&createReq.PolicyFamilyDefinitionOverrides, "policy-family-definition-overrides", createReq.PolicyFamilyDefinitionOverrides, `Policy definition JSON document expressed in Databricks Policy Definition Language.`)
	createCmd.Flags().StringVar(&createReq.PolicyFamilyId, "policy-family-id", createReq.PolicyFamilyId, `ID of the policy family.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME",
	Short: `Create a new policy.`,
	Long: `Create a new policy.
  
  Creates a new policy with prescribed settings.`,

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
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.Name = args[0]
		}

		response, err := w.ClusterPolicies.Create(ctx, createReq)
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

var deleteReq compute.DeletePolicy
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete POLICY_ID",
	Short: `Delete a cluster policy.`,
	Long: `Delete a cluster policy.
  
  Delete a policy for a cluster. Clusters governed by this policy can still run,
  but cannot be edited.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No POLICY_ID argument specified. Loading names for Cluster Policies drop-down."
				names, err := w.ClusterPolicies.PolicyNameToPolicyIdMap(ctx, compute.ListClusterPoliciesRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Cluster Policies drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID of the policy to delete")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the id of the policy to delete")
			}
			deleteReq.PolicyId = args[0]
		}

		err = w.ClusterPolicies.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start edit command

var editReq compute.EditPolicy
var editJson flags.JsonFlag

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags
	editCmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	editCmd.Flags().StringVar(&editReq.Definition, "definition", editReq.Definition, `Policy definition document expressed in Databricks Cluster Policy Definition Language.`)
	editCmd.Flags().StringVar(&editReq.Description, "description", editReq.Description, `Additional human-readable description of the cluster policy.`)
	editCmd.Flags().Int64Var(&editReq.MaxClustersPerUser, "max-clusters-per-user", editReq.MaxClustersPerUser, `Max number of clusters per user that can be active using this policy.`)
	editCmd.Flags().StringVar(&editReq.PolicyFamilyDefinitionOverrides, "policy-family-definition-overrides", editReq.PolicyFamilyDefinitionOverrides, `Policy definition JSON document expressed in Databricks Policy Definition Language.`)
	editCmd.Flags().StringVar(&editReq.PolicyFamilyId, "policy-family-id", editReq.PolicyFamilyId, `ID of the policy family.`)

}

var editCmd = &cobra.Command{
	Use:   "edit POLICY_ID NAME",
	Short: `Update a cluster policy.`,
	Long: `Update a cluster policy.
  
  Update an existing policy for cluster. This operation may make some clusters
  governed by the previous policy invalid.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
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
			editReq.PolicyId = args[0]
			editReq.Name = args[1]
		}

		err = w.ClusterPolicies.Edit(ctx, editReq)
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

var getReq compute.GetClusterPolicyRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get POLICY_ID",
	Short: `Get entity.`,
	Long: `Get entity.
  
  Get a cluster policy entity. Creation and editing is available to admins only.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No POLICY_ID argument specified. Loading names for Cluster Policies drop-down."
				names, err := w.ClusterPolicies.PolicyNameToPolicyIdMap(ctx, compute.ListClusterPoliciesRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Cluster Policies drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Canonical unique identifier for the cluster policy")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have canonical unique identifier for the cluster policy")
			}
			getReq.PolicyId = args[0]
		}

		response, err := w.ClusterPolicies.Get(ctx, getReq)
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

var listReq compute.ListClusterPoliciesRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().Var(&listReq.SortColumn, "sort-column", `The cluster policy attribute to sort by.`)
	listCmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order in which the policies get listed.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get a cluster policy.`,
	Long: `Get a cluster policy.
  
  Returns a list of policies accessible by the requesting user.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
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
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.ClusterPolicies.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service ClusterPolicies
