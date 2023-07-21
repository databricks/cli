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

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "providers",
		Short:   `Databricks Providers REST API.`,
		Long:    `Databricks Providers REST API`,
		GroupID: "sharing",
		Annotations: map[string]string{
			"package": "sharing",
		},
	}

	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
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

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Description about the provider.`)
	cmd.Flags().StringVar(&createReq.RecipientProfileStr, "recipient-profile-str", createReq.RecipientProfileStr, `This field is required when the __authentication_type__ is **TOKEN** or not provided.`)

	cmd.Use = "create NAME AUTHENTICATION_TYPE"
	cmd.Short = `Create an auth provider.`
	cmd.Long = `Create an auth provider.
  
  Creates a new authentication provider minimally based on a name and
  authentication type. The caller must be an admin on the metastore.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
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

	// TODO: short flags

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a provider.`
	cmd.Long = `Delete a provider.
  
  Deletes an authentication provider, if the caller is a metastore admin or is
  the owner of the provider.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

	// TODO: short flags

	cmd.Use = "get NAME"
	cmd.Short = `Get a provider.`
	cmd.Long = `Get a provider.
  
  Gets a specific authentication provider. The caller must supply the name of
  the provider, and must either be a metastore admin or the owner of the
  provider.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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
	var listJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&listReq.DataProviderGlobalMetastoreId, "data-provider-global-metastore-id", listReq.DataProviderGlobalMetastoreId, `If not provided, all providers will be returned.`)

	cmd.Use = "list"
	cmd.Short = `List providers.`
	cmd.Long = `List providers.
  
  Gets an array of available authentication providers. The caller must either be
  a metastore admin or the owner of the providers. Providers not owned by the
  caller are not included in the response. There is no guarantee of a specific
  ordering of the elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
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

	// TODO: short flags

	cmd.Use = "list-shares NAME"
	cmd.Short = `List shares by Provider.`
	cmd.Long = `List shares by Provider.
  
  Gets an array of a specified provider's shares within the metastore where:
  
  * the caller is a metastore admin, or * the caller is the owner.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		listSharesReq.Name = args[0]

		response, err := w.Providers.ListSharesAll(ctx, listSharesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Description about the provider.`)
	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name of the Provider.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of Provider owner.`)
	cmd.Flags().StringVar(&updateReq.RecipientProfileStr, "recipient-profile-str", updateReq.RecipientProfileStr, `This field is required when the __authentication_type__ is **TOKEN** or not provided.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update a provider.`
	cmd.Long = `Update a provider.
  
  Updates the information for an authentication provider, if the caller is a
  metastore admin or is the owner of the provider. If the update changes the
  provider name, the caller must be both a metastore admin and the owner of the
  provider.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			updateReq.Name = args[0]
		}

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
