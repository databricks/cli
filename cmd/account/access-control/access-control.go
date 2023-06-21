// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package access_control

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "access-control",
	Short: `These APIs manage access rules on resources in an account.`,
	Long: `These APIs manage access rules on resources in an account. Currently, only
  grant rules are supported. A grant rule specifies a role assigned to a set of
  principals. A list of rules attached to a resource is called a rule set.`,
	Annotations: map[string]string{
		"package": "iam",
	},
}

// start get-assignable-roles-for-resource command

var getAssignableRolesForResourceReq iam.GetAssignableRolesForResourceRequest
var getAssignableRolesForResourceJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getAssignableRolesForResourceCmd)
	// TODO: short flags
	getAssignableRolesForResourceCmd.Flags().Var(&getAssignableRolesForResourceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getAssignableRolesForResourceCmd = &cobra.Command{
	Use:   "get-assignable-roles-for-resource RESOURCE",
	Short: `Get assignable roles for a resource.`,
	Long: `Get assignable roles for a resource.
  
  Gets all the roles that can be granted on an account level resource. A role is
  grantable if the rule set on the resource can contain an access rule of the
  role.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getAssignableRolesForResourceJson.Unmarshal(&getAssignableRolesForResourceReq)
			if err != nil {
				return err
			}
		} else {
			getAssignableRolesForResourceReq.Resource = args[0]
		}

		response, err := a.AccessControl.GetAssignableRolesForResource(ctx, getAssignableRolesForResourceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get-rule-set command

var getRuleSetReq iam.GetRuleSetRequest
var getRuleSetJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getRuleSetCmd)
	// TODO: short flags
	getRuleSetCmd.Flags().Var(&getRuleSetJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getRuleSetCmd = &cobra.Command{
	Use:   "get-rule-set NAME ETAG",
	Short: `Get a rule set.`,
	Long: `Get a rule set.
  
  Get a rule set by its name. A rule set is always attached to a resource and
  contains a list of access rules on the said resource. Currently only a default
  rule set for each resource is supported.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getRuleSetJson.Unmarshal(&getRuleSetReq)
			if err != nil {
				return err
			}
		} else {
			getRuleSetReq.Name = args[0]
			getRuleSetReq.Etag = args[1]
		}

		response, err := a.AccessControl.GetRuleSet(ctx, getRuleSetReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start update-rule-set command

var updateRuleSetReq iam.UpdateRuleSetRequest
var updateRuleSetJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateRuleSetCmd)
	// TODO: short flags
	updateRuleSetCmd.Flags().Var(&updateRuleSetJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var updateRuleSetCmd = &cobra.Command{
	Use:   "update-rule-set",
	Short: `Update a rule set.`,
	Long: `Update a rule set.
  
  Replace the rules of a rule set. First, use get to read the current version of
  the rule set before modifying it. This pattern helps prevent conflicts between
  concurrent updates.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateRuleSetJson.Unmarshal(&updateRuleSetReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := a.AccessControl.UpdateRuleSet(ctx, updateRuleSetReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service AccountAccessControl
