// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package system_schemas

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "system-schemas",
	Short: `A system schema is a schema that lives within the system catalog.`,
	Long: `A system schema is a schema that lives within the system catalog. A system
  schema may contain information about customer usage of Unity Catalog such as
  audit-logs, billing-logs, lineage information, etc.`,
}

// start disable command

var disableReq catalog.DisableRequest
var disableJson flags.JsonFlag

func init() {
	Cmd.AddCommand(disableCmd)
	// TODO: short flags
	disableCmd.Flags().Var(&disableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
			err = disableJson.Unmarshal(&disableReq)
			if err != nil {
				return err
			}
		} else {
			disableReq.MetastoreId = args[0]
			disableReq.SchemaName = args[1]
		}

		err = w.SystemSchemas.Disable(ctx, disableReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start enable command

func init() {
	Cmd.AddCommand(enableCmd)

}

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: `Enable a system schema.`,
	Long: `Enable a system schema.
  
  Enables the system schema and adds it to the system catalog. The caller must
  be an account admin or a metastore admin.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = w.SystemSchemas.Enable(ctx)
		if err != nil {
			return err
		}
		return nil
	},
}

// start list command

var listReq catalog.ListSystemSchemasRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
			listReq.MetastoreId = args[0]
		}

		response, err := w.SystemSchemas.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service SystemSchemas
