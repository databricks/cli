// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace_iam_v2

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iamv2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-iam-v2",
		Short: `These APIs are used to manage identities and the workspace access of these identities in <Databricks>.`,
		Long: `These APIs are used to manage identities and the workspace access of these
  identities in <Databricks>.`,
		GroupID: "iamv2",
		Annotations: map[string]string{
			"package": "iamv2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateGroupProxy())
	cmd.AddCommand(newCreateServicePrincipalProxy())
	cmd.AddCommand(newCreateUserProxy())
	cmd.AddCommand(newCreateWorkspaceAccessDetailLocal())
	cmd.AddCommand(newDeleteGroupProxy())
	cmd.AddCommand(newDeleteServicePrincipalProxy())
	cmd.AddCommand(newDeleteUserProxy())
	cmd.AddCommand(newDeleteWorkspaceAccessDetailLocal())
	cmd.AddCommand(newGetGroupProxy())
	cmd.AddCommand(newGetServicePrincipalProxy())
	cmd.AddCommand(newGetUserProxy())
	cmd.AddCommand(newGetWorkspaceAccessDetailLocal())
	cmd.AddCommand(newListGroupsProxy())
	cmd.AddCommand(newListServicePrincipalsProxy())
	cmd.AddCommand(newListUsersProxy())
	cmd.AddCommand(newListWorkspaceAccessDetailsLocal())
	cmd.AddCommand(newResolveGroupProxy())
	cmd.AddCommand(newResolveServicePrincipalProxy())
	cmd.AddCommand(newResolveUserProxy())
	cmd.AddCommand(newUpdateGroupProxy())
	cmd.AddCommand(newUpdateServicePrincipalProxy())
	cmd.AddCommand(newUpdateUserProxy())
	cmd.AddCommand(newUpdateWorkspaceAccessDetailLocal())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createGroupProxyOverrides []func(
	*cobra.Command,
	*iamv2.CreateGroupProxyRequest,
)

func newCreateGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createGroupProxyReq iamv2.CreateGroupProxyRequest
	createGroupProxyReq.Group = iamv2.Group{}
	var createGroupProxyJson flags.JsonFlag

	cmd.Flags().Var(&createGroupProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createGroupProxyReq.Group.ExternalId, "external-id", createGroupProxyReq.Group.ExternalId, `ExternalId of the group in the customer's IdP.`)
	cmd.Flags().StringVar(&createGroupProxyReq.Group.GroupName, "group-name", createGroupProxyReq.Group.GroupName, `Display name of the group.`)

	cmd.Use = "create-group-proxy"
	cmd.Short = `Create a group in the account.`
	cmd.Long = `Create a group in the account.
  
  TODO: Write description later when this method is implemented`

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
			diags := createGroupProxyJson.Unmarshal(&createGroupProxyReq.Group)
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

		response, err := w.WorkspaceIamV2.CreateGroupProxy(ctx, createGroupProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createGroupProxyOverrides {
		fn(cmd, &createGroupProxyReq)
	}

	return cmd
}

// start create-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*iamv2.CreateServicePrincipalProxyRequest,
)

func newCreateServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createServicePrincipalProxyReq iamv2.CreateServicePrincipalProxyRequest
	createServicePrincipalProxyReq.ServicePrincipal = iamv2.ServicePrincipal{}
	var createServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&createServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&createServicePrincipalProxyReq.ServicePrincipal.AccountSpStatus, "account-sp-status", `The activity status of a service principal in a Databricks account. Supported values: [ACTIVE, INACTIVE]`)
	cmd.Flags().StringVar(&createServicePrincipalProxyReq.ServicePrincipal.ApplicationId, "application-id", createServicePrincipalProxyReq.ServicePrincipal.ApplicationId, `Application ID of the service principal.`)
	cmd.Flags().StringVar(&createServicePrincipalProxyReq.ServicePrincipal.DisplayName, "display-name", createServicePrincipalProxyReq.ServicePrincipal.DisplayName, `Display name of the service principal.`)
	cmd.Flags().StringVar(&createServicePrincipalProxyReq.ServicePrincipal.ExternalId, "external-id", createServicePrincipalProxyReq.ServicePrincipal.ExternalId, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "create-service-principal-proxy"
	cmd.Short = `Create a service principal in the account.`
	cmd.Long = `Create a service principal in the account.
  
  TODO: Write description later when this method is implemented`

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
			diags := createServicePrincipalProxyJson.Unmarshal(&createServicePrincipalProxyReq.ServicePrincipal)
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

		response, err := w.WorkspaceIamV2.CreateServicePrincipalProxy(ctx, createServicePrincipalProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createServicePrincipalProxyOverrides {
		fn(cmd, &createServicePrincipalProxyReq)
	}

	return cmd
}

// start create-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createUserProxyOverrides []func(
	*cobra.Command,
	*iamv2.CreateUserProxyRequest,
)

func newCreateUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var createUserProxyReq iamv2.CreateUserProxyRequest
	createUserProxyReq.User = iamv2.User{}
	var createUserProxyJson flags.JsonFlag

	cmd.Flags().Var(&createUserProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&createUserProxyReq.User.AccountUserStatus, "account-user-status", `The activity status of a user in a Databricks account. Supported values: [ACTIVE, INACTIVE]`)
	cmd.Flags().StringVar(&createUserProxyReq.User.ExternalId, "external-id", createUserProxyReq.User.ExternalId, `ExternalId of the user in the customer's IdP.`)
	// TODO: complex arg: name

	cmd.Use = "create-user-proxy USERNAME"
	cmd.Short = `Create a user in the account.`
	cmd.Long = `Create a user in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    USERNAME: Username/email of the user.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'username' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createUserProxyJson.Unmarshal(&createUserProxyReq.User)
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
			createUserProxyReq.User.Username = args[0]
		}

		response, err := w.WorkspaceIamV2.CreateUserProxy(ctx, createUserProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createUserProxyOverrides {
		fn(cmd, &createUserProxyReq)
	}

	return cmd
}

// start create-workspace-access-detail-local command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceAccessDetailLocalOverrides []func(
	*cobra.Command,
	*iamv2.CreateWorkspaceAccessDetailLocalRequest,
)

func newCreateWorkspaceAccessDetailLocal() *cobra.Command {
	cmd := &cobra.Command{}

	var createWorkspaceAccessDetailLocalReq iamv2.CreateWorkspaceAccessDetailLocalRequest
	createWorkspaceAccessDetailLocalReq.WorkspaceAccessDetail = iamv2.WorkspaceAccessDetail{}
	var createWorkspaceAccessDetailLocalJson flags.JsonFlag

	cmd.Flags().Var(&createWorkspaceAccessDetailLocalJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: permissions
	cmd.Flags().Var(&createWorkspaceAccessDetailLocalReq.WorkspaceAccessDetail.Status, "status", `The activity status of the principal in the workspace. Supported values: [ACTIVE, INACTIVE]`)

	cmd.Use = "create-workspace-access-detail-local"
	cmd.Short = `Define workspace access for a principal.`
	cmd.Long = `Define workspace access for a principal.
  
  TODO: Write description later when this method is implemented`

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
			diags := createWorkspaceAccessDetailLocalJson.Unmarshal(&createWorkspaceAccessDetailLocalReq.WorkspaceAccessDetail)
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

		response, err := w.WorkspaceIamV2.CreateWorkspaceAccessDetailLocal(ctx, createWorkspaceAccessDetailLocalReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createWorkspaceAccessDetailLocalOverrides {
		fn(cmd, &createWorkspaceAccessDetailLocalReq)
	}

	return cmd
}

// start delete-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteGroupProxyOverrides []func(
	*cobra.Command,
	*iamv2.DeleteGroupProxyRequest,
)

func newDeleteGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteGroupProxyReq iamv2.DeleteGroupProxyRequest

	cmd.Use = "delete-group-proxy INTERNAL_ID"
	cmd.Short = `Delete a group in the account.`
	cmd.Long = `Delete a group in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the group in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteGroupProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

		err = w.WorkspaceIamV2.DeleteGroupProxy(ctx, deleteGroupProxyReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteGroupProxyOverrides {
		fn(cmd, &deleteGroupProxyReq)
	}

	return cmd
}

// start delete-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*iamv2.DeleteServicePrincipalProxyRequest,
)

func newDeleteServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteServicePrincipalProxyReq iamv2.DeleteServicePrincipalProxyRequest

	cmd.Use = "delete-service-principal-proxy INTERNAL_ID"
	cmd.Short = `Delete a service principal in the account.`
	cmd.Long = `Delete a service principal in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the service principal in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteServicePrincipalProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

		err = w.WorkspaceIamV2.DeleteServicePrincipalProxy(ctx, deleteServicePrincipalProxyReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteServicePrincipalProxyOverrides {
		fn(cmd, &deleteServicePrincipalProxyReq)
	}

	return cmd
}

// start delete-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteUserProxyOverrides []func(
	*cobra.Command,
	*iamv2.DeleteUserProxyRequest,
)

func newDeleteUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteUserProxyReq iamv2.DeleteUserProxyRequest

	cmd.Use = "delete-user-proxy INTERNAL_ID"
	cmd.Short = `Delete a user in the account.`
	cmd.Long = `Delete a user in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the user in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteUserProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

		err = w.WorkspaceIamV2.DeleteUserProxy(ctx, deleteUserProxyReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteUserProxyOverrides {
		fn(cmd, &deleteUserProxyReq)
	}

	return cmd
}

// start delete-workspace-access-detail-local command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceAccessDetailLocalOverrides []func(
	*cobra.Command,
	*iamv2.DeleteWorkspaceAccessDetailLocalRequest,
)

func newDeleteWorkspaceAccessDetailLocal() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWorkspaceAccessDetailLocalReq iamv2.DeleteWorkspaceAccessDetailLocalRequest

	cmd.Use = "delete-workspace-access-detail-local PRINCIPAL_ID"
	cmd.Short = `Delete workspace access for a principal.`
	cmd.Long = `Delete workspace access for a principal.
  
  TODO: Write description later when this method is implemented

  Arguments:
    PRINCIPAL_ID: Required. ID of the principal in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &deleteWorkspaceAccessDetailLocalReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		err = w.WorkspaceIamV2.DeleteWorkspaceAccessDetailLocal(ctx, deleteWorkspaceAccessDetailLocalReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteWorkspaceAccessDetailLocalOverrides {
		fn(cmd, &deleteWorkspaceAccessDetailLocalReq)
	}

	return cmd
}

// start get-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getGroupProxyOverrides []func(
	*cobra.Command,
	*iamv2.GetGroupProxyRequest,
)

func newGetGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getGroupProxyReq iamv2.GetGroupProxyRequest

	cmd.Use = "get-group-proxy INTERNAL_ID"
	cmd.Short = `Get a group in the account.`
	cmd.Long = `Get a group in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the group in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getGroupProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

		response, err := w.WorkspaceIamV2.GetGroupProxy(ctx, getGroupProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getGroupProxyOverrides {
		fn(cmd, &getGroupProxyReq)
	}

	return cmd
}

// start get-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*iamv2.GetServicePrincipalProxyRequest,
)

func newGetServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getServicePrincipalProxyReq iamv2.GetServicePrincipalProxyRequest

	cmd.Use = "get-service-principal-proxy INTERNAL_ID"
	cmd.Short = `Get a service principal in the account.`
	cmd.Long = `Get a service principal in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the service principal in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getServicePrincipalProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

		response, err := w.WorkspaceIamV2.GetServicePrincipalProxy(ctx, getServicePrincipalProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getServicePrincipalProxyOverrides {
		fn(cmd, &getServicePrincipalProxyReq)
	}

	return cmd
}

// start get-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getUserProxyOverrides []func(
	*cobra.Command,
	*iamv2.GetUserProxyRequest,
)

func newGetUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var getUserProxyReq iamv2.GetUserProxyRequest

	cmd.Use = "get-user-proxy INTERNAL_ID"
	cmd.Short = `Get a user in the account.`
	cmd.Long = `Get a user in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the user in Databricks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getUserProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}

		response, err := w.WorkspaceIamV2.GetUserProxy(ctx, getUserProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getUserProxyOverrides {
		fn(cmd, &getUserProxyReq)
	}

	return cmd
}

