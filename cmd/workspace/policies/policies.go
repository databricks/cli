// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package policies

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policies",
		Short: `Attribute-Based Access Control (ABAC) provides high leverage governance for enforcing compliance policies in Unity Catalog.`,
		Long: `Attribute-Based Access Control (ABAC) provides high leverage governance for
  enforcing compliance policies in Unity Catalog. With ABAC policies, access is
  controlled in a hierarchical and scalable manner, based on data attributes
  rather than specific resources, enabling more flexible and comprehensive
  access control. ABAC policies in Unity Catalog support conditions on securable
  properties, governance tags, and environment contexts. Callers must have the
  MANAGE privilege on a securable to view, create, update, or delete ABAC
  policies.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreatePolicy())
	cmd.AddCommand(newDeletePolicy())
	cmd.AddCommand(newGetPolicy())
	cmd.AddCommand(newListPolicies())
	cmd.AddCommand(newUpdatePolicy())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createPolicyOverrides []func(
	*cobra.Command,
	*catalog.CreatePolicyRequest,
)

func newCreatePolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var createPolicyReq catalog.CreatePolicyRequest
	createPolicyReq.PolicyInfo = catalog.PolicyInfo{}
	var createPolicyJson flags.JsonFlag

	cmd.Flags().Var(&createPolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: column_mask
	cmd.Flags().StringVar(&createPolicyReq.PolicyInfo.Comment, "comment", createPolicyReq.PolicyInfo.Comment, `Optional description of the policy.`)
	// TODO: array: except_principals
	// TODO: array: match_columns
	cmd.Flags().StringVar(&createPolicyReq.PolicyInfo.Name, "name", createPolicyReq.PolicyInfo.Name, `Name of the policy.`)
	cmd.Flags().StringVar(&createPolicyReq.PolicyInfo.OnSecurableFullname, "on-securable-fullname", createPolicyReq.PolicyInfo.OnSecurableFullname, `Full name of the securable on which the policy is defined.`)
	cmd.Flags().Var(&createPolicyReq.PolicyInfo.OnSecurableType, "on-securable-type", `Type of the securable on which the policy is defined. Supported values: [
  CATALOG,
  CLEAN_ROOM,
  CONNECTION,
  CREDENTIAL,
  EXTERNAL_LOCATION,
  EXTERNAL_METADATA,
  FUNCTION,
  METASTORE,
  PIPELINE,
  PROVIDER,
  RECIPIENT,
  SCHEMA,
  SHARE,
  STAGING_TABLE,
  STORAGE_CREDENTIAL,
  TABLE,
  VOLUME,
]`)
	// TODO: complex arg: row_filter
	cmd.Flags().StringVar(&createPolicyReq.PolicyInfo.WhenCondition, "when-condition", createPolicyReq.PolicyInfo.WhenCondition, `Optional condition when the policy should take effect.`)

	cmd.Use = "create-policy TO_PRINCIPALS FOR_SECURABLE_TYPE POLICY_TYPE"
	cmd.Short = `Create an ABAC policy.`
	cmd.Long = `Create an ABAC policy.
  
  Creates a new policy on a securable. The new policy applies to the securable
  and all its descendants.

  Arguments:
    TO_PRINCIPALS: List of user or group names that the policy applies to. Required on create
      and optional on update.
    FOR_SECURABLE_TYPE: Type of securables that the policy should take effect on. Only TABLE is
      supported at this moment. Required on create and optional on update. 
      Supported values: [
        CATALOG,
        CLEAN_ROOM,
        CONNECTION,
        CREDENTIAL,
        EXTERNAL_LOCATION,
        EXTERNAL_METADATA,
        FUNCTION,
        METASTORE,
        PIPELINE,
        PROVIDER,
        RECIPIENT,
        SCHEMA,
        SHARE,
        STAGING_TABLE,
        STORAGE_CREDENTIAL,
        TABLE,
        VOLUME,
      ]
    POLICY_TYPE: Type of the policy. Required on create and ignored on update. 
      Supported values: [POLICY_TYPE_COLUMN_MASK, POLICY_TYPE_ROW_FILTER]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'to_principals', 'for_securable_type', 'policy_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createPolicyJson.Unmarshal(&createPolicyReq.PolicyInfo)
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
			_, err = fmt.Sscan(args[0], &createPolicyReq.PolicyInfo.ToPrincipals)
			if err != nil {
				return fmt.Errorf("invalid TO_PRINCIPALS: %s", args[0])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createPolicyReq.PolicyInfo.ForSecurableType)
			if err != nil {
				return fmt.Errorf("invalid FOR_SECURABLE_TYPE: %s", args[1])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createPolicyReq.PolicyInfo.PolicyType)
			if err != nil {
				return fmt.Errorf("invalid POLICY_TYPE: %s", args[2])
			}

		}

		response, err := w.Policies.CreatePolicy(ctx, createPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createPolicyOverrides {
		fn(cmd, &createPolicyReq)
	}

	return cmd
}

