// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package policy_compliance_for_clusters

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
		Use:   "policy-compliance-for-clusters",
		Short: `The policy compliance APIs allow you to view and manage the policy compliance status of clusters in your workspace.`,
		Long: `The policy compliance APIs allow you to view and manage the policy compliance
  status of clusters in your workspace.
  
  A cluster is compliant with its policy if its configuration satisfies all its
  policy rules. Clusters could be out of compliance if their policy was updated
  after the cluster was last edited.
  
  The get and list compliance APIs allow you to view the policy compliance
  status of a cluster. The enforce compliance API allows you to update a cluster
  to be compliant with the current version of its policy.`,
		GroupID: "compute",
		Annotations: map[string]string{
			"package": "compute",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newEnforceCompliance())
	cmd.AddCommand(newGetCompliance())
	cmd.AddCommand(newListCompliance())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start enforce-compliance command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var enforceComplianceOverrides []func(
	*cobra.Command,
	*compute.EnforceClusterComplianceRequest,
)

func newEnforceCompliance() *cobra.Command {
	cmd := &cobra.Command{}

	var enforceComplianceReq compute.EnforceClusterComplianceRequest
	var enforceComplianceJson flags.JsonFlag

	cmd.Flags().Var(&enforceComplianceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&enforceComplianceReq.ValidateOnly, "validate-only", enforceComplianceReq.ValidateOnly, `If set, previews the changes that would be made to a cluster to enforce compliance but does not update the cluster.`)

	cmd.Use = "enforce-compliance CLUSTER_ID"
	cmd.Short = `Enforce cluster policy compliance.`
	cmd.Long = `Enforce cluster policy compliance.
  
  Updates a cluster to be compliant with the current version of its policy. A
  cluster can be updated if it is in a RUNNING or TERMINATED state.
  
  If a cluster is updated while in a RUNNING state, it will be restarted so
  that the new attributes can take effect.
  
  If a cluster is updated while in a TERMINATED state, it will remain
  TERMINATED. The next time the cluster is started, the new attributes will
  take effect.
  
  Clusters created by the Databricks Jobs, DLT, or Models services cannot be
  enforced by this API. Instead, use the "Enforce job policy compliance" API to
  enforce policy compliance on jobs.

  Arguments:
    CLUSTER_ID: The ID of the cluster you want to enforce policy compliance on.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'cluster_id' in your JSON input")
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
			diags := enforceComplianceJson.Unmarshal(&enforceComplianceReq)
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
			enforceComplianceReq.ClusterId = args[0]
		}

		response, err := w.PolicyComplianceForClusters.EnforceCompliance(ctx, enforceComplianceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range enforceComplianceOverrides {
		fn(cmd, &enforceComplianceReq)
	}

	return cmd
}

// start get-compliance command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getComplianceOverrides []func(
	*cobra.Command,
	*compute.GetClusterComplianceRequest,
)

func newGetCompliance() *cobra.Command {
	cmd := &cobra.Command{}

	var getComplianceReq compute.GetClusterComplianceRequest

	cmd.Use = "get-compliance CLUSTER_ID"
	cmd.Short = `Get cluster policy compliance.`
	cmd.Long = `Get cluster policy compliance.
  
  Returns the policy compliance status of a cluster. Clusters could be out of
  compliance if their policy was updated after the cluster was last edited.

  Arguments:
    CLUSTER_ID: The ID of the cluster to get the compliance status`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getComplianceReq.ClusterId = args[0]

		response, err := w.PolicyComplianceForClusters.GetCompliance(ctx, getComplianceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getComplianceOverrides {
		fn(cmd, &getComplianceReq)
	}

	return cmd
}

// start list-compliance command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listComplianceOverrides []func(
	*cobra.Command,
	*compute.ListClusterCompliancesRequest,
)

func newListCompliance() *cobra.Command {
	cmd := &cobra.Command{}

	var listComplianceReq compute.ListClusterCompliancesRequest

	cmd.Flags().IntVar(&listComplianceReq.PageSize, "page-size", listComplianceReq.PageSize, `Use this field to specify the maximum number of results to be returned by the server.`)
	cmd.Flags().StringVar(&listComplianceReq.PageToken, "page-token", listComplianceReq.PageToken, `A page token that can be used to navigate to the next page or previous page as returned by next_page_token or prev_page_token.`)

	cmd.Use = "list-compliance POLICY_ID"
	cmd.Short = `List cluster policy compliance.`
	cmd.Long = `List cluster policy compliance.
  
  Returns the policy compliance status of all clusters that use a given policy.
  Clusters could be out of compliance if their policy was updated after the
  cluster was last edited.

  Arguments:
    POLICY_ID: Canonical unique identifier for the cluster policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listComplianceReq.PolicyId = args[0]

		response := w.PolicyComplianceForClusters.ListCompliance(ctx, listComplianceReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listComplianceOverrides {
		fn(cmd, &listComplianceReq)
	}

	return cmd
}

// end service PolicyComplianceForClusters
