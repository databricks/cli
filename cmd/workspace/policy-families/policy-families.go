// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package policy_families

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "compute",
		},
	}

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
	*compute.GetPolicyFamilyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq compute.GetPolicyFamilyRequest

	// TODO: short flags

	cmd.Use = "get POLICY_FAMILY_ID"
	cmd.Short = `Get policy family information.`
	cmd.Long = `Get policy family information.
  
  Retrieve the information for an policy family based on its identifier.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.PolicyFamilyId = args[0]

		response, err := w.PolicyFamilies.Get(ctx, getReq)
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
	*compute.ListPolicyFamiliesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq compute.ListPolicyFamiliesRequest

	// TODO: short flags

	cmd.Flags().Int64Var(&listReq.MaxResults, "max-results", listReq.MaxResults, `The max number of policy families to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `A token that can be used to get the next page of results.`)

	cmd.Use = "list"
	cmd.Short = `List policy families.`
	cmd.Long = `List policy families.
  
  Retrieve a list of policy families. This API is paginated.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		err := check(cmd, args)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, cmd.UsageString())
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response := w.PolicyFamilies.List(ctx, listReq)
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

// end service PolicyFamilies
