// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package policy_families

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "policy-families",
	Short: `View available policy families.`,
	Long: `View available policy families. A policy family contains a policy definition
  providing best practices for configuring clusters for a particular use case.
  
  Databricks manages and provides policy families for several common cluster use
  cases. You cannot create, edit, or delete policy families.
  
  Policy families cannot be used directly to create clusters. Instead, you
  create cluster policies using a policy family. Cluster policies created using
  a policy family inherit the policy family's policy definition.`,
}

// start get command

var getReq compute.GetPolicyFamilyRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use: "get POLICY_FAMILY_ID",

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		getReq.PolicyFamilyId = args[0]

		response, err := w.PolicyFamilies.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq compute.ListPolicyFamiliesRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().Int64Var(&listReq.MaxResults, "max-results", listReq.MaxResults, `The max number of policy families to return.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `A token that can be used to get the next page of results.`)

}

var listCmd = &cobra.Command{
	Use: "list",

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.PolicyFamilies.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service PolicyFamilies
