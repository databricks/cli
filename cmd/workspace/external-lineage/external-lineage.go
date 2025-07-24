// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package external_lineage

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
		Use:   "external-lineage",
		Short: `External Lineage APIs enable defining and managing lineage relationships between Databricks objects and external systems.`,
		Long: `External Lineage APIs enable defining and managing lineage relationships
  between Databricks objects and external systems. These APIs allow users to
  capture data flows connecting Databricks tables, models, and file paths with
  external metadata objects.
  
  With these APIs, users can create, update, delete, and list lineage
  relationships with support for column-level mappings and custom properties.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateExternalLineageRelationship())
	cmd.AddCommand(newDeleteExternalLineageRelationship())
	cmd.AddCommand(newListExternalLineageRelationships())
	cmd.AddCommand(newUpdateExternalLineageRelationship())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-external-lineage-relationship command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createExternalLineageRelationshipOverrides []func(
	*cobra.Command,
	*catalog.CreateExternalLineageRelationshipRequest,
)

func newCreateExternalLineageRelationship() *cobra.Command {
	cmd := &cobra.Command{}

	var createExternalLineageRelationshipReq catalog.CreateExternalLineageRelationshipRequest
	createExternalLineageRelationshipReq.ExternalLineageRelationship = catalog.CreateRequestExternalLineage{}
	var createExternalLineageRelationshipJson flags.JsonFlag

	cmd.Flags().Var(&createExternalLineageRelationshipJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: columns
	// TODO: map via StringToStringVar: properties

	cmd.Use = "create-external-lineage-relationship SOURCE TARGET"
	cmd.Short = `Create an external lineage relationship.`
	cmd.Long = `Create an external lineage relationship.
  
  Creates an external lineage relationship between a Databricks or external
  metadata object and another external metadata object.

  Arguments:
    SOURCE: Source object of the external lineage relationship.
    TARGET: Target object of the external lineage relationship.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'source', 'target' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createExternalLineageRelationshipJson.Unmarshal(&createExternalLineageRelationshipReq.ExternalLineageRelationship)
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
			_, err = fmt.Sscan(args[0], &createExternalLineageRelationshipReq.ExternalLineageRelationship.Source)
			if err != nil {
				return fmt.Errorf("invalid SOURCE: %s", args[0])
			}
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createExternalLineageRelationshipReq.ExternalLineageRelationship.Target)
			if err != nil {
				return fmt.Errorf("invalid TARGET: %s", args[1])
			}
		}

		response, err := w.ExternalLineage.CreateExternalLineageRelationship(ctx, createExternalLineageRelationshipReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createExternalLineageRelationshipOverrides {
		fn(cmd, &createExternalLineageRelationshipReq)
	}

	return cmd
}

// start delete-external-lineage-relationship command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteExternalLineageRelationshipOverrides []func(
	*cobra.Command,
	*catalog.DeleteExternalLineageRelationshipRequest,
)

func newDeleteExternalLineageRelationship() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteExternalLineageRelationshipReq catalog.DeleteExternalLineageRelationshipRequest
	var deleteExternalLineageRelationshipJson flags.JsonFlag

	cmd.Flags().Var(&deleteExternalLineageRelationshipJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-external-lineage-relationship"
	cmd.Short = `Delete an external lineage relationship.`
	cmd.Long = `Delete an external lineage relationship.
  
  Deletes an external lineage relationship between a Databricks or external
  metadata object and another external metadata object.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := deleteExternalLineageRelationshipJson.Unmarshal(&deleteExternalLineageRelationshipReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = w.ExternalLineage.DeleteExternalLineageRelationship(ctx, deleteExternalLineageRelationshipReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteExternalLineageRelationshipOverrides {
		fn(cmd, &deleteExternalLineageRelationshipReq)
	}

	return cmd
}

// start list-external-lineage-relationships command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listExternalLineageRelationshipsOverrides []func(
	*cobra.Command,
	*catalog.ListExternalLineageRelationshipsRequest,
)

func newListExternalLineageRelationships() *cobra.Command {
	cmd := &cobra.Command{}

	var listExternalLineageRelationshipsReq catalog.ListExternalLineageRelationshipsRequest
	var listExternalLineageRelationshipsJson flags.JsonFlag

	cmd.Flags().Var(&listExternalLineageRelationshipsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&listExternalLineageRelationshipsReq.PageSize, "page-size", listExternalLineageRelationshipsReq.PageSize, ``)
	cmd.Flags().StringVar(&listExternalLineageRelationshipsReq.PageToken, "page-token", listExternalLineageRelationshipsReq.PageToken, ``)

	cmd.Use = "list-external-lineage-relationships"
	cmd.Short = `List external lineage relationships.`
	cmd.Long = `List external lineage relationships.
  
  Lists external lineage relationships of a Databricks object or external
  metadata given a supplied direction.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := listExternalLineageRelationshipsJson.Unmarshal(&listExternalLineageRelationshipsReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response := w.ExternalLineage.ListExternalLineageRelationships(ctx, listExternalLineageRelationshipsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listExternalLineageRelationshipsOverrides {
		fn(cmd, &listExternalLineageRelationshipsReq)
	}

	return cmd
}

// start update-external-lineage-relationship command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateExternalLineageRelationshipOverrides []func(
	*cobra.Command,
	*catalog.UpdateExternalLineageRelationshipRequest,
)

func newUpdateExternalLineageRelationship() *cobra.Command {
	cmd := &cobra.Command{}

	var updateExternalLineageRelationshipReq catalog.UpdateExternalLineageRelationshipRequest
	updateExternalLineageRelationshipReq.ExternalLineageRelationship = catalog.UpdateRequestExternalLineage{}
	var updateExternalLineageRelationshipJson flags.JsonFlag

	cmd.Flags().Var(&updateExternalLineageRelationshipJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: columns
	// TODO: map via StringToStringVar: properties

	cmd.Use = "update-external-lineage-relationship UPDATE_MASK SOURCE TARGET"
	cmd.Short = `Update an external lineage relationship.`
	cmd.Long = `Update an external lineage relationship.
  
  Updates an external lineage relationship between a Databricks or external
  metadata object and another external metadata object.

  Arguments:
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.
      
      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.
    SOURCE: Source object of the external lineage relationship.
    TARGET: Target object of the external lineage relationship.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'source', 'target' in your JSON input")
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
			diags := updateExternalLineageRelationshipJson.Unmarshal(&updateExternalLineageRelationshipReq.ExternalLineageRelationship)
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
		updateExternalLineageRelationshipReq.UpdateMask = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &updateExternalLineageRelationshipReq.ExternalLineageRelationship.Source)
			if err != nil {
				return fmt.Errorf("invalid SOURCE: %s", args[1])
			}
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateExternalLineageRelationshipReq.ExternalLineageRelationship.Target)
			if err != nil {
				return fmt.Errorf("invalid TARGET: %s", args[2])
			}
		}

		response, err := w.ExternalLineage.UpdateExternalLineageRelationship(ctx, updateExternalLineageRelationshipReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateExternalLineageRelationshipOverrides {
		fn(cmd, &updateExternalLineageRelationshipReq)
	}

	return cmd
}

// end service ExternalLineage
