// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package volumes

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volumes",
		Short: `Volumes are a Unity Catalog (UC) capability for accessing, storing, governing, organizing and processing files.`,
		Long: `Volumes are a Unity Catalog (UC) capability for accessing, storing, governing,
  organizing and processing files. Use cases include running machine learning on
  unstructured data such as image, audio, video, or PDF files, organizing data
  sets during the data exploration stages in data science, working with
  libraries that require access to the local file system on cluster machines,
  storing library and config files of arbitrary formats such as .whl or .txt
  centrally and providing secure access across workspaces to it, or transforming
  and querying non-tabular data files in ETL.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
	}

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
	*catalog.CreateVolumeRequestContent,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateVolumeRequestContent

	// TODO: short flags

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `The comment attached to the volume.`)
	cmd.Flags().StringVar(&createReq.StorageLocation, "storage-location", createReq.StorageLocation, `The storage location on the cloud.`)

	cmd.Use = "create CATALOG_NAME SCHEMA_NAME NAME VOLUME_TYPE"
	cmd.Short = `Create a Volume.`
	cmd.Long = `Create a Volume.
  
  Creates a new volume.
  
  The user could create either an external volume or a managed volume. An
  external volume will be created in the specified external location, while a
  managed volume will be located in the default location which is specified by
  the parent schema, or the parent catalog, or the Metastore.
  
  For the volume creation to succeed, the user must satisfy following
  conditions: - The caller must be a metastore admin, or be the owner of the
  parent catalog and schema, or have the **USE_CATALOG** privilege on the parent
  catalog and the **USE_SCHEMA** privilege on the parent schema. - The caller
  must have **CREATE VOLUME** privilege on the parent schema.
  
  For an external volume, following conditions also need to satisfy - The caller
  must have **CREATE EXTERNAL VOLUME** privilege on the external location. -
  There are no other tables, nor volumes existing in the specified storage
  location. - The specified storage location is not under the location of other
  tables, nor volumes, or catalogs or schemas.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		createReq.CatalogName = args[0]
		createReq.SchemaName = args[1]
		createReq.Name = args[2]
		_, err = fmt.Sscan(args[3], &createReq.VolumeType)
		if err != nil {
			return fmt.Errorf("invalid VOLUME_TYPE: %s", args[3])
		}

		response, err := w.Volumes.Create(ctx, createReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreate())
	})
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*catalog.DeleteVolumeRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteVolumeRequest

	// TODO: short flags

	cmd.Use = "delete FULL_NAME_ARG"
	cmd.Short = `Delete a Volume.`
	cmd.Long = `Delete a Volume.
  
  Deletes a volume from the specified parent catalog and schema.
  
  The caller must be a metastore admin or an owner of the volume. For the latter
  case, the caller must also be the owner or have the **USE_CATALOG** privilege
  on the parent catalog and the **USE_SCHEMA** privilege on the parent schema.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FULL_NAME_ARG argument specified. Loading names for Volumes drop-down."
			names, err := w.Volumes.VolumeInfoNameToVolumeIdMap(ctx, catalog.ListVolumesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Volumes drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The three-level (fully qualified) name of the volume")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the three-level (fully qualified) name of the volume")
		}
		deleteReq.FullNameArg = args[0]

		err = w.Volumes.Delete(ctx, deleteReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDelete())
	})
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*catalog.ListVolumesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListVolumesRequest

	// TODO: short flags

	cmd.Use = "list CATALOG_NAME SCHEMA_NAME"
	cmd.Short = `List Volumes.`
	cmd.Long = `List Volumes.
  
  Gets an array of all volumes for the current metastore under the parent
  catalog and schema.
  
  The returned volumes are filtered based on the privileges of the calling user.
  For example, the metastore admin is able to list all the volumes. A regular
  user needs to be the owner or have the **READ VOLUME** privilege on the volume
  to recieve the volumes in the response. For the latter case, the caller must
  also be the owner or have the **USE_CATALOG** privilege on the parent catalog
  and the **USE_SCHEMA** privilege on the parent schema.
  
  There is no guarantee of a specific ordering of the elements in the array.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		listReq.CatalogName = args[0]
		listReq.SchemaName = args[1]

		response, err := w.Volumes.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newList())
	})
}

// start read command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var readOverrides []func(
	*cobra.Command,
	*catalog.ReadVolumeRequest,
)

func newRead() *cobra.Command {
	cmd := &cobra.Command{}

	var readReq catalog.ReadVolumeRequest

	// TODO: short flags

	cmd.Use = "read FULL_NAME_ARG"
	cmd.Short = `Get a Volume.`
	cmd.Long = `Get a Volume.
  
  Gets a volume from the metastore for a specific catalog and schema.
  
  The caller must be a metastore admin or an owner of (or have the **READ
  VOLUME** privilege on) the volume. For the latter case, the caller must also
  be the owner or have the **USE_CATALOG** privilege on the parent catalog and
  the **USE_SCHEMA** privilege on the parent schema.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FULL_NAME_ARG argument specified. Loading names for Volumes drop-down."
			names, err := w.Volumes.VolumeInfoNameToVolumeIdMap(ctx, catalog.ListVolumesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Volumes drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The three-level (fully qualified) name of the volume")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the three-level (fully qualified) name of the volume")
		}
		readReq.FullNameArg = args[0]

		response, err := w.Volumes.Read(ctx, readReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range readOverrides {
		fn(cmd, &readReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRead())
	})
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateVolumeRequestContent,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateVolumeRequestContent

	// TODO: short flags

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `The comment attached to the volume.`)
	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name of the volume.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The identifier of the user who owns the volume.`)

	cmd.Use = "update FULL_NAME_ARG"
	cmd.Short = `Update a Volume.`
	cmd.Long = `Update a Volume.
  
  Updates the specified volume under the specified parent catalog and schema.
  
  The caller must be a metastore admin or an owner of the volume. For the latter
  case, the caller must also be the owner or have the **USE_CATALOG** privilege
  on the parent catalog and the **USE_SCHEMA** privilege on the parent schema.
  
  Currently only the name, the owner or the comment of the volume could be
  updated.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FULL_NAME_ARG argument specified. Loading names for Volumes drop-down."
			names, err := w.Volumes.VolumeInfoNameToVolumeIdMap(ctx, catalog.ListVolumesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Volumes drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The three-level (fully qualified) name of the volume")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the three-level (fully qualified) name of the volume")
		}
		updateReq.FullNameArg = args[0]

		response, err := w.Volumes.Update(ctx, updateReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdate())
	})
}

// end service Volumes
