// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package entity_tag_assignments

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	ucentitytagassignments "github.com/databricks/cli/libs/sdktypes/uc/entitytagassignments/v1"
	sdkucentitytagassignments "github.com/databricks/sdk-go/uc/entitytagassignments/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entity-tag-assignments",
		Short: `Tags are attributes that include keys and optional values that you can use to organize and categorize entities in Unity Catalog.`,
		Long: `Tags are attributes that include keys and optional values that you can use to
  organize and categorize entities in Unity Catalog. Entity tagging is supported
  on catalogs, schemas, tables (including views), columns, and volumes. With
  these APIs, you can create, update, delete, and list tag assignments across
  Unity Catalog entities.`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*ucentitytagassignments.CreateEntityTagAssignmentRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq ucentitytagassignments.CreateEntityTagAssignmentRequest
	createReq.TagAssignment = &ucentitytagassignments.EntityTagAssignment{}

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.TagAssignment.TagValue, "tag-value", `The value of the tag.`)

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
    ENTITY_TYPE: The type of the entity to which the tag is assigned.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'entity_name', 'tag_key', 'entity_type' in your JSON input")
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
			diags := createJson.Unmarshal(&createReq.TagAssignment)
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
			createReq.TagAssignment.EntityName = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createReq.TagAssignment.TagKey = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createReq.TagAssignment.EntityType = new(args[2])
		}

		sdkReq, err := ucentitytagassignments.CreateEntityTagAssignmentRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.EntityTagAssignments.CreateEntityTagAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucentitytagassignments.EntityTagAssignmentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkucentitytagassignments.DeleteEntityTagAssignmentRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkucentitytagassignments.DeleteEntityTagAssignmentRequest

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
    ENTITY_TYPE: The type of the entity to which the tag is assigned.
    ENTITY_NAME: The fully qualified name of the entity to which the tag is assigned
    TAG_KEY: Required. The key of the tag to delete`

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

		deleteReq.EntityType = new(args[0])
		deleteReq.EntityName = new(args[1])
		deleteReq.TagKey = new(args[2])

		sdkReq := &deleteReq
		err = w.EntityTagAssignments.DeleteEntityTagAssignment(ctx, *sdkReq)
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
	*sdkucentitytagassignments.GetEntityTagAssignmentRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkucentitytagassignments.GetEntityTagAssignmentRequest

	cmd.Use = "get ENTITY_TYPE ENTITY_NAME TAG_KEY"
	cmd.Short = `Get an entity tag assignment.`
	cmd.Long = `Get an entity tag assignment.

  Gets a tag assignment for an Unity Catalog entity by tag key.

  Arguments:
    ENTITY_TYPE: The type of the entity to which the tag is assigned.
    ENTITY_NAME: The fully qualified name of the entity to which the tag is assigned
    TAG_KEY: Required. The key of the tag`

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

		getReq.EntityType = new(args[0])
		getReq.EntityName = new(args[1])
		getReq.TagKey = new(args[2])

		sdkReq := &getReq
		response, err := w.EntityTagAssignments.GetEntityTagAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucentitytagassignments.EntityTagAssignmentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*ucentitytagassignments.ListEntityTagAssignmentsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq ucentitytagassignments.ListEntityTagAssignmentsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Optional.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list ENTITY_TYPE ENTITY_NAME"
	cmd.Short = `List entity tag assignments.`
	cmd.Long = `List entity tag assignments.

  List tag assignments for an Unity Catalog entity

  PAGINATION BEHAVIOR: The API is by default paginated, a page may contain zero
  results while still providing a next_page_token. Clients must continue reading
  pages until next_page_token is absent, which is the only indication that the
  end of results has been reached.

  Arguments:
    ENTITY_TYPE: The type of the entity to which the tag is assigned.
    ENTITY_NAME: The fully qualified name of the entity to which the tag is assigned`

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

		listReq.EntityType = new(args[0])
		listReq.EntityName = new(args[1])

		sdkReq, err := ucentitytagassignments.ListEntityTagAssignmentsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.EntityTagAssignments.ListEntityTagAssignmentsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, ucentitytagassignments.EntityTagAssignmentFromSDK))
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
	*ucentitytagassignments.UpdateEntityTagAssignmentRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq ucentitytagassignments.UpdateEntityTagAssignmentRequest
	updateReq.TagAssignment = &ucentitytagassignments.EntityTagAssignment{}

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.TagAssignment.TagValue, "tag-value", `The value of the tag.`)

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
    ENTITY_TYPE: The type of the entity to which the tag is assigned.
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
			diags := updateJson.Unmarshal(&updateReq.TagAssignment)
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
		updateReq.TagAssignment.EntityType = new(args[0])
		updateReq.TagAssignment.EntityName = new(args[1])
		updateReq.TagAssignment.TagKey = new(args[2])
		updateReq.UpdateMask = new(args[3])

		sdkReq, err := ucentitytagassignments.UpdateEntityTagAssignmentRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.EntityTagAssignments.UpdateEntityTagAssignment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := ucentitytagassignments.EntityTagAssignmentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