// start delete-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deletePolicyOverrides []func(
	*cobra.Command,
	*catalog.DeletePolicyRequest,
)

func newDeletePolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var deletePolicyReq catalog.DeletePolicyRequest

	cmd.Use = "delete-policy ON_SECURABLE_TYPE ON_SECURABLE_FULLNAME NAME"
	cmd.Short = `Delete an ABAC policy.`
	cmd.Long = `Delete an ABAC policy.
  
  Delete an ABAC policy defined on a securable.

  Arguments:
    ON_SECURABLE_TYPE: Required. The type of the securable to delete the policy from.
    ON_SECURABLE_FULLNAME: Required. The fully qualified name of the securable to delete the policy
      from.
    NAME: Required. The name of the policy to delete`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deletePolicyReq.OnSecurableType = args[0]
		deletePolicyReq.OnSecurableFullname = args[1]
		deletePolicyReq.Name = args[2]

		response, err := w.Policies.DeletePolicy(ctx, deletePolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deletePolicyOverrides {
		fn(cmd, &deletePolicyReq)
	}

	return cmd
}

// start get-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPolicyOverrides []func(
	*cobra.Command,
	*catalog.GetPolicyRequest,
)

func newGetPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var getPolicyReq catalog.GetPolicyRequest

	cmd.Use = "get-policy ON_SECURABLE_TYPE ON_SECURABLE_FULLNAME NAME"
	cmd.Short = `Get an ABAC policy.`
	cmd.Long = `Get an ABAC policy.
  
  Get the policy definition on a securable

  Arguments:
    ON_SECURABLE_TYPE: Required. The type of the securable to retrieve the policy for.
    ON_SECURABLE_FULLNAME: Required. The fully qualified name of securable to retrieve policy for.
    NAME: Required. The name of the policy to retrieve.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPolicyReq.OnSecurableType = args[0]
		getPolicyReq.OnSecurableFullname = args[1]
		getPolicyReq.Name = args[2]

		response, err := w.Policies.GetPolicy(ctx, getPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPolicyOverrides {
		fn(cmd, &getPolicyReq)
	}

	return cmd
}

// start list-policies command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listPoliciesOverrides []func(
	*cobra.Command,
	*catalog.ListPoliciesRequest,
)

func newListPolicies() *cobra.Command {
	cmd := &cobra.Command{}

	var listPoliciesReq catalog.ListPoliciesRequest

	cmd.Flags().BoolVar(&listPoliciesReq.IncludeInherited, "include-inherited", listPoliciesReq.IncludeInherited, `Optional.`)
	cmd.Flags().IntVar(&listPoliciesReq.MaxResults, "max-results", listPoliciesReq.MaxResults, `Optional.`)
	cmd.Flags().StringVar(&listPoliciesReq.PageToken, "page-token", listPoliciesReq.PageToken, `Optional.`)

	cmd.Use = "list-policies ON_SECURABLE_TYPE ON_SECURABLE_FULLNAME"
	cmd.Short = `List ABAC policies.`
	cmd.Long = `List ABAC policies.
  
  List all policies defined on a securable. Optionally, the list can include
  inherited policies defined on the securable's parent schema or catalog.
  
  PAGINATION BEHAVIOR: The API is by default paginated, a page may contain zero
  results while still providing a next_page_token. Clients must continue reading
  pages until next_page_token is absent, which is the only indication that the
  end of results has been reached.

  Arguments:
    ON_SECURABLE_TYPE: Required. The type of the securable to list policies for.
    ON_SECURABLE_FULLNAME: Required. The fully qualified name of securable to list policies for.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listPoliciesReq.OnSecurableType = args[0]
		listPoliciesReq.OnSecurableFullname = args[1]

		response := w.Policies.ListPolicies(ctx, listPoliciesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listPoliciesOverrides {
		fn(cmd, &listPoliciesReq)
	}

	return cmd
}

// start update-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePolicyOverrides []func(
	*cobra.Command,
	*catalog.UpdatePolicyRequest,
)

func newUpdatePolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePolicyReq catalog.UpdatePolicyRequest
	updatePolicyReq.PolicyInfo = catalog.PolicyInfo{}
	var updatePolicyJson flags.JsonFlag

	cmd.Flags().Var(&updatePolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updatePolicyReq.UpdateMask, "update-mask", updatePolicyReq.UpdateMask, `Optional.`)
	// TODO: complex arg: column_mask
	cmd.Flags().StringVar(&updatePolicyReq.PolicyInfo.Comment, "comment", updatePolicyReq.PolicyInfo.Comment, `Optional description of the policy.`)
	// TODO: array: except_principals
	// TODO: array: match_columns
	cmd.Flags().StringVar(&updatePolicyReq.PolicyInfo.Name, "name", updatePolicyReq.PolicyInfo.Name, `Name of the policy.`)
	cmd.Flags().StringVar(&updatePolicyReq.PolicyInfo.OnSecurableFullname, "on-securable-fullname", updatePolicyReq.PolicyInfo.OnSecurableFullname, `Full name of the securable on which the policy is defined.`)
	cmd.Flags().Var(&updatePolicyReq.PolicyInfo.OnSecurableType, "on-securable-type", `Type of the securable on which the policy is defined. Supported values: [
  CATALOG,
  CLEAN_ROOM,
  CONNECTION,
  CREDENTIAL,
  EXTERNAL_LOCATION,
  EXTERNAL_METADATA,
  FUNCTION,
  METASTORE,
  PIPELINE,
  PROVIDER,
  RECIPIENT,
  SCHEMA,
  SHARE,
  STAGING_TABLE,
  STORAGE_CREDENTIAL,
  TABLE,
  VOLUME,
]`)
	// TODO: complex arg: row_filter
	cmd.Flags().StringVar(&updatePolicyReq.PolicyInfo.WhenCondition, "when-condition", updatePolicyReq.PolicyInfo.WhenCondition, `Optional condition when the policy should take effect.`)

	cmd.Use = "update-policy ON_SECURABLE_TYPE ON_SECURABLE_FULLNAME NAME TO_PRINCIPALS FOR_SECURABLE_TYPE POLICY_TYPE"
	cmd.Short = `Update an ABAC policy.`
	cmd.Long = `Update an ABAC policy.
  
  Update an ABAC policy on a securable.

  Arguments:
    ON_SECURABLE_TYPE: Required. The type of the securable to update the policy for.
    ON_SECURABLE_FULLNAME: Required. The fully qualified name of the securable to update the policy
      for.
    NAME: Required. The name of the policy to update.
    TO_PRINCIPALS: List of user or group names that the policy applies to. Required on create
      and optional on update.
    FOR_SECURABLE_TYPE: Type of securables that the policy should take effect on. Only TABLE is
      supported at this moment. Required on create and optional on update. 
      Supported values: [
        CATALOG,
        CLEAN_ROOM,
        CONNECTION,
        CREDENTIAL,
        EXTERNAL_LOCATION,
        EXTERNAL_METADATA,
        FUNCTION,
        METASTORE,
        PIPELINE,
        PROVIDER,
        RECIPIENT,
        SCHEMA,
        SHARE,
        STAGING_TABLE,
        STORAGE_CREDENTIAL,
        TABLE,
        VOLUME,
      ]
    POLICY_TYPE: Type of the policy. Required on create and ignored on update. 
      Supported values: [POLICY_TYPE_COLUMN_MASK, POLICY_TYPE_ROW_FILTER]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(3)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only ON_SECURABLE_TYPE, ON_SECURABLE_FULLNAME, NAME as positional arguments. Provide 'to_principals', 'for_securable_type', 'policy_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePolicyJson.Unmarshal(&updatePolicyReq.PolicyInfo)
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
		updatePolicyReq.OnSecurableType = args[0]
		updatePolicyReq.OnSecurableFullname = args[1]
		updatePolicyReq.Name = args[2]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updatePolicyReq.PolicyInfo.ToPrincipals)
			if err != nil {
				return fmt.Errorf("invalid TO_PRINCIPALS: %s", args[3])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[4], &updatePolicyReq.PolicyInfo.ForSecurableType)
			if err != nil {
				return fmt.Errorf("invalid FOR_SECURABLE_TYPE: %s", args[4])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[5], &updatePolicyReq.PolicyInfo.PolicyType)
			if err != nil {
				return fmt.Errorf("invalid POLICY_TYPE: %s", args[5])
			}

		}

		response, err := w.Policies.UpdatePolicy(ctx, updatePolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePolicyOverrides {
		fn(cmd, &updatePolicyReq)
	}

	return cmd
}

// end service Policies
