// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package volumes

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
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
}

// start create command

var createReq catalog.CreateVolumeRequestContent

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `The comment attached to the volume.`)
	createCmd.Flags().StringVar(&createReq.StorageLocation, "storage-location", createReq.StorageLocation, `The storage location on the cloud.`)

}

var createCmd = &cobra.Command{
	Use:   "create CATALOG_NAME NAME SCHEMA_NAME VOLUME_TYPE",
	Short: `Create a Volume.`,
	Long: `Create a Volume.
  
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
  tables, nor volumes, or catalogs or schemas.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(4),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		createReq.CatalogName = args[0]
		createReq.Name = args[1]
		createReq.SchemaName = args[2]
		_, err = fmt.Sscan(args[3], &createReq.VolumeType)
		if err != nil {
			return fmt.Errorf("invalid VOLUME_TYPE: %s", args[3])
		}

		response, err := w.Volumes.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq catalog.DeleteVolumeRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete FULL_NAME_ARG",
	Short: `Delete a Volume.`,
	Long: `Delete a Volume.
  
  Deletes a volume from the specified parent catalog and schema.
  
  The caller must be a metastore admin or an owner of the volume. For the latter
  case, the caller must also be the owner or have the **USE_CATALOG** privilege
  on the parent catalog and the **USE_SCHEMA** privilege on the parent schema.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Volumes.VolumeInfoNameToVolumeIdMap(ctx, catalog.ListVolumesRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "The three-level (fully qualified) name of the volume")
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
	},
}

// start list command

var listReq catalog.ListVolumesRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

}

var listCmd = &cobra.Command{
	Use:   "list CATALOG_NAME SCHEMA_NAME",
	Short: `List Volumes.`,
	Long: `List Volumes.
  
  Gets an array of all volumes for the current metastore under the parent
  catalog and schema.
  
  The returned volumes are filtered based on the privileges of the calling user.
  For example, the metastore admin is able to list all the volumes. A regular
  user needs to be the owner or have the **READ VOLUME** privilege on the volume
  to recieve the volumes in the response. For the latter case, the caller must
  also be the owner or have the **USE_CATALOG** privilege on the parent catalog
  and the **USE_SCHEMA** privilege on the parent schema.
  
  There is no guarantee of a specific ordering of the elements in the array.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		listReq.CatalogName = args[0]
		listReq.SchemaName = args[1]

		response, err := w.Volumes.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start read command

var readReq catalog.ReadVolumeRequest

func init() {
	Cmd.AddCommand(readCmd)
	// TODO: short flags

}

var readCmd = &cobra.Command{
	Use:   "read FULL_NAME_ARG",
	Short: `Get a Volume.`,
	Long: `Get a Volume.
  
  Gets a volume from the metastore for a specific catalog and schema.
  
  The caller must be a metastore admin or an owner of (or have the **READ
  VOLUME** privilege on) the volume. For the latter case, the caller must also
  be the owner or have the **USE_CATALOG** privilege on the parent catalog and
  the **USE_SCHEMA** privilege on the parent schema.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Volumes.VolumeInfoNameToVolumeIdMap(ctx, catalog.ListVolumesRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "The three-level (fully qualified) name of the volume")
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
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq catalog.UpdateVolumeRequestContent

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `The comment attached to the volume.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name of the volume.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The identifier of the user who owns the volume.`)

}

var updateCmd = &cobra.Command{
	Use:   "update FULL_NAME_ARG",
	Short: `Update a Volume.`,
	Long: `Update a Volume.
  
  Updates the specified volume under the specified parent catalog and schema.
  
  The caller must be a metastore admin or an owner of the volume. For the latter
  case, the caller must also be the owner or have the **USE_CATALOG** privilege
  on the parent catalog and the **USE_SCHEMA** privilege on the parent schema.
  
  Currently only the name, the owner or the comment of the volume could be
  updated.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Volumes.VolumeInfoNameToVolumeIdMap(ctx, catalog.ListVolumesRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "The three-level (fully qualified) name of the volume")
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
		return ui.Render(cmd, response)
	},
}

// end service Volumes
