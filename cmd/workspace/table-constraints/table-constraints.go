// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package table_constraints

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
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
}

// start create command

var createReq catalog.CreateTableConstraint
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a table constraint.`,
	Long: `Create a table constraint.
  
  Creates a new table constraint.
  
  For the table constraint creation to succeed, the user must satisfy both of
  these conditions: - the user must have the **USE_CATALOG** privilege on the
  table's parent catalog, the **USE_SCHEMA** privilege on the table's parent
  schema, and be the owner of the table. - if the new constraint is a
  __ForeignKeyConstraint__, the user must have the **USE_CATALOG** privilege on
  the referenced parent table's catalog, the **USE_SCHEMA** privilege on the
  referenced parent table's schema, and be the owner of the referenced parent
  table.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.FullNameArg = args[0]
			_, err = fmt.Sscan(args[1], &createReq.Constraint)
			if err != nil {
				return fmt.Errorf("invalid CONSTRAINT: %s", args[1])
			}
		}

		response, err := w.TableConstraints.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteTableConstraintRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete FULL_NAME CONSTRAINT_NAME CASCADE",
	Short: `Delete a table constraint.`,
	Long: `Delete a table constraint.
  
  Deletes a table constraint.
  
  For the table constraint deletion to succeed, the user must satisfy both of
  these conditions: - the user must have the **USE_CATALOG** privilege on the
  table's parent catalog, the **USE_SCHEMA** privilege on the table's parent
  schema, and be the owner of the table. - if __cascade__ argument is **true**,
  the user must have the following permissions on all of the child tables: the
  **USE_CATALOG** privilege on the table's catalog, the **USE_SCHEMA** privilege
  on the table's schema, and be the owner of the table.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(3)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			deleteReq.FullName = args[0]
			deleteReq.ConstraintName = args[1]
			_, err = fmt.Sscan(args[2], &deleteReq.Cascade)
			if err != nil {
				return fmt.Errorf("invalid CASCADE: %s", args[2])
			}
		}

		err = w.TableConstraints.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service TableConstraints
