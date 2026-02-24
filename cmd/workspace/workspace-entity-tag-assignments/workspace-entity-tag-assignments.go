// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_entity_tag_assignments

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
		Use:     "workspace-entity-tag-assignments",
		Short:   `Manage tag assignments on workspace-scoped objects.`,
		Long:    `Manage tag assignments on workspace-scoped objects.`,
		GroupID: "tags",
		RunE:    root.ReportUnknownSubcommand,
	}

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
	*tags.CreateTagAssignmentRequest,
)

func newCreateTagAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var createTagAssignmentReq tags.CreateTagAssignmentRequest
	createTagAssignmentReq.TagAssignment = tags.TagAssignment{}
	var createTagAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&createTagAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createTagAssignmentReq.TagAssignment.TagValue, "tag-value", createTagAssignmentReq.TagAssignment.TagValue, `The value of the tag.`)

	cmd.Use = "create-tag-assignment ENTITY_TYPE ENTITY_ID TAG_KEY"
	cmd.Short = `Create a tag assignment for an entity.`
	cmd.Long = `Create a tag assignment for an entity.

  Create a tag assignment

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name
    TAG_KEY: The key of the tag. The characters , . : / - = and leading/trailing spaces
      are not allowed`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'entity_type', 'entity_id', 'tag_key' in your JSON input")
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
			createTagAssignmentReq.TagAssignment.EntityType = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createTagAssignmentReq.TagAssignment.EntityId = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createTagAssignmentReq.TagAssignment.TagKey = args[2]
		}

		response, err := w.WorkspaceEntityTagAssignments.CreateTagAssignment(ctx, createTagAssignmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*tags.DeleteTagAssignmentRequest,
)

func newDeleteTagAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteTagAssignmentReq tags.DeleteTagAssignmentRequest

	cmd.Use = "delete-tag-assignment ENTITY_TYPE ENTITY_ID TAG_KEY"
	cmd.Short = `Delete a tag assignment for an entity.`
	cmd.Long = `Delete a tag assignment for an entity.

  Delete a tag assignment

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name
    TAG_KEY: The key of the tag. The characters , . : / - = and leading/trailing spaces
      are not allowed`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteTagAssignmentReq.EntityType = args[0]
		deleteTagAssignmentReq.EntityId = args[1]
		deleteTagAssignmentReq.TagKey = args[2]

		err = w.WorkspaceEntityTagAssignments.DeleteTagAssignment(ctx, deleteTagAssignmentReq)
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
	*tags.GetTagAssignmentRequest,
)

func newGetTagAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var getTagAssignmentReq tags.GetTagAssignmentRequest

	cmd.Use = "get-tag-assignment ENTITY_TYPE ENTITY_ID TAG_KEY"
	cmd.Short = `Get a tag assignment for an entity.`
	cmd.Long = `Get a tag assignment for an entity.

  Get a tag assignment

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name
    TAG_KEY: The key of the tag. The characters , . : / - = and leading/trailing spaces
      are not allowed`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getTagAssignmentReq.EntityType = args[0]
		getTagAssignmentReq.EntityId = args[1]
		getTagAssignmentReq.TagKey = args[2]

		response, err := w.WorkspaceEntityTagAssignments.GetTagAssignment(ctx, getTagAssignmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*tags.ListTagAssignmentsRequest,
)

func newListTagAssignments() *cobra.Command {
	cmd := &cobra.Command{}

	var listTagAssignmentsReq tags.ListTagAssignmentsRequest

	cmd.Flags().IntVar(&listTagAssignmentsReq.PageSize, "page-size", listTagAssignmentsReq.PageSize, `Optional.`)
	cmd.Flags().StringVar(&listTagAssignmentsReq.PageToken, "page-token", listTagAssignmentsReq.PageToken, `Pagination token to go to the next page of tag assignments.`)

	cmd.Use = "list-tag-assignments ENTITY_TYPE ENTITY_ID"
	cmd.Short = `List tag assignments for an entity.`
	cmd.Long = `List tag assignments for an entity.

  List the tag assignments for an entity

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces
    ENTITY_ID: The identifier of the entity to which the tag is assigned. For apps, the
      entity_id is the app name`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listTagAssignmentsReq.EntityType = args[0]
		listTagAssignmentsReq.EntityId = args[1]

		response := w.WorkspaceEntityTagAssignments.ListTagAssignments(ctx, listTagAssignmentsReq)
		return cmdio.RenderIterator(ctx, response)
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
	*tags.UpdateTagAssignmentRequest,
)

func newUpdateTagAssignment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateTagAssignmentReq tags.UpdateTagAssignmentRequest
	updateTagAssignmentReq.TagAssignment = tags.TagAssignment{}
	var updateTagAssignmentJson flags.JsonFlag

	cmd.Flags().Var(&updateTagAssignmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateTagAssignmentReq.TagAssignment.TagValue, "tag-value", updateTagAssignmentReq.TagAssignment.TagValue, `The value of the tag.`)

	cmd.Use = "update-tag-assignment ENTITY_TYPE ENTITY_ID TAG_KEY UPDATE_MASK"
	cmd.Short = `Update a tag assignment for an entity.`
	cmd.Long = `Update a tag assignment for an entity.

  Update a tag assignment

  Arguments:
    ENTITY_TYPE: The type of entity to which the tag is assigned. Allowed values are apps,
      dashboards, geniespaces
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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
		updateTagAssignmentReq.EntityType = args[0]
		updateTagAssignmentReq.EntityId = args[1]
		updateTagAssignmentReq.TagKey = args[2]
		updateTagAssignmentReq.UpdateMask = args[3]

		response, err := w.WorkspaceEntityTagAssignments.UpdateTagAssignment(ctx, updateTagAssignmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