// start get-workspace-access-detail-local command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceAccessDetailLocalOverrides []func(
	*cobra.Command,
	*iamv2.GetWorkspaceAccessDetailLocalRequest,
)

func newGetWorkspaceAccessDetailLocal() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceAccessDetailLocalReq iamv2.GetWorkspaceAccessDetailLocalRequest

	cmd.Flags().Var(&getWorkspaceAccessDetailLocalReq.View, "view", `Controls what fields are returned. Supported values: [BASIC, FULL]`)

	cmd.Use = "get-workspace-access-detail-local PRINCIPAL_ID"
	cmd.Short = `Get workspace access details for a principal.`
	cmd.Long = `Get workspace access details for a principal.
  
  Returns the access details for a principal in the current workspace. Allows
  for checking access details for any provisioned principal (user, service
  principal, or group) in the current workspace. * Provisioned principal here
  refers to one that has been synced into Databricks from the customer's IdP or
  added explicitly to Databricks via SCIM/UI. Allows for passing in a "view"
  parameter to control what fields are returned (BASIC by default or FULL).

  Arguments:
    PRINCIPAL_ID: Required. The internal ID of the principal (user/sp/group) for which the
      access details are being requested.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		_, err = fmt.Sscan(args[0], &getWorkspaceAccessDetailLocalReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}

		response, err := w.WorkspaceIamV2.GetWorkspaceAccessDetailLocal(ctx, getWorkspaceAccessDetailLocalReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceAccessDetailLocalOverrides {
		fn(cmd, &getWorkspaceAccessDetailLocalReq)
	}

	return cmd
}

// start list-groups-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listGroupsProxyOverrides []func(
	*cobra.Command,
	*iamv2.ListGroupsProxyRequest,
)

func newListGroupsProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listGroupsProxyReq iamv2.ListGroupsProxyRequest

	cmd.Flags().IntVar(&listGroupsProxyReq.PageSize, "page-size", listGroupsProxyReq.PageSize, `The maximum number of groups to return.`)
	cmd.Flags().StringVar(&listGroupsProxyReq.PageToken, "page-token", listGroupsProxyReq.PageToken, `A page token, received from a previous ListGroups call.`)

	cmd.Use = "list-groups-proxy"
	cmd.Short = `List groups in the account.`
	cmd.Long = `List groups in the account.
  
  TODO: Write description later when this method is implemented`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.WorkspaceIamV2.ListGroupsProxy(ctx, listGroupsProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listGroupsProxyOverrides {
		fn(cmd, &listGroupsProxyReq)
	}

	return cmd
}

// start list-service-principals-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listServicePrincipalsProxyOverrides []func(
	*cobra.Command,
	*iamv2.ListServicePrincipalsProxyRequest,
)

func newListServicePrincipalsProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listServicePrincipalsProxyReq iamv2.ListServicePrincipalsProxyRequest

	cmd.Flags().IntVar(&listServicePrincipalsProxyReq.PageSize, "page-size", listServicePrincipalsProxyReq.PageSize, `The maximum number of SPs to return.`)
	cmd.Flags().StringVar(&listServicePrincipalsProxyReq.PageToken, "page-token", listServicePrincipalsProxyReq.PageToken, `A page token, received from a previous ListServicePrincipals call.`)

	cmd.Use = "list-service-principals-proxy"
	cmd.Short = `List service principals in the account.`
	cmd.Long = `List service principals in the account.
  
  TODO: Write description later when this method is implemented`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.WorkspaceIamV2.ListServicePrincipalsProxy(ctx, listServicePrincipalsProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listServicePrincipalsProxyOverrides {
		fn(cmd, &listServicePrincipalsProxyReq)
	}

	return cmd
}

// start list-users-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listUsersProxyOverrides []func(
	*cobra.Command,
	*iamv2.ListUsersProxyRequest,
)

func newListUsersProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var listUsersProxyReq iamv2.ListUsersProxyRequest

	cmd.Flags().IntVar(&listUsersProxyReq.PageSize, "page-size", listUsersProxyReq.PageSize, `The maximum number of users to return.`)
	cmd.Flags().StringVar(&listUsersProxyReq.PageToken, "page-token", listUsersProxyReq.PageToken, `A page token, received from a previous ListUsers call.`)

	cmd.Use = "list-users-proxy"
	cmd.Short = `List users in the account.`
	cmd.Long = `List users in the account.
  
  TODO: Write description later when this method is implemented`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.WorkspaceIamV2.ListUsersProxy(ctx, listUsersProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listUsersProxyOverrides {
		fn(cmd, &listUsersProxyReq)
	}

	return cmd
}

// start list-workspace-access-details-local command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceAccessDetailsLocalOverrides []func(
	*cobra.Command,
	*iamv2.ListWorkspaceAccessDetailsLocalRequest,
)

func newListWorkspaceAccessDetailsLocal() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceAccessDetailsLocalReq iamv2.ListWorkspaceAccessDetailsLocalRequest

	cmd.Flags().IntVar(&listWorkspaceAccessDetailsLocalReq.PageSize, "page-size", listWorkspaceAccessDetailsLocalReq.PageSize, `The maximum number of workspace access details to return.`)
	cmd.Flags().StringVar(&listWorkspaceAccessDetailsLocalReq.PageToken, "page-token", listWorkspaceAccessDetailsLocalReq.PageToken, `A page token, received from a previous ListWorkspaceAccessDetails call.`)

	cmd.Use = "list-workspace-access-details-local"
	cmd.Short = `List workspace access details for a workspace.`
	cmd.Long = `List workspace access details for a workspace.
  
  TODO: Write description later when this method is implemented`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.WorkspaceIamV2.ListWorkspaceAccessDetailsLocal(ctx, listWorkspaceAccessDetailsLocalReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWorkspaceAccessDetailsLocalOverrides {
		fn(cmd, &listWorkspaceAccessDetailsLocalReq)
	}

	return cmd
}

// start resolve-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveGroupProxyOverrides []func(
	*cobra.Command,
	*iamv2.ResolveGroupProxyRequest,
)

func newResolveGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveGroupProxyReq iamv2.ResolveGroupProxyRequest
	var resolveGroupProxyJson flags.JsonFlag

	cmd.Flags().Var(&resolveGroupProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-group-proxy EXTERNAL_ID"
	cmd.Short = `Resolve an external group in the Databricks account.`
	cmd.Long = `Resolve an external group in the Databricks account.
  
  Resolves a group with the given external ID from the customer's IdP. If the
  group does not exist, it will be created in the account. If the customer is
  not onboarded onto Automatic Identity Management (AIM), this will return an
  error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the group in the customer's IdP.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'external_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveGroupProxyJson.Unmarshal(&resolveGroupProxyReq)
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
			resolveGroupProxyReq.ExternalId = args[0]
		}

		response, err := w.WorkspaceIamV2.ResolveGroupProxy(ctx, resolveGroupProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveGroupProxyOverrides {
		fn(cmd, &resolveGroupProxyReq)
	}

	return cmd
}

