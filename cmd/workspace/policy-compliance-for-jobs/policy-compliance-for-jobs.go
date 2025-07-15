// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package policy_compliance_for_jobs

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy-compliance-for-jobs",
		Short: `The compliance APIs allow you to view and manage the policy compliance status of jobs in your workspace.`,
		Long: `The compliance APIs allow you to view and manage the policy compliance status
  of jobs in your workspace. This API currently only supports compliance
  controls for cluster policies.
  
  A job is in compliance if its cluster configurations satisfy the rules of all
  their respective cluster policies. A job could be out of compliance if a
  cluster policy it uses was updated after the job was last edited. The job is
  considered out of compliance if any of its clusters no longer comply with
  their updated policies.
  
  The get and list compliance APIs allow you to view the policy compliance
  status of a job. The enforce compliance API allows you to update a job so that
  it becomes compliant with all of its policies.`,
		GroupID: "jobs",
		Annotations: map[string]string{
			"package": "jobs",
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
	*jobs.EnforcePolicyComplianceRequest,
)

func newEnforceCompliance() *cobra.Command {
	cmd := &cobra.Command{}

	var enforceComplianceReq jobs.EnforcePolicyComplianceRequest
	var enforceComplianceJson flags.JsonFlag

	cmd.Flags().Var(&enforceComplianceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&enforceComplianceReq.ValidateOnly, "validate-only", enforceComplianceReq.ValidateOnly, `If set, previews changes made to the job to comply with its policy, but does not update the job.`)

	cmd.Use = "enforce-compliance JOB_ID"
	cmd.Short = `Enforce job policy compliance.`
	cmd.Long = `Enforce job policy compliance.
  
  Updates a job so the job clusters that are created when running the job
  (specified in new_cluster) are compliant with the current versions of their
  respective cluster policies. All-purpose clusters used in the job will not be
  updated.

  Arguments:
    JOB_ID: The ID of the job you want to enforce policy compliance on.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'job_id' in your JSON input")
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
			_, err = fmt.Sscan(args[0], &enforceComplianceReq.JobId)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
		}

		response, err := w.PolicyComplianceForJobs.EnforceCompliance(ctx, enforceComplianceReq)
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
	*jobs.GetPolicyComplianceRequest,
)

func newGetCompliance() *cobra.Command {
	cmd := &cobra.Command{}

	var getComplianceReq jobs.GetPolicyComplianceRequest

	cmd.Use = "get-compliance JOB_ID"
	cmd.Short = `Get job policy compliance.`
	cmd.Long = `Get job policy compliance.
  
  Returns the policy compliance status of a job. Jobs could be out of compliance
  if a cluster policy they use was updated after the job was last edited and
  some of its job clusters no longer comply with their updated policies.

  Arguments:
    JOB_ID: The ID of the job whose compliance status you are requesting.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getComplianceReq.JobId)
		if err != nil {
			return fmt.Errorf("invalid JOB_ID: %s", args[0])
		}

		response, err := w.PolicyComplianceForJobs.GetCompliance(ctx, getComplianceReq)
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
	*jobs.ListJobComplianceRequest,
)

func newListCompliance() *cobra.Command {
	cmd := &cobra.Command{}

	var listComplianceReq jobs.ListJobComplianceRequest

	cmd.Flags().IntVar(&listComplianceReq.PageSize, "page-size", listComplianceReq.PageSize, `Use this field to specify the maximum number of results to be returned by the server.`)
	cmd.Flags().StringVar(&listComplianceReq.PageToken, "page-token", listComplianceReq.PageToken, `A page token that can be used to navigate to the next page or previous page as returned by next_page_token or prev_page_token.`)

	cmd.Use = "list-compliance POLICY_ID"
	cmd.Short = `List job policy compliance.`
	cmd.Long = `List job policy compliance.
  
  Returns the policy compliance status of all jobs that use a given policy. Jobs
  could be out of compliance if a cluster policy they use was updated after the
  job was last edited and its job clusters no longer comply with the updated
  policy.

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

		response := w.PolicyComplianceForJobs.ListCompliance(ctx, listComplianceReq)
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

// end service PolicyComplianceForJobs
