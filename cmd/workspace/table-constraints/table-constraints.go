// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package table_constraints

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
		Use:   "table-constraints",
		Short: `Primary key and foreign key constraints encode relationships between fields in tables.`,
		Long: `Primary key and foreign key constraints encode relationships between fields in
  tables.
  
  Primary and foreign keys are informational only and are not enforced. Foreign
  keys must reference a primary key in another table. This primary key is the
  parent constraint of the foreign key and the table this primary key is on is
  the parent table of the foreign key. Similarly, the foreign key is the child
  constraint of its referenced primary key; the table of the foreign key is the
  child table of the primary key.
  
  You can declare primary keys and foreign keys as part of the table
  specification during table creation. You can also add or drop constraints on
  existing tables.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())

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
	*catalog.CreateTableConstraint,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateTableConstraint
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a table constraint.`
	cmd.Long = `Create a table constraint.
  
  Creates a new table constraint.
  
  For the table constraint creation to succeed, the user must satisfy both of
  these conditions: - the user must have the **USE_CATALOG** privilege on the
  table's parent catalog, the **USE_SCHEMA** privilege on the table's parent
  schema, and be the owner of the table. - if the new constraint is a
  __ForeignKeyConstraint__, the user must have the **USE_CATALOG** privilege on
  the referenced parent table's catalog, the **USE_SCHEMA** privilege on the
  referenced parent table's schema, and be the owner of the referenced parent
  table.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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

		response, err := w.TableConstraints.Create(ctx, createReq)
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
	*catalog.DeleteTableConstraintRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteTableConstraintRequest

	cmd.Use = "delete FULL_NAME CONSTRAINT_NAME CASCADE"
	cmd.Short = `Delete a table constraint.`
	cmd.Long = `Delete a table constraint.
  
  Deletes a table constraint.
  
  For the table constraint deletion to succeed, the user must satisfy both of
  these conditions: - the user must have the **USE_CATALOG** privilege on the
  table's parent catalog, the **USE_SCHEMA** privilege on the table's parent
  schema, and be the owner of the table. - if __cascade__ argument is **true**,
  the user must have the following permissions on all of the child tables: the
  **USE_CATALOG** privilege on the table's catalog, the **USE_SCHEMA** privilege
  on the table's schema, and be the owner of the table.

  Arguments:
    FULL_NAME: Full name of the table referenced by the constraint.
    CONSTRAINT_NAME: The name of the constraint to delete.
    CASCADE: If true, try deleting all child constraints of the current constraint. If
      false, reject this operation if the current constraint has any child
      constraints.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.FullName = args[0]
		deleteReq.ConstraintName = args[1]
		_, err = fmt.Sscan(args[2], &deleteReq.Cascade)
		if err != nil {
			return fmt.Errorf("invalid CASCADE: %s", args[2])
		}

		err = w.TableConstraints.Delete(ctx, deleteReq)
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

// end service TableConstraints
