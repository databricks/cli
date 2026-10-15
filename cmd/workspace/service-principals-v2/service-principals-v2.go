// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package service_principals_v2

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	scim "github.com/databricks/cli/libs/sdktypes/scim/v1"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-principals-v2",
		Short: `*Public Preview* Identities for use with jobs, automated tools, and systems such as scripts, apps, and CI/CD platforms.`,
		Long: `This command is in Public Preview and may change without notice.

Identities for use with jobs, automated tools, and systems such as scripts,
  apps, and CI/CD platforms. Databricks recommends creating service principals
  to run production jobs or modify production data. If all processes that act on
  production data run with service principals, interactive users do not need any
  write, delete, or modify privileges in production. This eliminates the risk of
  a user overwriting production data by accident.`,
		GroupID: "iam",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	*scim.CreateServicePrincipalRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq scim.CreateServicePrincipalRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createReq.Active, "active", `If this user is active.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.ApplicationId, "application-id", `UUID relating to the service principal.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.DisplayName, "display-name", `String that represents a concatenation of given and family names.`)
	// TODO: array: entitlements
	flags.StringPtrVar(cmd.Flags(), &createReq.ExternalId, "external-id", ``)
	// TODO: array: groups
	flags.StringPtrVar(cmd.Flags(), &createReq.Id, "id", `Databricks service principal ID.`)
	// TODO: array: roles
	// TODO: array: schemas

	cmd.Use = "create"
	cmd.Short = `*Public Preview* Create a service principal.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a service principal.

  Creates a new service principal in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
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

		sdkReq, err := scim.CreateServicePrincipalRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Scim.CreateServicePrincipal(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := scim.ServicePrincipalFromSDK(response)
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
	*sdkscim.DeleteServicePrincipalRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkscim.DeleteServicePrincipalRequest

	cmd.Use = "delete ID"
	cmd.Short = `*Public Preview* Delete a service principal.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a service principal.

  Delete a single service principal in the Databricks workspace.

  Arguments:
    ID: Unique ID for a service principal in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteReq.Id = new(args[0])

		sdkReq := &deleteReq
		err = w.Scim.DeleteServicePrincipal(ctx, *sdkReq)
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
	*sdkscim.GetServicePrincipalRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkscim.GetServicePrincipalRequest

	cmd.Use = "get ID"
	cmd.Short = `*Public Preview* Get service principal details.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get service principal details.

  Gets the details for a single service principal define in the Databricks
  workspace.

  Arguments:
    ID: Unique ID for a service principal in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getReq.Id = new(args[0])

		sdkReq := &getReq
		response, err := w.Scim.GetServicePrincipal(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := scim.ServicePrincipalFromSDK(response)
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
	*scim.ListServicePrincipalsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq scim.ListServicePrincipalsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.StringPtrVar(cmd.Flags(), &listReq.Attributes, "attributes", `Comma-separated list of attributes to return in response.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.ExcludedAttributes, "excluded-attributes", `Comma-separated list of attributes to exclude in response.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.Filter, "filter", `Query by which the results have to be filtered.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.SortBy, "sort-by", `Attribute to sort the results.`)
	flags.EnumVar(cmd.Flags(), &listReq.SortOrder, "sort-order", `The order to sort the results. Supported values: [ascending, descending]`, "ListSortOrder", "ascending", "descending")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.Int64PtrVar(cmd.Flags(), &listReq.StartIndex, "start-index", `Specifies the index of the first result.`)
	cmd.Flags().Lookup("start-index").Hidden = true
	flags.Int64PtrVar(cmd.Flags(), &listReq.Count, "count", `Number of results per API page.`)
	cmd.Flags().Lookup("count").Hidden = true

	cmd.Use = "list"
	cmd.Short = `*Public Preview* List service principals.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List service principals.

  Gets the set of service principals associated with a Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := scim.ListServicePrincipalsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Scim.ListServicePrincipalsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, scim.ServicePrincipalFromSDK))
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
	*scim.PatchServicePrincipalRequest,
)

func newPatch() *cobra.Command {
	cmd := &cobra.Command{}
	var patchReq scim.PatchServicePrincipalRequest

	var patchJson flags.JsonFlag

	cmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: Operations
	// TODO: array: schemas

	cmd.Use = "patch ID"
	cmd.Short = `*Public Preview* Update service principal details.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update service principal details.

  Partially updates the details of a single service principal in the Databricks
  workspace.

  Arguments:
    ID: Unique ID in the Databricks workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := patchJson.Unmarshal(&patchReq)
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
		patchReq.Id = new(args[0])

		sdkReq, err := scim.PatchServicePrincipalRequestToSDK(&patchReq)
		if err != nil {
			return err
		}
		err = w.Scim.PatchServicePrincipal(ctx, *sdkReq)
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
	*scim.UpdateServicePrincipalRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq scim.UpdateServicePrincipalRequest

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &updateReq.Active, "active", `If this user is active.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.ApplicationId, "application-id", `UUID relating to the service principal.`)
	flags.StringPtrVar(cmd.Flags(), &updateReq.DisplayName, "display-name", `String that represents a concatenation of given and family names.`)
	// TODO: array: entitlements
	flags.StringPtrVar(cmd.Flags(), &updateReq.ExternalId, "external-id", ``)
	// TODO: array: groups
	// TODO: array: roles
	// TODO: array: schemas

	cmd.Use = "update ID"
	cmd.Short = `*Public Preview* Replace service principal.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Replace service principal.

  Updates the details of a single service principal.

  This action replaces the existing service principal with the same name.

  Arguments:
    ID: Databricks service principal ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

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
		updateReq.Id = new(args[0])

		sdkReq, err := scim.UpdateServicePrincipalRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		err = w.Scim.UpdateServicePrincipal(ctx, *sdkReq)
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
