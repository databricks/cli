// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package recipients

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
		Use:   "recipients",
		Short: `A recipient is an object you create using :method:recipients/create to represent an organization which you want to allow access shares.`,
		Long: `A recipient is an object you create using :method:recipients/create to
  represent an organization which you want to allow access shares. The way how
  sharing works differs depending on whether or not your recipient has access to
  a Databricks workspace that is enabled for Unity Catalog:
  
  - For recipients with access to a Databricks workspace that is enabled for
  Unity Catalog, you can create a recipient object along with a unique sharing
  identifier you get from the recipient. The sharing identifier is the key
  identifier that enables the secure connection. This sharing mode is called
  **Databricks-to-Databricks sharing**.
  
  - For recipients without access to a Databricks workspace that is enabled for
  Unity Catalog, when you create a recipient object, Databricks generates an
  activation link you can send to the recipient. The recipient follows the
  activation link to download the credential file, and then uses the credential
  file to establish a secure connection to receive the shared data. This sharing
  mode is called **open sharing**.`,
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
	cmd.AddCommand(newRotateToken())
	cmd.AddCommand(newSharePermissions())
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
	*sharing.CreateRecipient,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sharing.CreateRecipient
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `Description about the recipient.`)
	cmd.Flags().StringVar(&createReq.DataRecipientGlobalMetastoreId, "data-recipient-global-metastore-id", createReq.DataRecipientGlobalMetastoreId, `The global Unity Catalog metastore id provided by the data recipient.`)
	cmd.Flags().Int64Var(&createReq.ExpirationTime, "expiration-time", createReq.ExpirationTime, `Expiration timestamp of the token, in epoch milliseconds.`)
	// TODO: complex arg: ip_access_list
	cmd.Flags().StringVar(&createReq.Owner, "owner", createReq.Owner, `Username of the recipient owner.`)
	// TODO: complex arg: properties_kvpairs
	cmd.Flags().StringVar(&createReq.SharingCode, "sharing-code", createReq.SharingCode, `The one-time sharing code provided by the data recipient.`)

	cmd.Use = "create NAME AUTHENTICATION_TYPE"
	cmd.Short = `Create a share recipient.`
	cmd.Long = `Create a share recipient.
  
  Creates a new recipient with the delta sharing authentication type in the
  metastore. The caller must be a metastore admin or have the
  **CREATE_RECIPIENT** privilege on the metastore.

  Arguments:
    NAME: Name of Recipient.
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

		response, err := w.Recipients.Create(ctx, createReq)
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
	*sharing.DeleteRecipientRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sharing.DeleteRecipientRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a share recipient.`
	cmd.Long = `Delete a share recipient.
  
  Deletes the specified recipient from the metastore. The caller must be the
  owner of the recipient.

  Arguments:
    NAME: Name of the recipient.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Name = args[0]

		err = w.Recipients.Delete(ctx, deleteReq)
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
	*sharing.GetRecipientRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sharing.GetRecipientRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a share recipient.`
	cmd.Long = `Get a share recipient.
  
  Gets a share recipient from the metastore if:
  
  * the caller is the owner of the share recipient, or: * is a metastore admin

  Arguments:
    NAME: Name of the recipient.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Name = args[0]

		response, err := w.Recipients.Get(ctx, getReq)
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
	*sharing.ListRecipientsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sharing.ListRecipientsRequest

	cmd.Flags().StringVar(&listReq.DataRecipientGlobalMetastoreId, "data-recipient-global-metastore-id", listReq.DataRecipientGlobalMetastoreId, `If not provided, all recipients will be returned.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of recipients to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list"
	cmd.Short = `List share recipients.`
	cmd.Long = `List share recipients.
  
  Gets an array of all share recipients within the current metastore where:
  
  * the caller is a metastore admin, or * the caller is the owner. There is no
  guarantee of a specific ordering of the elements in the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Recipients.List(ctx, listReq)
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

// start rotate-token command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var rotateTokenOverrides []func(
	*cobra.Command,
	*sharing.RotateRecipientToken,
)

func newRotateToken() *cobra.Command {
	cmd := &cobra.Command{}

	var rotateTokenReq sharing.RotateRecipientToken
	var rotateTokenJson flags.JsonFlag

	cmd.Flags().Var(&rotateTokenJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "rotate-token NAME EXISTING_TOKEN_EXPIRE_IN_SECONDS"
	cmd.Short = `Rotate a token.`
	cmd.Long = `Rotate a token.
  
  Refreshes the specified recipient's delta sharing authentication token with
  the provided token info. The caller must be the owner of the recipient.

  Arguments:
    NAME: The name of the Recipient.
    EXISTING_TOKEN_EXPIRE_IN_SECONDS: The expiration time of the bearer token in ISO 8601 format. This will set
      the expiration_time of existing token only to a smaller timestamp, it
      cannot extend the expiration_time. Use 0 to expire the existing token
      immediately, negative number will return an error.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME as positional arguments. Provide 'existing_token_expire_in_seconds' in your JSON input")
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
			diags := rotateTokenJson.Unmarshal(&rotateTokenReq)
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
		rotateTokenReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &rotateTokenReq.ExistingTokenExpireInSeconds)
			if err != nil {
				return fmt.Errorf("invalid EXISTING_TOKEN_EXPIRE_IN_SECONDS: %s", args[1])
			}
		}

		response, err := w.Recipients.RotateToken(ctx, rotateTokenReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range rotateTokenOverrides {
		fn(cmd, &rotateTokenReq)
	}

	return cmd
}

// start share-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var sharePermissionsOverrides []func(
	*cobra.Command,
	*sharing.SharePermissionsRequest,
)

func newSharePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var sharePermissionsReq sharing.SharePermissionsRequest

	cmd.Flags().IntVar(&sharePermissionsReq.MaxResults, "max-results", sharePermissionsReq.MaxResults, `Maximum number of permissions to return.`)
	cmd.Flags().StringVar(&sharePermissionsReq.PageToken, "page-token", sharePermissionsReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "share-permissions NAME"
	cmd.Short = `Get recipient share permissions.`
	cmd.Long = `Get recipient share permissions.
  
  Gets the share permissions for the specified Recipient. The caller must be a
  metastore admin or the owner of the Recipient.

  Arguments:
    NAME: The name of the Recipient.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		sharePermissionsReq.Name = args[0]

		response, err := w.Recipients.SharePermissions(ctx, sharePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range sharePermissionsOverrides {
		fn(cmd, &sharePermissionsReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*sharing.UpdateRecipient,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sharing.UpdateRecipient
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `Description about the recipient.`)
	cmd.Flags().Int64Var(&updateReq.ExpirationTime, "expiration-time", updateReq.ExpirationTime, `Expiration timestamp of the token, in epoch milliseconds.`)
	// TODO: complex arg: ip_access_list
	cmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `New name for the recipient.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of the recipient owner.`)
	// TODO: complex arg: properties_kvpairs

	cmd.Use = "update NAME"
	cmd.Short = `Update a share recipient.`
	cmd.Long = `Update a share recipient.
  
  Updates an existing recipient in the metastore. The caller must be a metastore
  admin or the owner of the recipient. If the recipient name will be updated,
  the user must be both a metastore admin and the owner of the recipient.

  Arguments:
    NAME: Name of the recipient.`

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
		updateReq.Name = args[0]

		response, err := w.Recipients.Update(ctx, updateReq)
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

// end service Recipients
