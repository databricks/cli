// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package metastores

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "metastores",
	Short: `These APIs manage Unity Catalog metastores for an account.`,
	Long: `These APIs manage Unity Catalog metastores for an account. A metastore
  contains catalogs that can be associated with workspaces`,
}

// start create command

var createReq catalog.CreateMetastore
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Region, "region", createReq.Region, `Cloud region which the metastore serves (e.g., us-west-2, westus).`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME STORAGE_ROOT",
	Short: `Create metastore.`,
	Long: `Create metastore.
  
  Creates a Unity Catalog metastore.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.Name = args[0]
			createReq.StorageRoot = args[1]
		}

		response, err := a.Metastores.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteAccountMetastoreRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete METASTORE_ID",
	Short: `Delete a metastore.`,
	Long: `Delete a metastore.
  
  Deletes a Unity Catalog metastore for an account, both specified by ID.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			deleteReq.MetastoreId = args[0]
		}

		err = a.Metastores.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetAccountMetastoreRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get METASTORE_ID",
	Short: `Get a metastore.`,
	Long: `Get a metastore.
  
  Gets a Unity Catalog metastore from an account, both specified by ID.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			getReq.MetastoreId = args[0]
		}

		response, err := a.Metastores.Get(ctx, getReq)
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
	Short: `Get all metastores associated with an account.`,
	Long: `Get all metastores associated with an account.
  
  Gets all Unity Catalog metastores associated with an account specified by ID.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.Metastores.List(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.UpdateMetastore
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.DeltaSharingOrganizationName, "delta-sharing-organization-name", updateReq.DeltaSharingOrganizationName, `The organization name of a Delta Sharing entity, to be used in Databricks-to-Databricks Delta Sharing as the official name.`)
	updateCmd.Flags().Int64Var(&updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, "delta-sharing-recipient-token-lifetime-in-seconds", updateReq.DeltaSharingRecipientTokenLifetimeInSeconds, `The lifetime of delta sharing recipient token in seconds.`)
	updateCmd.Flags().Var(&updateReq.DeltaSharingScope, "delta-sharing-scope", `The scope of Delta Sharing enabled for the metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The user-specified name of the metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `The owner of the metastore.`)
	updateCmd.Flags().StringVar(&updateReq.PrivilegeModelVersion, "privilege-model-version", updateReq.PrivilegeModelVersion, `Privilege model version of the metastore, of the form major.minor (e.g., 1.0).`)
	updateCmd.Flags().StringVar(&updateReq.StorageRootCredentialId, "storage-root-credential-id", updateReq.StorageRootCredentialId, `UUID of storage credential to access the metastore storage_root.`)

}

var updateCmd = &cobra.Command{
	Use:   "update METASTORE_ID",
	Short: `Update a metastore.`,
	Long: `Update a metastore.
  
  Updates an existing Unity Catalog metastore.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			updateReq.MetastoreId = args[0]
		}

		response, err := a.Metastores.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service AccountMetastores