// start resolve-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*iamv2.ResolveServicePrincipalProxyRequest,
)

func newResolveServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveServicePrincipalProxyReq iamv2.ResolveServicePrincipalProxyRequest
	var resolveServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&resolveServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-service-principal-proxy EXTERNAL_ID"
	cmd.Short = `Resolve an external service principal in the Databricks account.`
	cmd.Long = `Resolve an external service principal in the Databricks account.
  
  Resolves an SP with the given external ID from the customer's IdP. If the SP
  does not exist, it will be created. If the customer is not onboarded onto
  Automatic Identity Management (AIM), this will return an error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the service principal in the customer's IdP.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'external_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveServicePrincipalProxyJson.Unmarshal(&resolveServicePrincipalProxyReq)
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
			resolveServicePrincipalProxyReq.ExternalId = args[0]
		}

		response, err := w.WorkspaceIamV2.ResolveServicePrincipalProxy(ctx, resolveServicePrincipalProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveServicePrincipalProxyOverrides {
		fn(cmd, &resolveServicePrincipalProxyReq)
	}

	return cmd
}

// start resolve-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resolveUserProxyOverrides []func(
	*cobra.Command,
	*iamv2.ResolveUserProxyRequest,
)

func newResolveUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var resolveUserProxyReq iamv2.ResolveUserProxyRequest
	var resolveUserProxyJson flags.JsonFlag

	cmd.Flags().Var(&resolveUserProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "resolve-user-proxy EXTERNAL_ID"
	cmd.Short = `Resolve an external user in the Databricks account.`
	cmd.Long = `Resolve an external user in the Databricks account.
  
  Resolves a user with the given external ID from the customer's IdP. If the
  user does not exist, it will be created. If the customer is not onboarded onto
  Automatic Identity Management (AIM), this will return an error.

  Arguments:
    EXTERNAL_ID: Required. The external ID of the user in the customer's IdP.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'external_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resolveUserProxyJson.Unmarshal(&resolveUserProxyReq)
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
			resolveUserProxyReq.ExternalId = args[0]
		}

		response, err := w.WorkspaceIamV2.ResolveUserProxy(ctx, resolveUserProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resolveUserProxyOverrides {
		fn(cmd, &resolveUserProxyReq)
	}

	return cmd
}

