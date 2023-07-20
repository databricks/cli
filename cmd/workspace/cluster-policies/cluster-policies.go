// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package cluster_policies

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
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
		GroupID: "compute",
		Annotations: map[string]string{
			"package": "compute",
		},
	}

	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newEdit())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*compute.CreatePolicy,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq compute.CreatePolicy
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Definition, "definition", createReq.Definition, `Policy definition document expressed in Databricks Cluster Policy Definition Language.`)
	cmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `Additional human-readable description of the cluster policy.`)
	cmd.Flags().Int64Var(&createReq.MaxClustersPerUser, "max-clusters-per-user", createReq.MaxClustersPerUser, `Max number of clusters per user that can be active using this policy.`)
	cmd.Flags().StringVar(&createReq.PolicyFamilyDefinitionOverrides, "policy-family-definition-overrides", createReq.PolicyFamilyDefinitionOverrides, `Policy definition JSON document expressed in Databricks Policy Definition Language.`)
	cmd.Flags().StringVar(&createReq.PolicyFamilyId, "policy-family-id", createReq.PolicyFamilyId, `ID of the policy family.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create a new policy.`
	cmd.Long = `Create a new policy.
  
  Creates a new policy with prescribed settings.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
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
	*compute.DeletePolicy,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq compute.DeletePolicy
	var deleteJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete POLICY_ID"
	cmd.Short = `Delete a cluster policy.`
	cmd.Long = `Delete a cluster policy.
  
  Delete a policy for a cluster. Clusters governed by this policy can still run,
  but cannot be edited.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			deleteReq.PolicyId = args[0]
		}

		err = w.ClusterPolicies.Delete(ctx, deleteReq)
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

// start edit command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var editOverrides []func(
	*cobra.Command,
	*compute.EditPolicy,
)

func newEdit() *cobra.Command {
	cmd := &cobra.Command{}

	var editReq compute.EditPolicy
	var editJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&editReq.Definition, "definition", editReq.Definition, `Policy definition document expressed in Databricks Cluster Policy Definition Language.`)
	cmd.Flags().StringVar(&editReq.Description, "description", editReq.Description, `Additional human-readable description of the cluster policy.`)
	cmd.Flags().Int64Var(&editReq.MaxClustersPerUser, "max-clusters-per-user", editReq.MaxClustersPerUser, `Max number of clusters per user that can be active using this policy.`)
	cmd.Flags().StringVar(&editReq.PolicyFamilyDefinitionOverrides, "policy-family-definition-overrides", editReq.PolicyFamilyDefinitionOverrides, `Policy definition JSON document expressed in Databricks Policy Definition Language.`)
	cmd.Flags().StringVar(&editReq.PolicyFamilyId, "policy-family-id", editReq.PolicyFamilyId, `ID of the policy family.`)

	cmd.Use = "edit POLICY_ID NAME"
	cmd.Short = `Update a cluster policy.`
	cmd.Long = `Update a cluster policy.
  
  Update an existing policy for cluster. This operation may make some clusters
  governed by the previous policy invalid.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*compute.GetClusterPolicyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq compute.GetClusterPolicyRequest

	// TODO: short flags

	cmd.Use = "get POLICY_ID"
	cmd.Short = `Get entity.`
	cmd.Long = `Get entity.
  
  Get a cluster policy entity. Creation and editing is available to admins only.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.PolicyId = args[0]

		response, err := w.ClusterPolicies.Get(ctx, getReq)
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
	*compute.ListClusterPoliciesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq compute.ListClusterPoliciesRequest
	var listJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&listReq.SortColumn, "sort-column", `The cluster policy attribute to sort by.`)
	cmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order in which the policies get listed.`)

	cmd.Use = "list"
	cmd.Short = `Get a cluster policy.`
	cmd.Long = `Get a cluster policy.
  
  Returns a list of policies accessible by the requesting user.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
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
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// end service ClusterPolicies
