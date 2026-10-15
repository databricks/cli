// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_entity_tag_assignments

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	tagassignments "github.com/databricks/cli/libs/sdktypes/tagassignments/v1"
	sdktagassignments "github.com/databricks/sdk-go/tagassignments/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace-entity-tag-assignments",
		Short:   `Manage tag assignments on workspace-scoped objects.`,
		Long:    `Manage tag assignments on workspace-scoped objects.`,
		GroupID: "tags",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateTagAssignment())
	cmd.AddCommand(newDeleteTagAssignment())
	cmd.AddCommand(newGetTagAssignment())
	cmd.AddCommand(newListTagAssignments())
	cmd.AddCommand(newUpdateTagAssignment())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-tag-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createTagAssignmentOverrides []func(
	*cobra.Command,
	*tagassignments.CreateTagAssignmentRequest,
)

func newCreateTagAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var createTagAssignmentReq tagassignments.CreateTagAssignmentRequest
	createTagAssignmentReq.TagAssignment = &tagassignments.TagAssignment{}

	var createTagAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&createTagAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createTagAssignmentReq.TagAssignment.TagValue, "tag-value", `The value of the tag.`)

	cmd.Use = "create-tag-assignment ENTITY_TYPE ENTITY_ID TAG_KEY"
	cmd.Short = `Create a tag assignment for an entity.`
	cmd.Long = `Create a tag assignment for an entity.

  Create a tag assignment

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces, notebooks
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name
    TAG_KEY: The key of the tag. The characters , . : / - = and leading/trailing spaces
      are not allowed`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'entity_type', 'entity_id', 'tag_key' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createTagAssignmentJson.Unmarshal(&createTagAssignmentReq.TagAssignment)
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
			createTagAssignmentReq.TagAssignment.EntityType = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createTagAssignmentReq.TagAssignment.EntityId = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createTagAssignmentReq.TagAssignment.TagKey = new(args[2])
		}

		sdkReq, err := tagassignments.CreateTagAssignmentRequestToSDK(&createTagAssignmentReq)
		if err != nil {
			return err
		}
		response, err := w.TagAssignments.CreateTagAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := tagassignments.TagAssignmentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createTagAssignmentOverrides {
		fn(cmd, &createTagAssignmentReq)
	}

	return cmd
}

// start delete-tag-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteTagAssignmentOverrides []func(
	*cobra.Command,
	*sdktagassignments.DeleteTagAssignmentRequest,
)

func newDeleteTagAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteTagAssignmentReq sdktagassignments.DeleteTagAssignmentRequest

	cmd.Use = "delete-tag-assignment ENTITY_TYPE ENTITY_ID TAG_KEY"
	cmd.Short = `Delete a tag assignment for an entity.`
	cmd.Long = `Delete a tag assignment for an entity.

  Delete a tag assignment

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces, notebooks
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name
    TAG_KEY: The key of the tag. The characters , . : / - = and leading/trailing spaces
      are not allowed`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteTagAssignmentReq.EntityType = new(args[0])
		deleteTagAssignmentReq.EntityId = new(args[1])
		deleteTagAssignmentReq.TagKey = new(args[2])

		sdkReq := &deleteTagAssignmentReq
		err = w.TagAssignments.DeleteTagAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteTagAssignmentOverrides {
		fn(cmd, &deleteTagAssignmentReq)
	}

	return cmd
}

// start get-tag-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getTagAssignmentOverrides []func(
	*cobra.Command,
	*sdktagassignments.GetTagAssignmentRequest,
)

func newGetTagAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var getTagAssignmentReq sdktagassignments.GetTagAssignmentRequest

	cmd.Use = "get-tag-assignment ENTITY_TYPE ENTITY_ID TAG_KEY"
	cmd.Short = `Get a tag assignment for an entity.`
	cmd.Long = `Get a tag assignment for an entity.

  Get a tag assignment

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces, notebooks
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name
    TAG_KEY: The key of the tag. The characters , . : / - = and leading/trailing spaces
      are not allowed`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getTagAssignmentReq.EntityType = new(args[0])
		getTagAssignmentReq.EntityId = new(args[1])
		getTagAssignmentReq.TagKey = new(args[2])

		sdkReq := &getTagAssignmentReq
		response, err := w.TagAssignments.GetTagAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := tagassignments.TagAssignmentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getTagAssignmentOverrides {
		fn(cmd, &getTagAssignmentReq)
	}

	return cmd
}

// start list-tag-assignments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listTagAssignmentsOverrides []func(
	*cobra.Command,
	*tagassignments.ListTagAssignmentsRequest,
)

func newListTagAssignments() *cobra.Command {
	cmd := &cobra.Command{}
	var listTagAssignmentsReq tagassignments.ListTagAssignmentsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listTagAssignmentsLimit int

	flags.IntPtrVar(cmd.Flags(), &listTagAssignmentsReq.PageSize, "page-size", `Optional.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listTagAssignmentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listTagAssignmentsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-tag-assignments ENTITY_TYPE ENTITY_ID"
	cmd.Short = `List tag assignments for an entity.`
	cmd.Long = `List tag assignments for an entity.

  List the tag assignments for an entity

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces, notebooks
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listTagAssignmentsReq.EntityType = new(args[0])
		listTagAssignmentsReq.EntityId = new(args[1])

		sdkReq, err := tagassignments.ListTagAssignmentsRequestToSDK(&listTagAssignmentsReq)
		if err != nil {
			return err
		}
		if listTagAssignmentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listTagAssignmentsLimit)
		}
		if listTagAssignmentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listTagAssignmentsLimit)
		}
		response := w.TagAssignments.ListTagAssignmentsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, tagassignments.TagAssignmentFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listTagAssignmentsOverrides {
		fn(cmd, &listTagAssignmentsReq)
	}

	return cmd
}

// start update-tag-assignment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateTagAssignmentOverrides []func(
	*cobra.Command,
	*tagassignments.UpdateTagAssignmentRequest,
)

func newUpdateTagAssignment() *cobra.Command {
	cmd := &cobra.Command{}
	var updateTagAssignmentReq tagassignments.UpdateTagAssignmentRequest
	updateTagAssignmentReq.TagAssignment = &tagassignments.TagAssignment{}

	var updateTagAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&updateTagAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateTagAssignmentReq.TagAssignment.TagValue, "tag-value", `The value of the tag.`)

	cmd.Use = "update-tag-assignment ENTITY_TYPE ENTITY_ID TAG_KEY UPDATE_MASK"
	cmd.Short = `Update a tag assignment for an entity.`
	cmd.Long = `Update a tag assignment for an entity.

  Update a tag assignment

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces, notebooks
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name
    TAG_KEY: The key of the tag. The characters , . : / - = and leading/trailing spaces
      are not allowed
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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateTagAssignmentJson.Unmarshal(&updateTagAssignmentReq.TagAssignment)
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
		updateTagAssignmentReq.TagAssignment.EntityType = new(args[0])
		updateTagAssignmentReq.TagAssignment.EntityId = new(args[1])
		updateTagAssignmentReq.TagAssignment.TagKey = new(args[2])
		updateTagAssignmentReq.UpdateMask = new(args[3])

		sdkReq, err := tagassignments.UpdateTagAssignmentRequestToSDK(&updateTagAssignmentReq)
		if err != nil {
			return err
		}
		response, err := w.TagAssignments.UpdateTagAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := tagassignments.TagAssignmentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateTagAssignmentOverrides {
		fn(cmd, &updateTagAssignmentReq)
	}

	return cmd
}

// end service WorkspaceEntityTagAssignments