// start update-group-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateGroupProxyOverrides []func(
	*cobra.Command,
	*iamv2.UpdateGroupProxyRequest,
)

func newUpdateGroupProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateGroupProxyReq iamv2.UpdateGroupProxyRequest
	updateGroupProxyReq.Group = iamv2.Group{}
	var updateGroupProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateGroupProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateGroupProxyReq.Group.ExternalId, "external-id", updateGroupProxyReq.Group.ExternalId, `ExternalId of the group in the customer's IdP.`)
	cmd.Flags().StringVar(&updateGroupProxyReq.Group.GroupName, "group-name", updateGroupProxyReq.Group.GroupName, `Display name of the group.`)

	cmd.Use = "update-group-proxy INTERNAL_ID UPDATE_MASK"
	cmd.Short = `Update a group in the account.`
	cmd.Long = `Update a group in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the group in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateGroupProxyJson.Unmarshal(&updateGroupProxyReq.Group)
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
		_, err = fmt.Sscan(args[0], &updateGroupProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}
		updateGroupProxyReq.UpdateMask = args[1]

		response, err := w.WorkspaceIamV2.UpdateGroupProxy(ctx, updateGroupProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateGroupProxyOverrides {
		fn(cmd, &updateGroupProxyReq)
	}

	return cmd
}

// start update-service-principal-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateServicePrincipalProxyOverrides []func(
	*cobra.Command,
	*iamv2.UpdateServicePrincipalProxyRequest,
)

func newUpdateServicePrincipalProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateServicePrincipalProxyReq iamv2.UpdateServicePrincipalProxyRequest
	updateServicePrincipalProxyReq.ServicePrincipal = iamv2.ServicePrincipal{}
	var updateServicePrincipalProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateServicePrincipalProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&updateServicePrincipalProxyReq.ServicePrincipal.AccountSpStatus, "account-sp-status", `The activity status of a service principal in a Databricks account. Supported values: [ACTIVE, INACTIVE]`)
	cmd.Flags().StringVar(&updateServicePrincipalProxyReq.ServicePrincipal.ApplicationId, "application-id", updateServicePrincipalProxyReq.ServicePrincipal.ApplicationId, `Application ID of the service principal.`)
	cmd.Flags().StringVar(&updateServicePrincipalProxyReq.ServicePrincipal.DisplayName, "display-name", updateServicePrincipalProxyReq.ServicePrincipal.DisplayName, `Display name of the service principal.`)
	cmd.Flags().StringVar(&updateServicePrincipalProxyReq.ServicePrincipal.ExternalId, "external-id", updateServicePrincipalProxyReq.ServicePrincipal.ExternalId, `ExternalId of the service principal in the customer's IdP.`)

	cmd.Use = "update-service-principal-proxy INTERNAL_ID UPDATE_MASK"
	cmd.Short = `Update a service principal in the account.`
	cmd.Long = `Update a service principal in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the service principal in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateServicePrincipalProxyJson.Unmarshal(&updateServicePrincipalProxyReq.ServicePrincipal)
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
		_, err = fmt.Sscan(args[0], &updateServicePrincipalProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}
		updateServicePrincipalProxyReq.UpdateMask = args[1]

		response, err := w.WorkspaceIamV2.UpdateServicePrincipalProxy(ctx, updateServicePrincipalProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateServicePrincipalProxyOverrides {
		fn(cmd, &updateServicePrincipalProxyReq)
	}

	return cmd
}

// start update-user-proxy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateUserProxyOverrides []func(
	*cobra.Command,
	*iamv2.UpdateUserProxyRequest,
)

func newUpdateUserProxy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateUserProxyReq iamv2.UpdateUserProxyRequest
	updateUserProxyReq.User = iamv2.User{}
	var updateUserProxyJson flags.JsonFlag

	cmd.Flags().Var(&updateUserProxyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&updateUserProxyReq.User.AccountUserStatus, "account-user-status", `The activity status of a user in a Databricks account. Supported values: [ACTIVE, INACTIVE]`)
	cmd.Flags().StringVar(&updateUserProxyReq.User.ExternalId, "external-id", updateUserProxyReq.User.ExternalId, `ExternalId of the user in the customer's IdP.`)
	// TODO: complex arg: name

	cmd.Use = "update-user-proxy INTERNAL_ID UPDATE_MASK USERNAME"
	cmd.Short = `Update a user in the account.`
	cmd.Long = `Update a user in the account.
  
  TODO: Write description later when this method is implemented

  Arguments:
    INTERNAL_ID: Required. Internal ID of the user in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.
    USERNAME: Username/email of the user.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only INTERNAL_ID, UPDATE_MASK as positional arguments. Provide 'username' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateUserProxyJson.Unmarshal(&updateUserProxyReq.User)
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
		_, err = fmt.Sscan(args[0], &updateUserProxyReq.InternalId)
		if err != nil {
			return fmt.Errorf("invalid INTERNAL_ID: %s", args[0])
		}
		updateUserProxyReq.UpdateMask = args[1]
		if !cmd.Flags().Changed("json") {
			updateUserProxyReq.User.Username = args[2]
		}

		response, err := w.WorkspaceIamV2.UpdateUserProxy(ctx, updateUserProxyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateUserProxyOverrides {
		fn(cmd, &updateUserProxyReq)
	}

	return cmd
}

// start update-workspace-access-detail-local command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceAccessDetailLocalOverrides []func(
	*cobra.Command,
	*iamv2.UpdateWorkspaceAccessDetailLocalRequest,
)

func newUpdateWorkspaceAccessDetailLocal() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceAccessDetailLocalReq iamv2.UpdateWorkspaceAccessDetailLocalRequest
	updateWorkspaceAccessDetailLocalReq.WorkspaceAccessDetail = iamv2.WorkspaceAccessDetail{}
	var updateWorkspaceAccessDetailLocalJson flags.JsonFlag

	cmd.Flags().Var(&updateWorkspaceAccessDetailLocalJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: permissions
	cmd.Flags().Var(&updateWorkspaceAccessDetailLocalReq.WorkspaceAccessDetail.Status, "status", `The activity status of the principal in the workspace. Supported values: [ACTIVE, INACTIVE]`)

	cmd.Use = "update-workspace-access-detail-local PRINCIPAL_ID UPDATE_MASK"
	cmd.Short = `Update workspace access for a principal.`
	cmd.Long = `Update workspace access for a principal.
  
  TODO: Write description later when this method is implemented

  Arguments:
    PRINCIPAL_ID: Required. ID of the principal in Databricks.
    UPDATE_MASK: Optional. The list of fields to update.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateWorkspaceAccessDetailLocalJson.Unmarshal(&updateWorkspaceAccessDetailLocalReq.WorkspaceAccessDetail)
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
		_, err = fmt.Sscan(args[0], &updateWorkspaceAccessDetailLocalReq.PrincipalId)
		if err != nil {
			return fmt.Errorf("invalid PRINCIPAL_ID: %s", args[0])
		}
		updateWorkspaceAccessDetailLocalReq.UpdateMask = args[1]

		response, err := w.WorkspaceIamV2.UpdateWorkspaceAccessDetailLocal(ctx, updateWorkspaceAccessDetailLocalReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceAccessDetailLocalOverrides {
		fn(cmd, &updateWorkspaceAccessDetailLocalReq)
	}

	return cmd
}

// end service workspace_iamV2
