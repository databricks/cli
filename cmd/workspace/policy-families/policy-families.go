// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package policy_families

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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
	Annotations: map[string]string{
		"package": "compute",
	},
}

// start get command
var getReq compute.GetPolicyFamilyRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get POLICY_FAMILY_ID",
	Short: `Get policy family information.`,
	Long: `Get policy family information.
  
  Retrieve the information for an policy family based on its identifier.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.PolicyFamilyId = args[0]

		response, err := w.PolicyFamilies.Get(ctx, getReq)
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
var listReq compute.ListPolicyFamiliesRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().Int64Var(&listReq.MaxResults, "max-results", listReq.MaxResults, `The max number of policy families to return.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `A token that can be used to get the next page of results.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List policy families.`,
	Long: `List policy families.
  
  Retrieve a list of policy families. This API is paginated.`,

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

		response, err := w.PolicyFamilies.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service PolicyFamilies
