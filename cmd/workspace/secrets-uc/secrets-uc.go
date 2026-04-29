// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package secrets_uc

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets-uc",
		Short: `A secret is a Unity Catalog securable object that stores sensitive credential data (such as passwords, tokens, and keys) within a three-level namespace (**catalog_name.schema_name.secret_name**).`,
		Long: `A secret is a Unity Catalog securable object that stores sensitive credential
  data (such as passwords, tokens, and keys) within a three-level namespace
  (**catalog_name.schema_name.secret_name**).
  
  Secrets can be managed using standard Unity Catalog permissions and are scoped
  to a schema within a catalog.`,
		GroupID: "catalog",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateSecret())
	cmd.AddCommand(newDeleteSecret())
	cmd.AddCommand(newGetSecret())
	cmd.AddCommand(newListSecrets())
	cmd.AddCommand(newUpdateSecret())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSecretOverrides []func(
	*cobra.Command,
	*catalog.CreateSecretRequest,
)

func newCreateSecret() *cobra.Command {
	cmd := &cobra.Command{}

	var createSecretReq catalog.CreateSecretRequest
	createSecretReq.Secret = catalog.Secret{}
	var createSecretJson flags.JsonFlag

	cmd.Flags().Var(&createSecretJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSecretReq.Secret.Comment, "comment", createSecretReq.Secret.Comment, `User-provided free-form text description of the secret.`)
	var expireTimeParam string
	cmd.Flags().StringVar(&expireTimeParam, "expire-time", expireTimeParam, `User-provided expiration time of the secret.`)
	cmd.Flags().StringVar(&createSecretReq.Secret.Owner, "owner", createSecretReq.Secret.Owner, `The owner of the secret.`)

	cmd.Use = "create-secret NAME CATALOG_NAME SCHEMA_NAME VALUE"
	cmd.Short = `Create a secret.`
	cmd.Long = `Create a secret.
  
  Creates a new secret in Unity Catalog.
  
  You must be the owner of the parent schema or have the **CREATE_SECRET** and
  **USE SCHEMA** privileges on the parent schema and **USE CATALOG** on the
  parent catalog.
  
  The secret is stored in the specified catalog and schema, and the **value**
  field contains the sensitive data to be securely stored.

  Arguments:
    NAME: The name of the secret, relative to its parent schema.
    CATALOG_NAME: The name of the catalog where the schema and the secret reside.
    SCHEMA_NAME: The name of the schema where the secret resides.
    VALUE: The secret value to store. This field is input-only and is not returned in
      responses — use the **effective_value** field (via GetSecret with
      **include_value** set to true) to read the secret value. The maximum size
      is 60 KiB (pre-encryption). Accepted content includes passwords, tokens,
      keys, and other sensitive credential data.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'catalog_name', 'schema_name', 'value' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createSecretJson.Unmarshal(&createSecretReq.Secret)
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
			createSecretReq.Secret.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createSecretReq.Secret.CatalogName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createSecretReq.Secret.SchemaName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			createSecretReq.Secret.Value = args[3]
		}

		if expireTimeParam != "" {
			expireTimeBytes := []byte(fmt.Sprintf("\"%s\"", expireTimeParam))
			var expireTimeField sdktime.Time
			err = json.Unmarshal(expireTimeBytes, &expireTimeField)
			if err != nil {
				return fmt.Errorf("invalid EXPIRE_TIME: %s", expireTimeParam)
			}
			createSecretReq.Secret.ExpireTime = &expireTimeField
		}

		response, err := w.SecretsUc.CreateSecret(ctx, createSecretReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSecretOverrides {
		fn(cmd, &createSecretReq)
	}

	return cmd
}

// start delete-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSecretOverrides []func(
	*cobra.Command,
	*catalog.DeleteSecretRequest,
)

func newDeleteSecret() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSecretReq catalog.DeleteSecretRequest

	cmd.Use = "delete-secret FULL_NAME"
	cmd.Short = `Delete a secret.`
	cmd.Long = `Delete a secret.
  
  Deletes a secret by its three-level (fully qualified) name.
  
  You must be the owner of the secret or a metastore admin.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the secret (for example,
      **catalog_name.schema_name.secret_name**).`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteSecretReq.FullName = args[0]

		err = w.SecretsUc.DeleteSecret(ctx, deleteSecretReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSecretOverrides {
		fn(cmd, &deleteSecretReq)
	}

	return cmd
}

// start get-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSecretOverrides []func(
	*cobra.Command,
	*catalog.GetSecretRequest,
)

func newGetSecret() *cobra.Command {
	cmd := &cobra.Command{}

	var getSecretReq catalog.GetSecretRequest

	cmd.Flags().BoolVar(&getSecretReq.IncludeBrowse, "include-browse", getSecretReq.IncludeBrowse, `Whether to include secrets in the response for which you only have the **BROWSE** privilege, which limits access to metadata.`)

	cmd.Use = "get-secret FULL_NAME"
	cmd.Short = `Get a secret.`
	cmd.Long = `Get a secret.
  
  Gets a secret by its three-level (fully qualified) name.
  
  You must be a metastore admin, the owner of the secret, or have the **MANAGE**
  privilege on the secret.
  
  The secret value isn't returned by default. To retrieve it, you must also have
  the **READ_SECRET** privilege and set **include_value** to true in the
  request.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the secret (for example,
      **catalog_name.schema_name.secret_name**).`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSecretReq.FullName = args[0]

		response, err := w.SecretsUc.GetSecret(ctx, getSecretReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSecretOverrides {
		fn(cmd, &getSecretReq)
	}

	return cmd
}

// start list-secrets command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSecretsOverrides []func(
	*cobra.Command,
	*catalog.ListSecretsRequest,
)

func newListSecrets() *cobra.Command {
	cmd := &cobra.Command{}

	var listSecretsReq catalog.ListSecretsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSecretsLimit int

	cmd.Flags().StringVar(&listSecretsReq.CatalogName, "catalog-name", listSecretsReq.CatalogName, `The name of the catalog under which to list secrets.`)
	cmd.Flags().BoolVar(&listSecretsReq.IncludeBrowse, "include-browse", listSecretsReq.IncludeBrowse, `Whether to include secrets in the response for which you only have the **BROWSE** privilege, which limits access to metadata.`)
	cmd.Flags().IntVar(&listSecretsReq.PageSize, "page-size", listSecretsReq.PageSize, `Maximum number of secrets to return.`)
	cmd.Flags().StringVar(&listSecretsReq.SchemaName, "schema-name", listSecretsReq.SchemaName, `The name of the schema under which to list secrets.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSecretsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSecretsReq.PageToken, "page-token", listSecretsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-secrets"
	cmd.Short = `List secrets.`
	cmd.Long = `List secrets.
  
  Lists secrets in Unity Catalog.
  
  You must be a metastore admin, the owner of the secret, or have the **MANAGE**
  privilege on the secret.
  
  Both **catalog_name** and **schema_name** must be specified together to filter
  secrets within a specific schema. Results are paginated; use the
  **page_token** field from the response to retrieve subsequent pages.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.SecretsUc.ListSecrets(ctx, listSecretsReq)
		if listSecretsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSecretsLimit)
		}
		if listSecretsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSecretsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSecretsOverrides {
		fn(cmd, &listSecretsReq)
	}

	return cmd
}

// start update-secret command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSecretOverrides []func(
	*cobra.Command,
	*catalog.UpdateSecretRequest,
)

func newUpdateSecret() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSecretReq catalog.UpdateSecretRequest
	updateSecretReq.Secret = catalog.Secret{}
	var updateSecretJson flags.JsonFlag

	cmd.Flags().Var(&updateSecretJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateSecretReq.Secret.Comment, "comment", updateSecretReq.Secret.Comment, `User-provided free-form text description of the secret.`)
	var expireTimeParam string
	cmd.Flags().StringVar(&expireTimeParam, "expire-time", expireTimeParam, `User-provided expiration time of the secret.`)
	cmd.Flags().StringVar(&updateSecretReq.Secret.Owner, "owner", updateSecretReq.Secret.Owner, `The owner of the secret.`)

	cmd.Use = "update-secret FULL_NAME UPDATE_MASK NAME CATALOG_NAME SCHEMA_NAME VALUE"
	cmd.Short = `Update a secret.`
	cmd.Long = `Update a secret.
  
  Updates an existing secret in Unity Catalog.
  
  You must be the owner of the secret or a metastore admin. If you are a
  metastore admin, only the **owner** field can be changed.
  
  Use the **update_mask** field to specify which fields to update. Supported
  updatable fields include **value**, **comment**, **owner**, and
  **expire_time**.

  Arguments:
    FULL_NAME: The three-level (fully qualified) name of the secret (for example,
      **catalog_name.schema_name.secret_name**).
    UPDATE_MASK: The field mask specifying which fields of the secret to update. Supported
      fields: **value**, **comment**, **owner**, **expire_time**.
    NAME: The name of the secret, relative to its parent schema.
    CATALOG_NAME: The name of the catalog where the schema and the secret reside.
    SCHEMA_NAME: The name of the schema where the secret resides.
    VALUE: The secret value to store. This field is input-only and is not returned in
      responses — use the **effective_value** field (via GetSecret with
      **include_value** set to true) to read the secret value. The maximum size
      is 60 KiB (pre-encryption). Accepted content includes passwords, tokens,
      keys, and other sensitive credential data.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only FULL_NAME, UPDATE_MASK as positional arguments. Provide 'name', 'catalog_name', 'schema_name', 'value' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateSecretJson.Unmarshal(&updateSecretReq.Secret)
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
		updateSecretReq.FullName = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateSecretReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateSecretReq.Secret.Name = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateSecretReq.Secret.CatalogName = args[3]
		}
		if !cmd.Flags().Changed("json") {
			updateSecretReq.Secret.SchemaName = args[4]
		}
		if !cmd.Flags().Changed("json") {
			updateSecretReq.Secret.Value = args[5]
		}

		if expireTimeParam != "" {
			expireTimeBytes := []byte(fmt.Sprintf("\"%s\"", expireTimeParam))
			var expireTimeField sdktime.Time
			err = json.Unmarshal(expireTimeBytes, &expireTimeField)
			if err != nil {
				return fmt.Errorf("invalid EXPIRE_TIME: %s", expireTimeParam)
			}
			updateSecretReq.Secret.ExpireTime = &expireTimeField
		}

		response, err := w.SecretsUc.UpdateSecret(ctx, updateSecretReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSecretOverrides {
		fn(cmd, &updateSecretReq)
	}

	return cmd
}

// end service SecretsUc
