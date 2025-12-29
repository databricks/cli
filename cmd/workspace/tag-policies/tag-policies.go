// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package tag_policies

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/tags"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag-policies",
		Short: `The Tag Policy API allows you to manage policies for governed tags in Databricks.`,
		Long: `The Tag Policy API allows you to manage policies for governed tags in
  Databricks. For Terraform usage, see the [Tag Policy Terraform documentation].
  Permissions for tag policies can be managed using the [Account Access Control
  Proxy API].

  [Account Access Control Proxy API]: https://docs.databricks.com/api/workspace/accountaccesscontrolproxy
  [Tag Policy Terraform documentation]: https://registry.terraform.io/providers/databricks/databricks/latest/docs/resources/tag_policy`,
		GroupID: "tags",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateTagPolicy())
	cmd.AddCommand(newDeleteTagPolicy())
	cmd.AddCommand(newGetTagPolicy())
	cmd.AddCommand(newListTagPolicies())
	cmd.AddCommand(newUpdateTagPolicy())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-tag-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createTagPolicyOverrides []func(
	*cobra.Command,
	*tags.CreateTagPolicyRequest,
)

func newCreateTagPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var createTagPolicyReq tags.CreateTagPolicyRequest
	createTagPolicyReq.TagPolicy = tags.TagPolicy{}
	var createTagPolicyJson flags.JsonFlag

	cmd.Flags().Var(&createTagPolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createTagPolicyReq.TagPolicy.Description, "description", createTagPolicyReq.TagPolicy.Description, ``)
	// TODO: array: values

	cmd.Use = "create-tag-policy TAG_KEY"
	cmd.Short = `Create a new tag policy.`
	cmd.Long = `Create a new tag policy.

  Creates a new tag policy, making the associated tag key governed. For
  Terraform usage, see the [Tag Policy Terraform documentation]. To manage
  permissions for tag policies, use the [Account Access Control Proxy API].

  [Account Access Control Proxy API]: https://docs.databricks.com/api/workspace/accountaccesscontrolproxy
  [Tag Policy Terraform documentation]: https://registry.terraform.io/providers/databricks/databricks/latest/docs/resources/tag_policy`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'tag_key' in your JSON input")
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
			diags := createTagPolicyJson.Unmarshal(&createTagPolicyReq.TagPolicy)
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
			createTagPolicyReq.TagPolicy.TagKey = args[0]
		}

		response, err := w.TagPolicies.CreateTagPolicy(ctx, createTagPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createTagPolicyOverrides {
		fn(cmd, &createTagPolicyReq)
	}

	return cmd
}

// start delete-tag-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteTagPolicyOverrides []func(
	*cobra.Command,
	*tags.DeleteTagPolicyRequest,
)

func newDeleteTagPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteTagPolicyReq tags.DeleteTagPolicyRequest

	cmd.Use = "delete-tag-policy TAG_KEY"
	cmd.Short = `Delete a tag policy.`
	cmd.Long = `Delete a tag policy.

  Deletes a tag policy by its associated governed tag's key, leaving that tag
  key ungoverned. For Terraform usage, see the [Tag Policy Terraform
  documentation].

  [Tag Policy Terraform documentation]: https://registry.terraform.io/providers/databricks/databricks/latest/docs/resources/tag_policy`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteTagPolicyReq.TagKey = args[0]

		err = w.TagPolicies.DeleteTagPolicy(ctx, deleteTagPolicyReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteTagPolicyOverrides {
		fn(cmd, &deleteTagPolicyReq)
	}

	return cmd
}

// start get-tag-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getTagPolicyOverrides []func(
	*cobra.Command,
	*tags.GetTagPolicyRequest,
)

func newGetTagPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var getTagPolicyReq tags.GetTagPolicyRequest

	cmd.Use = "get-tag-policy TAG_KEY"
	cmd.Short = `Get a tag policy.`
	cmd.Long = `Get a tag policy.

  Gets a single tag policy by its associated governed tag's key. For Terraform
  usage, see the [Tag Policy Terraform documentation]. To list granted
  permissions for tag policies, use the [Account Access Control Proxy API].

  [Account Access Control Proxy API]: https://docs.databricks.com/api/workspace/accountaccesscontrolproxy
  [Tag Policy Terraform documentation]: https://registry.terraform.io/providers/databricks/databricks/latest/docs/data-sources/tag_policy`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getTagPolicyReq.TagKey = args[0]

		response, err := w.TagPolicies.GetTagPolicy(ctx, getTagPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getTagPolicyOverrides {
		fn(cmd, &getTagPolicyReq)
	}

	return cmd
}

// start list-tag-policies command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listTagPoliciesOverrides []func(
	*cobra.Command,
	*tags.ListTagPoliciesRequest,
)

func newListTagPolicies() *cobra.Command {
	cmd := &cobra.Command{}

	var listTagPoliciesReq tags.ListTagPoliciesRequest

	cmd.Flags().IntVar(&listTagPoliciesReq.PageSize, "page-size", listTagPoliciesReq.PageSize, `The maximum number of results to return in this request.`)
	cmd.Flags().StringVar(&listTagPoliciesReq.PageToken, "page-token", listTagPoliciesReq.PageToken, `An optional page token received from a previous list tag policies call.`)

	cmd.Use = "list-tag-policies"
	cmd.Short = `List tag policies.`
	cmd.Long = `List tag policies.

  Lists the tag policies for all governed tags in the account. For Terraform
  usage, see the [Tag Policy Terraform documentation]. To list granted
  permissions for tag policies, use the [Account Access Control Proxy API].

  [Account Access Control Proxy API]: https://docs.databricks.com/api/workspace/accountaccesscontrolproxy
  [Tag Policy Terraform documentation]: https://registry.terraform.io/providers/databricks/databricks/latest/docs/data-sources/tag_policies`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.TagPolicies.ListTagPolicies(ctx, listTagPoliciesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listTagPoliciesOverrides {
		fn(cmd, &listTagPoliciesReq)
	}

	return cmd
}

// start update-tag-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateTagPolicyOverrides []func(
	*cobra.Command,
	*tags.UpdateTagPolicyRequest,
)

func newUpdateTagPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateTagPolicyReq tags.UpdateTagPolicyRequest
	updateTagPolicyReq.TagPolicy = tags.TagPolicy{}
	var updateTagPolicyJson flags.JsonFlag

	cmd.Flags().Var(&updateTagPolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateTagPolicyReq.TagPolicy.Description, "description", updateTagPolicyReq.TagPolicy.Description, ``)
	// TODO: array: values

	cmd.Use = "update-tag-policy TAG_KEY UPDATE_MASK"
	cmd.Short = `Update an existing tag policy.`
	cmd.Long = `Update an existing tag policy.

  Updates an existing tag policy for a single governed tag. For Terraform usage,
  see the [Tag Policy Terraform documentation]. To manage permissions for tag
  policies, use the [Account Access Control Proxy API].

  [Account Access Control Proxy API]: https://docs.databricks.com/api/workspace/accountaccesscontrolproxy
  [Tag Policy Terraform documentation]: https://registry.terraform.io/providers/databricks/databricks/latest/docs/resources/tag_policy

  Arguments:
    TAG_KEY:
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateTagPolicyJson.Unmarshal(&updateTagPolicyReq.TagPolicy)
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
		updateTagPolicyReq.TagKey = args[0]
		updateTagPolicyReq.UpdateMask = args[1]

		response, err := w.TagPolicies.UpdateTagPolicy(ctx, updateTagPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateTagPolicyOverrides {
		fn(cmd, &updateTagPolicyReq)
	}

	return cmd
}

// end service TagPolicies
