// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package access_control

import (
	"errors"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access-control",
		Short: `*Public Preview* These APIs manage access rules on resources in an account.`,
		Long: `This command is in Public Preview and may change without notice.

These APIs manage access rules on resources in an account. Currently, only
  grant rules are supported. A grant rule specifies a role assigned to a set of
  principals. A list of rules attached to a resource is called a rule set.`,
		GroupID: "iam",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newGetAssignableRolesForResource())
	cmd.AddCommand(newGetRuleSet())
	cmd.AddCommand(newUpdateRuleSet())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-assignable-roles-for-resource command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getAssignableRolesForResourceOverrides []func(
	*cobra.Command,
	*accessmanagement.ListAssignableRolesForResourceRequest,
)

func newGetAssignableRolesForResource() *cobra.Command {
	cmd := &cobra.Command{}
	var getAssignableRolesForResourceReq accessmanagement.ListAssignableRolesForResourceRequest

	cmd.Use = "get-assignable-roles-for-resource RESOURCE"
	cmd.Short = `*Public Preview* Get assignable roles for a resource.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get assignable roles for a resource.

  Gets all the roles that can be granted on an account level resource. A role is
  grantable if the rule set on the resource can contain an access rule of the
  role.

  Arguments:
    RESOURCE: The resource name for which assignable roles will be listed.

      Examples | Summary :--- | :--- resource=accounts/<ACCOUNT_ID> | A
      resource name for the account.
      resource=accounts/<ACCOUNT_ID>/groups/<GROUP_ID> | A resource name for
      the group. resource=accounts/<ACCOUNT_ID>/servicePrincipals/<SP_ID> | A
      resource name for the service principal.
      resource=accounts/<ACCOUNT_ID>/tagPolicies/<TAG_POLICY_ID> | A resource
      name for the tag policy.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getAssignableRolesForResourceReq.Resource = new(args[0])

		sdkReq, err := accessmanagement.ListAssignableRolesForResourceRequestToSDK(&getAssignableRolesForResourceReq)
		if err != nil {
			return err
		}
		response, err := a.AccessManagement.ListAssignableRolesForResource(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.ListAssignableRolesForResourceResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getAssignableRolesForResourceOverrides {
		fn(cmd, &getAssignableRolesForResourceReq)
	}

	return cmd
}

// start get-rule-set command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRuleSetOverrides []func(
	*cobra.Command,
	*accessmanagement.GetRuleSetRequest,
)

func newGetRuleSet() *cobra.Command {
	cmd := &cobra.Command{}
	var getRuleSetReq accessmanagement.GetRuleSetRequest

	cmd.Use = "get-rule-set NAME ETAG"
	cmd.Short = `*Public Preview* Get a rule set.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a rule set.

  Get a rule set by its name. A rule set is always attached to a resource and
  contains a list of access rules on the said resource. Currently only a default
  rule set for each resource is supported.

  Arguments:
    NAME: The ruleset name associated with the request.

      Examples | Summary :--- | :---
      name=accounts/<ACCOUNT_ID>/ruleSets/default | A name for a rule set on
      the account.
      name=accounts/<ACCOUNT_ID>/groups/<GROUP_ID>/ruleSets/default | A name
      for a rule set on the group.
      name=accounts/<ACCOUNT_ID>/servicePrincipals/<SERVICE_PRINCIPAL_APPLICATION_ID>/ruleSets/default
      | A name for a rule set on the service principal.
      name=accounts/<ACCOUNT_ID>/tagPolicies/<TAG_POLICY_ID>/ruleSets/default
      | A name for a rule set on the tag policy.
    ETAG: Etag used for versioning. The response is at least as fresh as the eTag
      provided. Etag is used for optimistic concurrency control as a way to help
      prevent simultaneous updates of a rule set from overwriting each other. It
      is strongly suggested that systems make use of the etag in the read ->
      modify -> write pattern to perform rule set updates in order to avoid race
      conditions that is get an etag from a GET rule set request, and pass it
      with the PUT update request to identify the rule set version you are
      updating.

      Examples | Summary :--- | :--- etag= | An empty etag can only be used in
      GET to indicate no freshness requirements.
      etag=RENUAAABhSweA4NvVmmUYdiU717H3Tgy0UJdor3gE4a+mq/oj9NjAf8ZsQ== | An
      etag encoded a specific version of the rule set to get or to be updated.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getRuleSetReq.Name = new(args[0])
		getRuleSetReq.Etag = new(args[1])

		sdkReq, err := accessmanagement.GetRuleSetRequestToSDK(&getRuleSetReq)
		if err != nil {
			return err
		}
		response, err := a.AccessManagement.GetRuleSet(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.RuleSetFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRuleSetOverrides {
		fn(cmd, &getRuleSetReq)
	}

	return cmd
}

// start update-rule-set command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateRuleSetOverrides []func(
	*cobra.Command,
	*accessmanagement.UpdateRuleSetRequest,
)

func newUpdateRuleSet() *cobra.Command {
	cmd := &cobra.Command{}
	var updateRuleSetReq accessmanagement.UpdateRuleSetRequest

	var updateRuleSetJson flags.JsonFlag

	cmd.Flags().Var(&updateRuleSetJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-rule-set"
	cmd.Short = `*Public Preview* Update a rule set.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update a rule set.

  Replace the rules of a rule set. First, use get to read the current version of
  the rule set before modifying it. This pattern helps prevent conflicts between
  concurrent updates.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateRuleSetJson.Unmarshal(&updateRuleSetReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := accessmanagement.UpdateRuleSetRequestToSDK(&updateRuleSetReq)
		if err != nil {
			return err
		}
		response, err := a.AccessManagement.UpdateRuleSet(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.RuleSetFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateRuleSetOverrides {
		fn(cmd, &updateRuleSetReq)
	}

	return cmd
}

// end service AccountAccessControl
