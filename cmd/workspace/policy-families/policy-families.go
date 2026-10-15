// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package policy_families

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	policyfamilies "github.com/databricks/cli/libs/sdktypes/policyfamilies/v2"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy-families",
		Short: `View available policy families.`,
		Long: `View available policy families. A policy family contains a policy definition
  providing best practices for configuring clusters for a particular use case.

  Databricks manages and provides policy families for several common cluster use
  cases. You cannot create, edit, or delete policy families.

  Policy families cannot be used directly to create clusters. Instead, you
  create cluster policies using a policy family. Cluster policies created using
  a policy family inherit the policy family's policy definition.`,
		GroupID: "compute",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*policyfamilies.GetPolicyFamilyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq policyfamilies.GetPolicyFamilyRequest

	flags.Int64PtrVar(cmd.Flags(), &getReq.Version, "version", `The version number for the family to fetch.`)

	cmd.Use = "get POLICY_FAMILY_ID"
	cmd.Short = `Get policy family information.`
	cmd.Long = `Get policy family information.

  Retrieve the information for an policy family based on its identifier and
  version

  Arguments:
    POLICY_FAMILY_ID: The family ID about which to retrieve information.`

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

		getReq.PolicyFamilyId = new(args[0])

		sdkReq, err := policyfamilies.GetPolicyFamilyRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.PolicyFamilies.GetPolicyFamily(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := policyfamilies.PolicyFamilyFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*policyfamilies.ListPolicyFamiliesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq policyfamilies.ListPolicyFamiliesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.Int64PtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of policy families to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List policy families.`
	cmd.Long = `List policy families.

  Returns the list of policy definition types available to use at their latest
  version. This API is paginated.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := policyfamilies.ListPolicyFamiliesRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.PolicyFamilies.ListPolicyFamiliesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, policyfamilies.PolicyFamilyFromSDK))
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

// end service PolicyFamilies
