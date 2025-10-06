// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package providers

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: `A data provider is an object representing the organization in the real world who shares the data.`,
		Long: `A data provider is an object representing the organization in the real world
  who shares the data. A provider contains shares which further contain the
  shared data.`,
		GroupID: "sharing",
		Annotations: map[string]string{
			"package": "sharing",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListProviderShareAssets())
	cmd.AddCommand(newListShares())
	cmd.AddCommand(newUpdate())

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
	*sharing.CreateProvider,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sharing.CreateProvider
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Description about the provider.`)
	cmd.Flags().StringVar(&createReq.RecipientProfileStr, "recipient-profile-str", createReq.RecipientProfileStr, `This field is required when the __authentication_type__ is **TOKEN**, **OAUTH_CLIENT_CREDENTIALS** or not provided.`)

	cmd.Use = "create NAME AUTHENTICATION_TYPE"
	cmd.Short = `Create an auth provider.`
	cmd.Long = `Create an auth provider.
  
  Creates a new authentication provider minimally based on a name and
  authentication type. The caller must be an admin on the metastore.

  Arguments:
    NAME: The name of the Provider.
    AUTHENTICATION_TYPE:  
      Supported values: [DATABRICKS, OAUTH_CLIENT_CREDENTIALS, OIDC_FEDERATION, TOKEN]`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'authentication_type' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
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

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sharing.DeleteProviderRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sharing.DeleteProviderRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a provider.`
	cmd.Long = `Delete a provider.
  
  Deletes an authentication provider, if the caller is a metastore admin or is
  the owner of the provider.

  Arguments:
    NAME: Name of the provider.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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

		err = w.Providers.Delete(ctx, deleteReq)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sharing.GetProviderRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sharing.GetProviderRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a provider.`
	cmd.Long = `Get a provider.
  
  Gets a specific authentication provider. The caller must supply the name of
  the provider, and must either be a metastore admin or the owner of the
  provider.

  Arguments:
    NAME: Name of the provider.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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

		response, err := w.Providers.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*sharing.ListProvidersRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sharing.ListProvidersRequest

	cmd.Flags().StringVar(&listReq.DataProviderGlobalMetastoreId, "data-provider-global-metastore-id", listReq.DataProviderGlobalMetastoreId, `If not provided, all providers will be returned.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of providers to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List providers.`
	cmd.Long = `List providers.
  
  Gets an array of available authentication providers. The caller must either be
  a metastore admin or the owner of the providers. Providers not owned by the
  caller are not included in the response. There is no guarantee of a specific
  ordering of the elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Providers.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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

// start list-provider-share-assets command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listProviderShareAssetsOverrides []func(
	*cobra.Command,
	*sharing.ListProviderShareAssetsRequest,
)

func newListProviderShareAssets() *cobra.Command {
	cmd := &cobra.Command{}

	var listProviderShareAssetsReq sharing.ListProviderShareAssetsRequest

	cmd.Flags().IntVar(&listProviderShareAssetsReq.FunctionMaxResults, "function-max-results", listProviderShareAssetsReq.FunctionMaxResults, `Maximum number of functions to return.`)
	cmd.Flags().IntVar(&listProviderShareAssetsReq.NotebookMaxResults, "notebook-max-results", listProviderShareAssetsReq.NotebookMaxResults, `Maximum number of notebooks to return.`)
	cmd.Flags().IntVar(&listProviderShareAssetsReq.TableMaxResults, "table-max-results", listProviderShareAssetsReq.TableMaxResults, `Maximum number of tables to return.`)
	cmd.Flags().IntVar(&listProviderShareAssetsReq.VolumeMaxResults, "volume-max-results", listProviderShareAssetsReq.VolumeMaxResults, `Maximum number of volumes to return.`)

	cmd.Use = "list-provider-share-assets PROVIDER_NAME SHARE_NAME"
	cmd.Short = `List assets by provider share.`
	cmd.Long = `List assets by provider share.
  
  Get arrays of assets associated with a specified provider's share. The caller
  is the recipient of the share.

  Arguments:
    PROVIDER_NAME: The name of the provider who owns the share.
    SHARE_NAME: The name of the share.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listProviderShareAssetsReq.ProviderName = args[0]
		listProviderShareAssetsReq.ShareName = args[1]

		response, err := w.Providers.ListProviderShareAssets(ctx, listProviderShareAssetsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listProviderShareAssetsOverrides {
		fn(cmd, &listProviderShareAssetsReq)
	}

	return cmd
}

// start list-shares command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSharesOverrides []func(
	*cobra.Command,
	*sharing.ListSharesRequest,
)

func newListShares() *cobra.Command {
	cmd := &cobra.Command{}

	var listSharesReq sharing.ListSharesRequest

	cmd.Flags().IntVar(&listSharesReq.MaxResults, "max-results", listSharesReq.MaxResults, `Maximum number of shares to return.`)
	cmd.Flags().StringVar(&listSharesReq.PageToken, "page-token", listSharesReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list-shares NAME"
	cmd.Short = `List shares by Provider.`
	cmd.Long = `List shares by Provider.
  
  Gets an array of a specified provider's shares within the metastore where:
  
  * the caller is a metastore admin, or * the caller is the owner.

  Arguments:
    NAME: Name of the provider in which to list shares.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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

		response := w.Providers.ListShares(ctx, listSharesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSharesOverrides {
		fn(cmd, &listSharesReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*sharing.UpdateProvider,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sharing.UpdateProvider
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Description about the provider.`)
	cmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `New name for the provider.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of Provider owner.`)
	cmd.Flags().StringVar(&updateReq.RecipientProfileStr, "recipient-profile-str", updateReq.RecipientProfileStr, `This field is required when the __authentication_type__ is **TOKEN**, **OAUTH_CLIENT_CREDENTIALS** or not provided.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update a provider.`
	cmd.Long = `Update a provider.
  
  Updates the information for an authentication provider, if the caller is a
  metastore admin or is the owner of the provider. If the update changes the
  provider name, the caller must be both a metastore admin and the owner of the
  provider.

  Arguments:
    NAME: Name of the provider.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
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
		updateReq.Name = args[0]

		response, err := w.Providers.Update(ctx, updateReq)
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

// end service Providers
