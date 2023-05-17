// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package external_locations

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "external-locations",
	Short: `An external location is an object that combines a cloud storage path with a storage credential that authorizes access to the cloud storage path.`,
	Long: `An external location is an object that combines a cloud storage path with a
  storage credential that authorizes access to the cloud storage path. Each
  external location is subject to Unity Catalog access-control policies that
  control which users and groups can access the credential. If a user does not
  have access to an external location in Unity Catalog, the request fails and
  Unity Catalog does not attempt to authenticate to your cloud tenant on the
  user’s behalf.
  
  Databricks recommends using external locations rather than using storage
  credentials directly.
  
  To create external locations, you must be a metastore admin or a user with the
  **CREATE_EXTERNAL_LOCATION** privilege.`,
}

// start create command

var createReq catalog.CreateExternalLocation

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	createCmd.Flags().BoolVar(&createReq.ReadOnly, "read-only", createReq.ReadOnly, `Indicates whether the external location is read-only.`)
	createCmd.Flags().BoolVar(&createReq.SkipValidation, "skip-validation", createReq.SkipValidation, `Skips validation of the storage credential associated with the external location.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME URL CREDENTIAL_NAME",
	Short: `Create an external location.`,
	Long: `Create an external location.
  
  Creates a new external location entry in the metastore. The caller must be a
  metastore admin or have the **CREATE_EXTERNAL_LOCATION** privilege on both the
  metastore and the associated storage credential.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		createReq.Name = args[0]
		createReq.Url = args[1]
		createReq.CredentialName = args[2]

		response, err := w.ExternalLocations.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteExternalLocationRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if there are dependent external tables or mounts.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete an external location.`,
	Long: `Delete an external location.
  
  Deletes the specified external location from the metastore. The caller must be
  the owner of the external location.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		deleteReq.Name = args[0]

		err = w.ExternalLocations.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetExternalLocationRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get an external location.`,
	Long: `Get an external location.
  
  Gets an external location from the metastore. The caller must be either a
  metastore admin, the owner of the external location, or a user that has some
  privilege on the external location.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		getReq.Name = args[0]

		response, err := w.ExternalLocations.Get(ctx, getReq)
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
	Short: `List external locations.`,
	Long: `List external locations.
  
  Gets an array of external locations (__ExternalLocationInfo__ objects) from
  the metastore. The caller must be a metastore admin, the owner of the external
  location, or a user that has some privilege on the external location. There is
  no guarantee of a specific ordering of the elements in the array.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.ExternalLocations.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.UpdateExternalLocation

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	updateCmd.Flags().StringVar(&updateReq.CredentialName, "credential-name", updateReq.CredentialName, `Name of the storage credential used with this location.`)
	updateCmd.Flags().BoolVar(&updateReq.Force, "force", updateReq.Force, `Force update even if changing url invalidates dependent external tables or mounts.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of the external location.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The owner of the external location.`)
	updateCmd.Flags().BoolVar(&updateReq.ReadOnly, "read-only", updateReq.ReadOnly, `Indicates whether the external location is read-only.`)
	updateCmd.Flags().StringVar(&updateReq.Url, "url", updateReq.Url, `Path URL of the external location.`)

}

var updateCmd = &cobra.Command{
	Use:   "update NAME",
	Short: `Update an external location.`,
	Long: `Update an external location.
  
  Updates an external location in the metastore. The caller must be the owner of
  the external location, or be a metastore admin. In the second case, the admin
  can only update the name of the external location.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		updateReq.Name = args[0]

		response, err := w.ExternalLocations.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service ExternalLocations
