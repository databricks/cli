// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package connections

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "connections",
	Short: `Connections allow for creating a connection to an external data source.`,
	Long: `Connections allow for creating a connection to an external data source.
  
  A connection is an abstraction of an external data source that can be
  connected from Databricks Compute. Creating a connection object is the first
  step to managing external data sources within Unity Catalog, with the second
  step being creating a data object (catalog, schema, or table) using the
  connection. Data objects derived from a connection can be written to or read
  from similar to other Unity Catalog data objects based on cloud storage. Users
  may create different types of connections with each connection having a unique
  set of configuration options to support credential management and other
  settings.`,
	Annotations: map[string]string{
		"package": "catalog",
	},

	// This service is being previewed; hide from help output.
	Hidden: true,
}

// start create command

var createReq catalog.CreateConnection
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	createCmd.Flags().StringVar(&createReq.Owner, "owner", createReq.Owner, `Username of current owner of the connection.`)
	// TODO: map via StringToStringVar: properties_kvpairs
	createCmd.Flags().BoolVar(&createReq.ReadOnly, "read-only", createReq.ReadOnly, `If the connection is read only.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a connection.`,
	Long: `Create a connection.
  
  Creates a new connection
  
  Creates a new connection to an external data source. It allows users to
  specify connection details and configurations for interaction with the
  external server.`,

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
			createReq.Name = args[0]
			_, err = fmt.Sscan(args[1], &createReq.ConnectionType)
			if err != nil {
				return fmt.Errorf("invalid CONNECTION_TYPE: %s", args[1])
			}
			_, err = fmt.Sscan(args[2], &createReq.OptionsKvpairs)
			if err != nil {
				return fmt.Errorf("invalid OPTIONS_KVPAIRS: %s", args[2])
			}
		}

		response, err := w.Connections.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteConnectionRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME_ARG",
	Short: `Delete a connection.`,
	Long: `Delete a connection.
  
  Deletes the connection that matches the supplied name.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No NAME_ARG argument specified. Loading names for Connections drop-down."
				names, err := w.Connections.ConnectionInfoNameToFullNameMap(ctx)
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Connections drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The name of the connection to be deleted")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the name of the connection to be deleted")
			}
			deleteReq.NameArg = args[0]
		}

		err = w.Connections.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetConnectionRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get NAME_ARG",
	Short: `Get a connection.`,
	Long: `Get a connection.
  
  Gets a connection from it's name.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No NAME_ARG argument specified. Loading names for Connections drop-down."
				names, err := w.Connections.ConnectionInfoNameToFullNameMap(ctx)
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Connections drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Name of the connection")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have name of the connection")
			}
			getReq.NameArg = args[0]
		}

		response, err := w.Connections.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List connections.`,
	Long: `List connections.
  
  List all connections.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Connections.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.UpdateConnection
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a connection.`,
	Long: `Update a connection.
  
  Updates the connection that matches the supplied name.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			updateReq.Name = args[0]
			_, err = fmt.Sscan(args[1], &updateReq.OptionsKvpairs)
			if err != nil {
				return fmt.Errorf("invalid OPTIONS_KVPAIRS: %s", args[1])
			}
			updateReq.NameArg = args[2]
		}

		response, err := w.Connections.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service Connections
