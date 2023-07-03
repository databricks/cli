// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package system_schemas

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "system-schemas",
	Short: `A system schema is a schema that lives within the system catalog.`,
	Long: `A system schema is a schema that lives within the system catalog. A system
  schema may contain information about customer usage of Unity Catalog such as
  audit-logs, billing-logs, lineage information, etc.`,
	Annotations: map[string]string{
		"package": "catalog",
	},

	// This service is being previewed; hide from help output.
	Hidden: true,
}

// start disable command
var disableReq catalog.DisableRequest

func init() {
	Cmd.AddCommand(disableCmd)
	// TODO: short flags

}

var disableCmd = &cobra.Command{
	Use:   "disable METASTORE_ID SCHEMA_NAME",
	Short: `Disable a system schema.`,
	Long: `Disable a system schema.
  
  Disables the system schema and removes it from the system catalog. The caller
  must be an account admin or a metastore admin.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		disableReq.MetastoreId = args[0]
		_, err = fmt.Sscan(args[1], &disableReq.SchemaName)
		if err != nil {
			return fmt.Errorf("invalid SCHEMA_NAME: %s", args[1])
		}

		err = w.SystemSchemas.Disable(ctx, disableReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start enable command
var enableReq catalog.EnableRequest

func init() {
	Cmd.AddCommand(enableCmd)
	// TODO: short flags

}

var enableCmd = &cobra.Command{
	Use:   "enable METASTORE_ID SCHEMA_NAME",
	Short: `Enable a system schema.`,
	Long: `Enable a system schema.
  
  Enables the system schema and adds it to the system catalog. The caller must
  be an account admin or a metastore admin.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		enableReq.MetastoreId = args[0]
		_, err = fmt.Sscan(args[1], &enableReq.SchemaName)
		if err != nil {
			return fmt.Errorf("invalid SCHEMA_NAME: %s", args[1])
		}

		err = w.SystemSchemas.Enable(ctx, enableReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command
var listReq catalog.ListSystemSchemasRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

}

var listCmd = &cobra.Command{
	Use:   "list METASTORE_ID",
	Short: `List system schemas.`,
	Long: `List system schemas.
  
  Gets an array of system schemas for a metastore. The caller must be an account
  admin or a metastore admin.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		listReq.MetastoreId = args[0]

		response, err := w.SystemSchemas.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service SystemSchemas
