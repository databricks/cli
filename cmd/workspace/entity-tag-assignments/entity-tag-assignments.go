// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package entity_tag_assignments

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
		Use:   "entity-tag-assignments",
		Short: `Tags are attributes that include keys and optional values that you can use to organize and categorize entities in Unity Catalog.`,
		Long: `Tags are attributes that include keys and optional values that you can use to
  organize and categorize entities in Unity Catalog. Entity tagging is currently
  supported on catalogs, schemas, tables (including views), columns, volumes.
  With these APIs, users can create, update, delete, and list tag assignments
  across Unity Catalog entities`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*catalog.CreateEntityTagAssignmentRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateEntityTagAssignmentRequest
	createReq.TagAssignment = catalog.EntityTagAssignment{}
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.TagAssignment.TagValue, "tag-value", createReq.TagAssignment.TagValue, `The value of the tag. Wire name: 'tag_value'.`)

	cmd.Use = "create ENTITY_NAME TAG_KEY ENTITY_TYPE"
	cmd.Short = `Create an entity tag assignment.`
	cmd.Long = `Create an entity tag assignment.
  
  Creates a tag assignment for an Unity Catalog entity.
  
  To add tags to Unity Catalog entities, you must own the entity or have the
  following privileges: - **APPLY TAG** on the entity - **USE SCHEMA** on the
  entity's parent schema - **USE CATALOG** on the entity's parent catalog
  
  To add a governed tag to Unity Catalog entities, you must also have the
  **ASSIGN** or **MANAGE** permission on the tag policy. See [Manage tag policy
  permissions].
  
  [Manage tag policy permissions]: https://docs.databricks.com/aws/en/admin/tag-policies/manage-permissions

  Arguments:
    ENTITY_NAME: The fully qualified name of the entity to which the tag is assigned
    TAG_KEY: The key of the tag
    ENTITY_TYPE: The type of the entity to which the tag is assigned. Allowed values are:
      catalogs, schemas, tables, columns, volumes.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'entity_name', 'tag_key', 'entity_type' in your JSON input")
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
			diags := createJson.Unmarshal(&createReq.TagAssignment)
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
			createReq.TagAssignment.EntityName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createReq.TagAssignment.TagKey = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createReq.TagAssignment.EntityType = args[2]
		}

		response, err := w.EntityTagAssignments.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*catalog.DeleteEntityTagAssignmentRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteEntityTagAssignmentRequest

	cmd.Use = "delete ENTITY_TYPE ENTITY_NAME TAG_KEY"
	cmd.Short = `Delete an entity tag assignment.`
	cmd.Long = `Delete an entity tag assignment.
  
  Deletes a tag assignment for an Unity Catalog entity by its key.
  
  To delete tags from Unity Catalog entities, you must own the entity or have
  the following privileges: - **APPLY TAG** on the entity - **USE_SCHEMA** on
  the entity's parent schema - **USE_CATALOG** on the entity's parent catalog
  
  To delete a governed tag from Unity Catalog entities, you must also have the
  **ASSIGN** or **MANAGE** permission on the tag policy. See [Manage tag policy
  permissions].
  
  [Manage tag policy permissions]: https://docs.databricks.com/aws/en/admin/tag-policies/manage-permissions

  Arguments:
    ENTITY_TYPE: The type of the entity to which the tag is assigned. Allowed values are:
      catalogs, schemas, tables, columns, volumes.
    ENTITY_NAME: The fully qualified name of the entity to which the tag is assigned
    TAG_KEY: Required. The key of the tag to delete`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.EntityType = args[0]
		deleteReq.EntityName = args[1]
		deleteReq.TagKey = args[2]

		err = w.EntityTagAssignments.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*catalog.GetEntityTagAssignmentRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetEntityTagAssignmentRequest

	cmd.Use = "get ENTITY_TYPE ENTITY_NAME TAG_KEY"
	cmd.Short = `Get an entity tag assignment.`
	cmd.Long = `Get an entity tag assignment.
  
  Gets a tag assignment for an Unity Catalog entity by tag key.

  Arguments:
    ENTITY_TYPE: The type of the entity to which the tag is assigned. Allowed values are:
      catalogs, schemas, tables, columns, volumes.
    ENTITY_NAME: The fully qualified name of the entity to which the tag is assigned
    TAG_KEY: Required. The key of the tag`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.EntityType = args[0]
		getReq.EntityName = args[1]
		getReq.TagKey = args[2]

		response, err := w.EntityTagAssignments.Get(ctx, getReq)
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
	*catalog.ListEntityTagAssignmentsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListEntityTagAssignmentsRequest

	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Optional. Wire name: 'max_results'.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Optional. Wire name: 'page_token'.`)

	cmd.Use = "list ENTITY_TYPE ENTITY_NAME"
	cmd.Short = `List entity tag assignments.`
	cmd.Long = `List entity tag assignments.
  
  List tag assignments for an Unity Catalog entity

  Arguments:
    ENTITY_TYPE: The type of the entity to which the tag is assigned. Allowed values are:
      catalogs, schemas, tables, columns, volumes.
    ENTITY_NAME: The fully qualified name of the entity to which the tag is assigned`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.EntityType = args[0]
		listReq.EntityName = args[1]

		response := w.EntityTagAssignments.List(ctx, listReq)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateEntityTagAssignmentRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateEntityTagAssignmentRequest
	updateReq.TagAssignment = catalog.EntityTagAssignment{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.TagAssignment.TagValue, "tag-value", updateReq.TagAssignment.TagValue, `The value of the tag. Wire name: 'tag_value'.`)

	cmd.Use = "update ENTITY_TYPE ENTITY_NAME TAG_KEY UPDATE_MASK"
	cmd.Short = `Update an entity tag assignment.`
	cmd.Long = `Update an entity tag assignment.
  
  Updates an existing tag assignment for an Unity Catalog entity.
  
  To update tags to Unity Catalog entities, you must own the entity or have the
  following privileges: - **APPLY TAG** on the entity - **USE SCHEMA** on the
  entity's parent schema - **USE CATALOG** on the entity's parent catalog
  
  To update a governed tag to Unity Catalog entities, you must also have the
  **ASSIGN** or **MANAGE** permission on the tag policy. See [Manage tag policy
  permissions].
  
  [Manage tag policy permissions]: https://docs.databricks.com/aws/en/admin/tag-policies/manage-permissions

  Arguments:
    ENTITY_TYPE: The type of the entity to which the tag is assigned. Allowed values are:
      catalogs, schemas, tables, columns, volumes.
    ENTITY_NAME: The fully qualified name of the entity to which the tag is assigned
    TAG_KEY: The key of the tag
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
			diags := updateJson.Unmarshal(&updateReq.TagAssignment)
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
		updateReq.EntityType = args[0]
		updateReq.EntityName = args[1]
		updateReq.TagKey = args[2]
		updateReq.UpdateMask = args[3]

		response, err := w.EntityTagAssignments.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service EntityTagAssignments
