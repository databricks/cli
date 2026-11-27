// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package serverless_compute_service

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "serverless-compute-service",
	Short:   `ServerlessCompute Service API.`,
	GroupID: "compute",

	// This service is being previewed; hide from help output.
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `ServerlessCompute Service API.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "PRIVATE_PREVIEW",
		"launch_stage_display": "Private Preview",
	}

	// Add methods
	cmd.AddCommand(newCreateComputePolicy())
	cmd.AddCommand(newDeleteComputePolicy())
	cmd.AddCommand(newGetComputePolicy())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListComputePolicies())
	cmd.AddCommand(newUpdateComputePolicy())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// start create-compute-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createComputePolicyOverrides []func(
	*cobra.Command,
	*compute.CreateComputePolicyRequest,
)

func newCreateComputePolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var createComputePolicyReq compute.CreateComputePolicyRequest
	createComputePolicyReq.ComputePolicy = compute.ComputePolicy{}
	var createComputePolicyJson flags.JsonFlag

	cmd.Flags().Var(&createComputePolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: allowed_compute_products
	cmd.Flags().StringVar(&createComputePolicyReq.ComputePolicy.ComputePolicyId, "compute-policy-id", createComputePolicyReq.ComputePolicy.ComputePolicyId, `The unique identifier of the compute policy (UUID).`)
	cmd.Flags().StringVar(&createComputePolicyReq.ComputePolicy.DisplayName, "display-name", createComputePolicyReq.ComputePolicy.DisplayName, `The display name of the compute policy.`)
	// TODO: map via StringToStringVar: tags

	cmd.Use = "create-compute-policy"
	cmd.Short = `Create a compute policy.`
	cmd.Long = `Create a compute policy.

  Creates a custom compute policy in the workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createComputePolicyJson.Unmarshal(&createComputePolicyReq.ComputePolicy)
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

		response, err := w.ServerlessComputeService.CreateComputePolicy(ctx, createComputePolicyReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createComputePolicyOverrides {
		fn(cmd, &createComputePolicyReq)
	}

	return cmd
}

// start delete-compute-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteComputePolicyOverrides []func(
	*cobra.Command,
	*compute.DeleteComputePolicyRequest,
)

func newDeleteComputePolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteComputePolicyReq compute.DeleteComputePolicyRequest

	cmd.Use = "delete-compute-policy COMPUTE_POLICY_ID"
	cmd.Short = `Delete a compute policy.`
	cmd.Long = `Delete a compute policy.

  Deletes a custom compute policy in the workspace. Default (system-provisioned)
  policies cannot be deleted.

  Arguments:
    COMPUTE_POLICY_ID: The ID of the compute policy to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteComputePolicyReq.ComputePolicyId = args[0]

		err = w.ServerlessComputeService.DeleteComputePolicy(ctx, deleteComputePolicyReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteComputePolicyOverrides {
		fn(cmd, &deleteComputePolicyReq)
	}

	return cmd
}

// start get-compute-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getComputePolicyOverrides []func(
	*cobra.Command,
	*compute.GetComputePolicyRequest,
)

func newGetComputePolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var getComputePolicyReq compute.GetComputePolicyRequest

	cmd.Use = "get-compute-policy COMPUTE_POLICY_ID"
	cmd.Short = `Get a compute policy.`
	cmd.Long = `Get a compute policy.

  Retrieves a compute policy by its ID.

  Arguments:
    COMPUTE_POLICY_ID: The ID of the compute policy to retrieve.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getComputePolicyReq.ComputePolicyId = args[0]

		response, err := w.ServerlessComputeService.GetComputePolicy(ctx, getComputePolicyReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getComputePolicyOverrides {
		fn(cmd, &getComputePolicyReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*compute.ListServerlessComputesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq compute.ListServerlessComputesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Maximum number of computes to return per response.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List serverless computes.`
	cmd.Long = `List serverless computes.

  Lists Interactive serverless computes that the caller can use or manage.
  Results are returned in an unspecified order.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ServerlessComputeService.List(ctx, listReq)
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

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

// start list-compute-policies command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listComputePoliciesOverrides []func(
	*cobra.Command,
	*compute.ListComputePoliciesRequest,
)

func newListComputePolicies() *cobra.Command {
	cmd := &cobra.Command{}

	var listComputePoliciesReq compute.ListComputePoliciesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listComputePoliciesLimit int

	cmd.Flags().IntVar(&listComputePoliciesReq.PageSize, "page-size", listComputePoliciesReq.PageSize, `Maximum number of policies to return per response.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listComputePoliciesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listComputePoliciesReq.PageToken, "page-token", listComputePoliciesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-compute-policies"
	cmd.Short = `List compute policies.`
	cmd.Long = `List compute policies.

  Lists compute policies in the workspace. Results are returned in an
  unspecified order; callers should sort client-side if a specific order is
  needed.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ServerlessComputeService.ListComputePolicies(ctx, listComputePoliciesReq)
		if listComputePoliciesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listComputePoliciesLimit)
		}
		if listComputePoliciesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listComputePoliciesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listComputePoliciesOverrides {
		fn(cmd, &listComputePoliciesReq)
	}

	return cmd
}

// start update-compute-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateComputePolicyOverrides []func(
	*cobra.Command,
	*compute.UpdateComputePolicyRequest,
)

func newUpdateComputePolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateComputePolicyReq compute.UpdateComputePolicyRequest
	updateComputePolicyReq.ComputePolicy = compute.ComputePolicy{}
	var updateComputePolicyJson flags.JsonFlag

	cmd.Flags().Var(&updateComputePolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: allowed_compute_products
	cmd.Flags().StringVar(&updateComputePolicyReq.ComputePolicy.ComputePolicyId, "compute-policy-id", updateComputePolicyReq.ComputePolicy.ComputePolicyId, `The unique identifier of the compute policy (UUID).`)
	cmd.Flags().StringVar(&updateComputePolicyReq.ComputePolicy.DisplayName, "display-name", updateComputePolicyReq.ComputePolicy.DisplayName, `The display name of the compute policy.`)
	// TODO: map via StringToStringVar: tags

	cmd.Use = "update-compute-policy COMPUTE_POLICY_ID UPDATE_MASK"
	cmd.Short = `Update a compute policy.`
	cmd.Long = `Update a compute policy.

  Updates a custom compute policy in the workspace. The display_name and
  allowed_compute_products are mutable; use update_mask to control which fields
  change. Default (system-provisioned) policies cannot be updated.

  Arguments:
    COMPUTE_POLICY_ID: The unique identifier of the compute policy (UUID). Server-assigned on
      create.
    UPDATE_MASK: Fields to update, relative to compute_policy. Must contain at least one
      path; the supported paths are display_name and
      allowed_compute_products. Supplied product arms replace their
      constraints and omitted products remain unchanged. An empty arm clears
      that product's limits; an absent or empty products object is invalid.
      Removing a product requires its delete API. Nested masks and wildcards are
      unsupported. display_name cannot be cleared: if it is in the mask it
      must be non-empty.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateComputePolicyJson.Unmarshal(&updateComputePolicyReq.ComputePolicy)
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
		updateComputePolicyReq.ComputePolicyId = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateComputePolicyReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.ServerlessComputeService.UpdateComputePolicy(ctx, updateComputePolicyReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateComputePolicyOverrides {
		fn(cmd, &updateComputePolicyReq)
	}

	return cmd
}

// end service ServerlessComputeService
