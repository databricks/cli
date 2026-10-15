// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package providers

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	sharing "github.com/databricks/cli/libs/sdktypes/sharing/v1"
	sdksharing "github.com/databricks/sdk-go/sharing/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*sharing.CreateProviderRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq sharing.CreateProviderRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createReq.Comment, "comment", `Description about the provider.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.RecipientProfileStr, "recipient-profile-str", `This field is required when the __authentication_type__ is **TOKEN**, **OAUTH_CLIENT_CREDENTIALS** or not provided.`)

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'authentication_type' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createReq.AuthenticationType = sharing.DeltaSharingAuthenticationType(args[1])
		}

		sdkReq, err := sharing.CreateProviderRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Sharing.CreateProvider(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.ProviderInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdksharing.DeleteProviderRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdksharing.DeleteProviderRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a provider.`
	cmd.Long = `Delete a provider.

  Deletes an authentication provider, if the caller is a metastore admin or is
  the owner of the provider.

  Arguments:
    NAME: Name of the provider.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No NAME argument specified. Loading names for Providers drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Sharing.ListProvidersIter(ctx, sdksharing.ListProvidersRequest{}), []string{"Name"}, []string{"MetastoreId"})
			sp.Close()
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
			return errors.New("expected to have name of the provider")
		}
		deleteReq.NameArg = new(args[0])

		sdkReq := &deleteReq
		response, err := w.Sharing.DeleteProvider(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*sdksharing.GetProviderRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdksharing.GetProviderRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a provider.`
	cmd.Long = `Get a provider.

  Gets a specific authentication provider. The caller must supply the name of
  the provider, and must either be a metastore admin or the owner of the
  provider.

  Arguments:
    NAME: Name of the provider.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No NAME argument specified. Loading names for Providers drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Sharing.ListProvidersIter(ctx, sdksharing.ListProvidersRequest{}), []string{"Name"}, []string{"MetastoreId"})
			sp.Close()
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
			return errors.New("expected to have name of the provider")
		}
		getReq.NameArg = new(args[0])

		sdkReq := &getReq
		response, err := w.Sharing.GetProvider(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.ProviderInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.StringPtrVar(cmd.Flags(), &listReq.DataProviderGlobalMetastoreId, "data-provider-global-metastore-id", `If not provided, all providers will be returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Maximum number of providers to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List providers.`
	cmd.Long = `List providers.

  Gets an array of available authentication providers. The caller must either be
  a metastore admin, have the **USE_PROVIDER** privilege on the providers, or be
  the owner of the providers. Providers not owned by the caller and for which
  the caller does not have the **USE_PROVIDER** privilege are not included in
  the response. There is no guarantee of a specific ordering of the elements in
  the array.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := sharing.ListProvidersRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Sharing.ListProvidersIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, sharing.ProviderInfoFromSDK))
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

	flags.IntPtrVar(cmd.Flags(), &listProviderShareAssetsReq.FunctionMaxResults, "function-max-results", `Maximum number of functions to return.`)
	flags.IntPtrVar(cmd.Flags(), &listProviderShareAssetsReq.NotebookMaxResults, "notebook-max-results", `Maximum number of notebooks to return.`)
	flags.IntPtrVar(cmd.Flags(), &listProviderShareAssetsReq.TableMaxResults, "table-max-results", `Maximum number of tables to return.`)
	flags.IntPtrVar(cmd.Flags(), &listProviderShareAssetsReq.VolumeMaxResults, "volume-max-results", `Maximum number of volumes to return.`)

	cmd.Use = "list-provider-share-assets PROVIDER_NAME SHARE_NAME"
	cmd.Short = `List assets by provider share.`
	cmd.Long = `List assets by provider share.

  Get arrays of assets associated with a specified provider's share. The caller
  is the recipient of the share.

  Arguments:
    PROVIDER_NAME: The name of the provider who owns the share.
    SHARE_NAME: The name of the share.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listProviderShareAssetsReq.ProviderNameArg = new(args[0])
		listProviderShareAssetsReq.ShareNameArg = new(args[1])

		sdkReq, err := sharing.ListProviderShareAssetsRequestToSDK(&listProviderShareAssetsReq)
		if err != nil {
			return err
		}
		response, err := w.Sharing.ListProviderShareAssets(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.ListProviderShareAssetsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sharing.ListProviderSharesRequest,
)

func newListShares() *cobra.Command {
	cmd := &cobra.Command{}
	var listSharesReq sharing.ListProviderSharesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSharesLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSharesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSharesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listSharesReq.MaxResults, "max-results", `Maximum number of shares to return.`)
	cmd.Flags().Lookup("max-results").Hidden = true

	cmd.Use = "list-shares NAME"
	cmd.Short = `List shares by Provider.`
	cmd.Long = `List shares by Provider.

  Gets an array of a specified provider's shares within the metastore where:

  * the caller is a metastore admin, or * the caller is the owner.

  Arguments:
    NAME: Name of the provider in which to list shares.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No NAME argument specified. Loading names for Providers drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Sharing.ListProvidersIter(ctx, sdksharing.ListProvidersRequest{}), []string{"Name"}, []string{"MetastoreId"})
			sp.Close()
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
			return errors.New("expected to have name of the provider in which to list shares")
		}
		listSharesReq.ProviderNameArg = new(args[0])

		sdkReq, err := sharing.ListProviderSharesRequestToSDK(&listSharesReq)
		if err != nil {
			return err
		}
		if listSharesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSharesLimit)
		}
		if listSharesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSharesLimit)
		}
		response := w.Sharing.ListProviderSharesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, sharing.ProviderShareFromSDK))
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
	*sharing.UpdateProviderRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq sharing.UpdateProviderRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateReq.Comment, "comment", `Description about the provider.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.NewName, "new-name", `New name for the provider.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.Owner, "owner", `Username of Provider owner.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.RecipientProfileStr, "recipient-profile-str", `This field is required when the __authentication_type__ is **TOKEN**, **OAUTH_CLIENT_CREDENTIALS** or not provided.`)

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No NAME argument specified. Loading names for Providers drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Sharing.ListProvidersIter(ctx, sdksharing.ListProvidersRequest{}), []string{"Name"}, []string{"MetastoreId"})
			sp.Close()
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
			return errors.New("expected to have name of the provider")
		}
		updateReq.NameArg = new(args[0])

		sdkReq, err := sharing.UpdateProviderRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Sharing.UpdateProvider(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := sharing.ProviderInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
