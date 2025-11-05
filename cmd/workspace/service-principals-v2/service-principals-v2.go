// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package service_principals_v2

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-principals-v2",
		Short: `Identities for use with jobs, automated tools, and systems such as scripts, apps, and CI/CD platforms.`,
		Long: `Identities for use with jobs, automated tools, and systems such as scripts,
  apps, and CI/CD platforms. Databricks recommends creating service principals
  to run production jobs or modify production data. If all processes that act on
  production data run with service principals, interactive users do not need any
  write, delete, or modify privileges in production. This eliminates the risk of
  a user overwriting production data by accident.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newPatch())
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
	*iam.CreateServicePrincipalRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq iam.CreateServicePrincipalRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createReq.Active, "active", createReq.Active, `If this user is active. Wire name: 'active'.`)
	cmd.Flags().StringVar(&createReq.ApplicationId, "application-id", createReq.ApplicationId, `UUID relating to the service principal. Wire name: 'applicationId'.`)
	cmd.Flags().StringVar(&createReq.DisplayName, "display-name", createReq.DisplayName, `String that represents a concatenation of given and family names. Wire name: 'displayName'.`)
	// TODO: array: entitlements
	cmd.Flags().StringVar(&createReq.ExternalId, "external-id", createReq.ExternalId, `Wire name: 'externalId'.`)
	// TODO: array: groups
	cmd.Flags().StringVar(&createReq.Id, "id", createReq.Id, `Databricks service principal ID. Wire name: 'id'.`)
	// TODO: array: roles
	// TODO: array: schemas

	cmd.Use = "create"
	cmd.Short = `Create a service principal.`
	cmd.Long = `Create a service principal.
  
  Creates a new service principal in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
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

		response, err := w.ServicePrincipalsV2.Create(ctx, createReq)
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
	*iam.DeleteServicePrincipalRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq iam.DeleteServicePrincipalRequest

	cmd.Use = "delete ID"
	cmd.Short = `Delete a service principal.`
	cmd.Long = `Delete a service principal.
  
  Delete a single service principal in the Databricks workspace.

  Arguments:
    ID: Unique ID for a service principal in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Id = args[0]

		err = w.ServicePrincipalsV2.Delete(ctx, deleteReq)
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
	*iam.GetServicePrincipalRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq iam.GetServicePrincipalRequest

	cmd.Use = "get ID"
	cmd.Short = `Get service principal details.`
	cmd.Long = `Get service principal details.
  
  Gets the details for a single service principal define in the Databricks
  workspace.

  Arguments:
    ID: Unique ID for a service principal in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Id = args[0]

		response, err := w.ServicePrincipalsV2.Get(ctx, getReq)
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
	*iam.ListServicePrincipalsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq iam.ListServicePrincipalsRequest

	cmd.Flags().StringVar(&listReq.Attributes, "attributes", listReq.Attributes, `Comma-separated list of attributes to return in response. Wire name: 'attributes'.`)
	cmd.Flags().Int64Var(&listReq.Count, "count", listReq.Count, `Desired number of results per page. Wire name: 'count'.`)
	cmd.Flags().StringVar(&listReq.ExcludedAttributes, "excluded-attributes", listReq.ExcludedAttributes, `Comma-separated list of attributes to exclude in response. Wire name: 'excludedAttributes'.`)
	cmd.Flags().StringVar(&listReq.Filter, "filter", listReq.Filter, `Query by which the results have to be filtered. Wire name: 'filter'.`)
	cmd.Flags().StringVar(&listReq.SortBy, "sort-by", listReq.SortBy, `Attribute to sort the results. Wire name: 'sortBy'.`)
	cmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order to sort the results. Supported values: [ascending, descending]. Wire name: 'sortOrder'.`)
	cmd.Flags().Int64Var(&listReq.StartIndex, "start-index", listReq.StartIndex, `Specifies the index of the first result. Wire name: 'startIndex'.`)

	cmd.Use = "list"
	cmd.Short = `List service principals.`
	cmd.Long = `List service principals.
  
  Gets the set of service principals associated with a Databricks workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ServicePrincipalsV2.List(ctx, listReq)
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

// start patch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchOverrides []func(
	*cobra.Command,
	*iam.PatchServicePrincipalRequest,
)

func newPatch() *cobra.Command {
	cmd := &cobra.Command{}

	var patchReq iam.PatchServicePrincipalRequest
	var patchJson flags.JsonFlag

	cmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: Operations
	// TODO: array: schemas

	cmd.Use = "patch ID"
	cmd.Short = `Update service principal details.`
	cmd.Long = `Update service principal details.
  
  Partially updates the details of a single service principal in the Databricks
  workspace.

  Arguments:
    ID: Unique ID in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := patchJson.Unmarshal(&patchReq)
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
		patchReq.Id = args[0]

		err = w.ServicePrincipalsV2.Patch(ctx, patchReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchOverrides {
		fn(cmd, &patchReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*iam.UpdateServicePrincipalRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq iam.UpdateServicePrincipalRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateReq.Active, "active", updateReq.Active, `If this user is active. Wire name: 'active'.`)
	cmd.Flags().StringVar(&updateReq.ApplicationId, "application-id", updateReq.ApplicationId, `UUID relating to the service principal. Wire name: 'applicationId'.`)
	cmd.Flags().StringVar(&updateReq.DisplayName, "display-name", updateReq.DisplayName, `String that represents a concatenation of given and family names. Wire name: 'displayName'.`)
	// TODO: array: entitlements
	cmd.Flags().StringVar(&updateReq.ExternalId, "external-id", updateReq.ExternalId, `Wire name: 'externalId'.`)
	// TODO: array: groups
	// TODO: array: roles
	// TODO: array: schemas

	cmd.Use = "update ID"
	cmd.Short = `Replace service principal.`
	cmd.Long = `Replace service principal.
  
  Updates the details of a single service principal.
  
  This action replaces the existing service principal with the same name.

  Arguments:
    ID: Databricks service principal ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

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
		updateReq.Id = args[0]

		err = w.ServicePrincipalsV2.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
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

// end service ServicePrincipalsV2
