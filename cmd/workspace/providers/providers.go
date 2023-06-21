// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package providers

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "providers",
	Short: `Databricks Providers REST API.`,
	Long:  `Databricks Providers REST API`,
	Annotations: map[string]string{
		"package": "sharing",
	},
}

// start create command

var createReq sharing.CreateProvider
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Description about the provider.`)
	createCmd.Flags().StringVar(&createReq.RecipientProfileStr, "recipient-profile-str", createReq.RecipientProfileStr, `This field is required when the __authentication_type__ is **TOKEN** or not provided.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME AUTHENTICATION_TYPE",
	Short: `Create an auth provider.`,
	Long: `Create an auth provider.
  
  Creates a new authentication provider minimally based on a name and
  authentication type. The caller must be an admin on the metastore.`,

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
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.Name = args[0]
			_, err = fmt.Sscan(args[1], &createReq.AuthenticationType)
			if err != nil {
				return fmt.Errorf("invalid AUTHENTICATION_TYPE: %s", args[1])
			}
		}

		response, err := w.Providers.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start delete command

var deleteReq sharing.DeleteProviderRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete a provider.`,
	Long: `Delete a provider.
  
  Deletes an authentication provider, if the caller is a metastore admin or is
  the owner of the provider.`,

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
				promptSpinner <- "No NAME argument specified. Loading names for Providers drop-down."
				names, err := w.Providers.ProviderInfoNameToMetastoreIdMap(ctx, sharing.ListProvidersRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Providers drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Name of the provider")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have name of the provider")
			}
			deleteReq.Name = args[0]
		}

		err = w.Providers.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start get command

var getReq sharing.GetProviderRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get a provider.`,
	Long: `Get a provider.
  
  Gets a specific authentication provider. The caller must supply the name of
  the provider, and must either be a metastore admin or the owner of the
  provider.`,

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
				promptSpinner <- "No NAME argument specified. Loading names for Providers drop-down."
				names, err := w.Providers.ProviderInfoNameToMetastoreIdMap(ctx, sharing.ListProvidersRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Providers drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Name of the provider")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have name of the provider")
			}
			getReq.Name = args[0]
		}

		response, err := w.Providers.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list command

var listReq sharing.ListProvidersRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().StringVar(&listReq.DataProviderGlobalMetastoreId, "data-provider-global-metastore-id", listReq.DataProviderGlobalMetastoreId, `If not provided, all providers will be returned.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List providers.`,
	Long: `List providers.
  
  Gets an array of available authentication providers. The caller must either be
  a metastore admin or the owner of the providers. Providers not owned by the
  caller are not included in the response. There is no guarantee of a specific
  ordering of the elements in the array.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
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
		}

		response, err := w.Providers.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start list-shares command

var listSharesReq sharing.ListSharesRequest
var listSharesJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listSharesCmd)
	// TODO: short flags
	listSharesCmd.Flags().Var(&listSharesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var listSharesCmd = &cobra.Command{
	Use:   "list-shares NAME",
	Short: `List shares by Provider.`,
	Long: `List shares by Provider.
  
  Gets an array of a specified provider's shares within the metastore where:
  
  * the caller is a metastore admin, or * the caller is the owner.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listSharesJson.Unmarshal(&listSharesReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No NAME argument specified. Loading names for Providers drop-down."
				names, err := w.Providers.ProviderInfoNameToMetastoreIdMap(ctx, sharing.ListProvidersRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Providers drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "Name of the provider in which to list shares")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have name of the provider in which to list shares")
			}
			listSharesReq.Name = args[0]
		}

		response, err := w.Providers.ListSharesAll(ctx, listSharesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// start update command

var updateReq sharing.UpdateProvider
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Description about the provider.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name of the Provider.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of Provider owner.`)
	updateCmd.Flags().StringVar(&updateReq.RecipientProfileStr, "recipient-profile-str", updateReq.RecipientProfileStr, `This field is required when the __authentication_type__ is **TOKEN** or not provided.`)

}

var updateCmd = &cobra.Command{
	Use:   "update NAME",
	Short: `Update a provider.`,
	Long: `Update a provider.
  
  Updates the information for an authentication provider, if the caller is a
  metastore admin or is the owner of the provider. If the update changes the
  provider name, the caller must be both a metastore admin and the owner of the
  provider.`,

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
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No NAME argument specified. Loading names for Providers drop-down."
				names, err := w.Providers.ProviderInfoNameToMetastoreIdMap(ctx, sharing.ListProvidersRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Providers drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The name of the Provider")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the name of the provider")
			}
			updateReq.Name = args[0]
		}

		response, err := w.Providers.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	ValidArgsFunction: cobra.NoFileCompletions,
}

// end service Providers
