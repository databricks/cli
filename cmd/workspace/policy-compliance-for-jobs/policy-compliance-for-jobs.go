// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package policy_compliance_for_jobs

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
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
		GroupID: "lakeflow",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*jobs.EnforcePolicyComplianceForJob,
)

func newEnforceCompliance() *cobra.Command {
	cmd := &cobra.Command{}
	var enforceComplianceReq jobs.EnforcePolicyComplianceForJob

	var enforceComplianceJson flags.JsonFlag

	cmd.Flags().Var(&enforceComplianceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &enforceComplianceReq.ValidateOnly, "validate-only", `If set, previews changes made to the job to comply with its policy, but does not update the job.`)

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'job_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := enforceComplianceJson.Unmarshal(&enforceComplianceReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			var enforceComplianceJobIdArg int64
			_, err = fmt.Sscan(args[0], &enforceComplianceJobIdArg)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
			enforceComplianceReq.JobId = &enforceComplianceJobIdArg
		}

		sdkReq, err := jobs.EnforcePolicyComplianceForJobToSDK(&enforceComplianceReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.EnforcePolicyComplianceForJob(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := jobs.EnforcePolicyComplianceResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*jobs.GetPolicyComplianceForJobRequest,
)

func newGetCompliance() *cobra.Command {
	cmd := &cobra.Command{}
	var getComplianceReq jobs.GetPolicyComplianceForJobRequest

	cmd.Use = "get-compliance JOB_ID"
	cmd.Short = `Get job policy compliance.`
	cmd.Long = `Get job policy compliance.

  Returns the policy compliance status of a job. Jobs could be out of compliance
  if a cluster policy they use was updated after the job was last edited and
  some of its job clusters no longer comply with their updated policies.

  Arguments:
    JOB_ID: The ID of the job whose compliance status you are requesting.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		var getComplianceJobIdArg int64
		_, err = fmt.Sscan(args[0], &getComplianceJobIdArg)
		if err != nil {
			return fmt.Errorf("invalid JOB_ID: %s", args[0])
		}
		getComplianceReq.JobId = &getComplianceJobIdArg

		sdkReq, err := jobs.GetPolicyComplianceForJobRequestToSDK(&getComplianceReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.GetPolicyComplianceForJob(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := jobs.GetPolicyComplianceForJobResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*jobs.ListJobComplianceForPolicy,
)

func newListCompliance() *cobra.Command {
	cmd := &cobra.Command{}
	var listComplianceReq jobs.ListJobComplianceForPolicy

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listComplianceLimit int

	flags.IntPtrVar(cmd.Flags(), &listComplianceReq.PageSize, "page-size", `Use this field to specify the maximum number of results to be returned by the server.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listComplianceLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listComplianceReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listComplianceReq.PolicyId = new(args[0])

		sdkReq, err := jobs.ListJobComplianceForPolicyToSDK(&listComplianceReq)
		if err != nil {
			return err
		}
		if listComplianceLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listComplianceLimit)
		}
		if listComplianceLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listComplianceLimit)
		}
		response := w.Jobs.ListJobComplianceForPolicyIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, jobs.ListJobComplianceForPolicy_JobComplianceFromSDK))
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
