// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package providers

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "providers",
	Short: `Databricks Delta Sharing: Providers REST API.`,
	Long:  `Databricks Delta Sharing: Providers REST API`,
}

// start create command

var createReq unitycatalog.CreateProvider
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Description about the provider.`)
	createCmd.Flags().StringVar(&createReq.Owner, "owner", createReq.Owner, `Username of Provider owner.`)
	// TODO: complex arg: recipient_profile
	createCmd.Flags().StringVar(&createReq.RecipientProfileStr, "recipient-profile-str", createReq.RecipientProfileStr, `This field is only present when the authentication type is TOKEN.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create an auth provider.`,
	Long: `Create an auth provider.
  
  Creates a new authentication provider minimally based on a name and
  authentication type. The caller must be an admin on the Metastore.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Providers.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq unitycatalog.DeleteProviderRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete a provider.`,
	Long: `Delete a provider.
  
  Deletes an authentication provider, if the caller is a Metastore admin or is
  the owner of the provider.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		deleteReq.Name = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Providers.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq unitycatalog.GetProviderRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get a provider.`,
	Long: `Get a provider.
  
  Gets a specific authentication provider. The caller must supply the name of
  the provider, and must either be a Metastore admin or the owner of the
  provider.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		getReq.Name = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Providers.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq unitycatalog.ListProvidersRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.DataProviderGlobalMetastoreId, "data-provider-global-metastore-id", listReq.DataProviderGlobalMetastoreId, `If not provided, all providers will be returned.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List providers.`,
	Long: `List providers.
  
  Gets an array of available authentication providers. The caller must either be
  a Metastore admin or the owner of the providers. Providers not owned by the
  caller are not included in the response.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Providers.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list-shares command

var listSharesReq unitycatalog.ListSharesRequest

func init() {
	Cmd.AddCommand(listSharesCmd)
	// TODO: short flags

}

var listSharesCmd = &cobra.Command{
	Use:   "list-shares NAME",
	Short: `List shares.`,
	Long: `List shares.
  
  Gets an array of all shares within the Metastore where:
  
  * the caller is a Metastore admin, or * the caller is the owner.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		listSharesReq.Name = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Providers.ListShares(ctx, listSharesReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq unitycatalog.UpdateProvider
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().Var(&updateReq.AuthenticationType, "authentication-type", `The delta sharing authentication type.`)
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Description about the provider.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of Provider owner.`)
	// TODO: complex arg: recipient_profile
	updateCmd.Flags().StringVar(&updateReq.RecipientProfileStr, "recipient-profile-str", updateReq.RecipientProfileStr, `This field is only present when the authentication type is TOKEN.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a provider.`,
	Long: `Update a provider.
  
  Updates the information for an authentication provider, if the caller is a
  Metastore admin or is the owner of the provider. If the update changes the
  provider name, the caller must be both a Metastore admin and the owner of the
  provider.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Providers.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Providers
