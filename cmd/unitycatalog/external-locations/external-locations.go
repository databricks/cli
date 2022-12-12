// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package external_locations

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "external-locations",
	Short: `An external location is an object that combines a cloud storage path with a storage credential that authorizes access to the cloud storage path.`,
	Long: `An external location is an object that combines a cloud storage path with a
  storage credential that authorizes access to the cloud storage path. Each
  storage location is subject to Unity Catalog access-control policies that
  control which users and groups can access the credential. If a user does not
  have access to a storage location in Unity Catalog, the request fails and
  Unity Catalog does not attempt to authenticate to your cloud tenant on the
  user’s behalf.
  
  Databricks recommends using external locations rather than using storage
  credentials directly.
  
  To create external locations, you must be a metastore admin or a user with the
  CREATE EXTERNAL LOCATION privilege.`,
}

// start create command

var createReq unitycatalog.CreateExternalLocation

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME URL CREDENTIAL_NAME",
	Short: `Create an external location.`,
	Long: `Create an external location.
  
  Creates a new External Location entry in the Metastore. The caller must be a
  Metastore admin or have the CREATE EXTERNAL LOCATION privilege on the
  Metastore.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		createReq.Name = args[0]
		createReq.Url = args[1]
		createReq.CredentialName = args[2]

		response, err := w.ExternalLocations.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq unitycatalog.DeleteExternalLocationRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if there are dependent external tables or mounts.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete an external location.`,
	Long: `Delete an external location.
  
  Deletes the specified external location from the Metastore. The caller must be
  the owner of the external location.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteReq.Name = args[0]

		err = w.ExternalLocations.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq unitycatalog.GetExternalLocationRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get an external location.`,
	Long: `Get an external location.
  
  Gets an external location from the Metastore. The caller must be either a
  Metastore admin, the owner of the external location, or has an appropriate
  privilege level on the Metastore.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getReq.Name = args[0]

		response, err := w.ExternalLocations.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
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
  
  Gets an array of External Locations (ExternalLocationInfo objects) from the
  Metastore. The caller must be a Metastore admin, is the owner of the external
  location, or has privileges to access the external location.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.ExternalLocations.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq unitycatalog.UpdateExternalLocation

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	updateCmd.Flags().StringVar(&updateReq.CredentialName, "credential-name", updateReq.CredentialName, `Current name of the Storage Credential this location uses.`)
	updateCmd.Flags().BoolVar(&updateReq.Force, "force", updateReq.Force, `Force update even if changing url invalidates dependent external tables or mounts.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of the External Location.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The owner of the External Location.`)
	updateCmd.Flags().StringVar(&updateReq.Url, "url", updateReq.Url, `Path URL of the External Location.`)

}

var updateCmd = &cobra.Command{
	Use:   "update NAME",
	Short: `Update an external location.`,
	Long: `Update an external location.
  
  Updates an external location in the Metastore. The caller must be the owner of
  the externa location, or be a Metastore admin. In the second case, the admin
  can only update the name of the external location.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		updateReq.Name = args[0]

		err = w.ExternalLocations.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service ExternalLocations
