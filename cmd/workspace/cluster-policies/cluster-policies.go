// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package cluster_policies

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
		Use:   "cluster-policies",
		Short: `You can use cluster policies to control users' ability to configure clusters based on a set of rules.`,
		Long: `You can use cluster policies to control users' ability to configure clusters
  based on a set of rules. These rules specify which attributes or attribute
  values can be used during cluster creation. Cluster policies have ACLs that
  limit their use to specific users and groups.
  
  With cluster policies, you can: - Auto-install cluster libraries on the next
  restart by listing them in the policy's "libraries" field (Public Preview). -
  Limit users to creating clusters with the prescribed settings. - Simplify the
  user interface, enabling more users to create clusters, by fixing and hiding
  some fields. - Manage costs by setting limits on attributes that impact the
  hourly rate.
  
  Cluster policy permissions limit which policies a user can select in the
  Policy drop-down when the user creates a cluster: - A user who has
  unrestricted cluster create permission can select the Unrestricted policy and
  create fully-configurable clusters. - A user who has both unrestricted cluster
  create permission and access to cluster policies can select the Unrestricted
  policy and policies they have access to. - A user that has access to only
  cluster policies, can select the policies they have access to.
  
  If no policies exist in the workspace, the Policy drop-down doesn't appear.
  Only admin users can create, edit, and delete policies. Admin users also have
  access to all policies.`,
		GroupID: "compute",
		Annotations: map[string]string{
			"package": "compute",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newEdit())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newList())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newUpdatePermissions())

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
	*compute.CreatePolicy,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq compute.CreatePolicy
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Definition, "definition", createReq.Definition, `Policy definition document expressed in [Databricks Cluster Policy Definition Language](https://docs.databricks.com/administration-guide/clusters/policy-definition.html).`)
	cmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `Additional human-readable description of the cluster policy.`)
	// TODO: array: libraries
	cmd.Flags().Int64Var(&createReq.MaxClustersPerUser, "max-clusters-per-user", createReq.MaxClustersPerUser, `Max number of clusters per user that can be active using this policy.`)
	cmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `Cluster Policy name requested by the user.`)
	cmd.Flags().StringVar(&createReq.PolicyFamilyDefinitionOverrides, "policy-family-definition-overrides", createReq.PolicyFamilyDefinitionOverrides, `Policy definition JSON document expressed in [Databricks Policy Definition Language](https://docs.databricks.com/administration-guide/clusters/policy-definition.html).`)
	cmd.Flags().StringVar(&createReq.PolicyFamilyId, "policy-family-id", createReq.PolicyFamilyId, `ID of the policy family.`)

	cmd.Use = "create"
	cmd.Short = `Create a new policy.`
	cmd.Long = `Create a new policy.
  
  Creates a new policy with prescribed settings.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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

	cmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete POLICY_ID"
	cmd.Short = `Delete a cluster policy.`
	cmd.Long = `Delete a cluster policy.
  
  Delete a policy for a cluster. Clusters governed by this policy can still run,
  but cannot be edited.

  Arguments:
    POLICY_ID: The ID of the policy to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'policy_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := deleteJson.Unmarshal(&deleteReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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

	cmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&editReq.Definition, "definition", editReq.Definition, `Policy definition document expressed in [Databricks Cluster Policy Definition Language](https://docs.databricks.com/administration-guide/clusters/policy-definition.html).`)
	cmd.Flags().StringVar(&editReq.Description, "description", editReq.Description, `Additional human-readable description of the cluster policy.`)
	// TODO: array: libraries
	cmd.Flags().Int64Var(&editReq.MaxClustersPerUser, "max-clusters-per-user", editReq.MaxClustersPerUser, `Max number of clusters per user that can be active using this policy.`)
	cmd.Flags().StringVar(&editReq.Name, "name", editReq.Name, `Cluster Policy name requested by the user.`)
	cmd.Flags().StringVar(&editReq.PolicyFamilyDefinitionOverrides, "policy-family-definition-overrides", editReq.PolicyFamilyDefinitionOverrides, `Policy definition JSON document expressed in [Databricks Policy Definition Language](https://docs.databricks.com/administration-guide/clusters/policy-definition.html).`)
	cmd.Flags().StringVar(&editReq.PolicyFamilyId, "policy-family-id", editReq.PolicyFamilyId, `ID of the policy family.`)

	cmd.Use = "edit POLICY_ID"
	cmd.Short = `Update a cluster policy.`
	cmd.Long = `Update a cluster policy.
  
  Update an existing policy for cluster. This operation may make some clusters
  governed by the previous policy invalid.

  Arguments:
    POLICY_ID: The ID of the policy to update.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'policy_id' in your JSON input")
			}
			return nil
		}
		return nil
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
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No POLICY_ID argument specified. Loading names for Cluster Policies drop-down."
				names, err := w.ClusterPolicies.PolicyNameToPolicyIdMap(ctx, compute.ListClusterPoliciesRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Cluster Policies drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID of the policy to update")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the id of the policy to update")
			}
			editReq.PolicyId = args[0]
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

	cmd.Use = "get POLICY_ID"
	cmd.Short = `Get a cluster policy.`
	cmd.Long = `Get a cluster policy.
  
  Get a cluster policy entity. Creation and editing is available to admins only.

  Arguments:
    POLICY_ID: Canonical unique identifier for the Cluster Policy.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No POLICY_ID argument specified. Loading names for Cluster Policies drop-down."
			names, err := w.ClusterPolicies.PolicyNameToPolicyIdMap(ctx, compute.ListClusterPoliciesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Cluster Policies drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Canonical unique identifier for the Cluster Policy")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have canonical unique identifier for the cluster policy")
		}
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

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*compute.GetClusterPolicyPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq compute.GetClusterPolicyPermissionLevelsRequest

	cmd.Use = "get-permission-levels CLUSTER_POLICY_ID"
	cmd.Short = `Get cluster policy permission levels.`
	cmd.Long = `Get cluster policy permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    CLUSTER_POLICY_ID: The cluster policy for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No CLUSTER_POLICY_ID argument specified. Loading names for Cluster Policies drop-down."
			names, err := w.ClusterPolicies.PolicyNameToPolicyIdMap(ctx, compute.ListClusterPoliciesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Cluster Policies drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster policy for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster policy for which to get or manage permissions")
		}
		getPermissionLevelsReq.ClusterPolicyId = args[0]

		response, err := w.ClusterPolicies.GetPermissionLevels(ctx, getPermissionLevelsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionLevelsOverrides {
		fn(cmd, &getPermissionLevelsReq)
	}

	return cmd
}

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*compute.GetClusterPolicyPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq compute.GetClusterPolicyPermissionsRequest

	cmd.Use = "get-permissions CLUSTER_POLICY_ID"
	cmd.Short = `Get cluster policy permissions.`
	cmd.Long = `Get cluster policy permissions.
  
  Gets the permissions of a cluster policy. Cluster policies can inherit
  permissions from their root object.

  Arguments:
    CLUSTER_POLICY_ID: The cluster policy for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No CLUSTER_POLICY_ID argument specified. Loading names for Cluster Policies drop-down."
			names, err := w.ClusterPolicies.PolicyNameToPolicyIdMap(ctx, compute.ListClusterPoliciesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Cluster Policies drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster policy for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster policy for which to get or manage permissions")
		}
		getPermissionsReq.ClusterPolicyId = args[0]

		response, err := w.ClusterPolicies.GetPermissions(ctx, getPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionsOverrides {
		fn(cmd, &getPermissionsReq)
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

	cmd.Flags().Var(&listReq.SortColumn, "sort-column", `The cluster policy attribute to sort by. Supported values: [POLICY_CREATION_TIME, POLICY_NAME]`)
	cmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order in which the policies get listed. Supported values: [ASC, DESC]`)

	cmd.Use = "list"
	cmd.Short = `List cluster policies.`
	cmd.Long = `List cluster policies.
  
  Returns a list of policies accessible by the requesting user.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ClusterPolicies.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*compute.ClusterPolicyPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq compute.ClusterPolicyPermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions CLUSTER_POLICY_ID"
	cmd.Short = `Set cluster policy permissions.`
	cmd.Long = `Set cluster policy permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    CLUSTER_POLICY_ID: The cluster policy for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No CLUSTER_POLICY_ID argument specified. Loading names for Cluster Policies drop-down."
			names, err := w.ClusterPolicies.PolicyNameToPolicyIdMap(ctx, compute.ListClusterPoliciesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Cluster Policies drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster policy for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster policy for which to get or manage permissions")
		}
		setPermissionsReq.ClusterPolicyId = args[0]

		response, err := w.ClusterPolicies.SetPermissions(ctx, setPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setPermissionsOverrides {
		fn(cmd, &setPermissionsReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*compute.ClusterPolicyPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq compute.ClusterPolicyPermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions CLUSTER_POLICY_ID"
	cmd.Short = `Update cluster policy permissions.`
	cmd.Long = `Update cluster policy permissions.
  
  Updates the permissions on a cluster policy. Cluster policies can inherit
  permissions from their root object.

  Arguments:
    CLUSTER_POLICY_ID: The cluster policy for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No CLUSTER_POLICY_ID argument specified. Loading names for Cluster Policies drop-down."
			names, err := w.ClusterPolicies.PolicyNameToPolicyIdMap(ctx, compute.ListClusterPoliciesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Cluster Policies drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster policy for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster policy for which to get or manage permissions")
		}
		updatePermissionsReq.ClusterPolicyId = args[0]

		response, err := w.ClusterPolicies.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
	}

	return cmd
}

// end service ClusterPolicies
