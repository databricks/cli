package secrets

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func init() {
	listScopesCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "Scope"}}	{{white "Backend Type"}}
	{{range .}}{{.Name|green}}	{{.BackendType}}
	{{end}}`)

	Cmd.AddCommand(putSecretCmd)
	// TODO: short flags
	putSecretCmd.Flags().Var(&putSecretJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	putSecretCmd.Flags().StringVar(&putSecretReq.BytesValue, "bytes-value", putSecretReq.BytesValue, `If specified, value will be stored as bytes.`)
	putSecretCmd.Flags().StringVar(&putSecretReq.StringValue, "string-value", putSecretReq.StringValue, `If specified, note that the value will be stored in UTF-8 (MB4) form.`)
}

var putSecretReq workspace.PutSecret
var putSecretJson flags.JsonFlag

var putSecretCmd = &cobra.Command{
	Use:   "put-secret SCOPE KEY",
	Short: `Add a secret.`,
	Long: `Add a secret.

  Inserts a secret under the provided scope with the given name. If a secret
  already exists with the same name, this command overwrites the existing
  secret's value. The server encrypts the secret using the secret scope's
  encryption settings before storing it.

  You must have WRITE or MANAGE permission on the secret scope. The secret
  key must consist of alphanumeric characters, dashes, underscores, and periods,
  and cannot exceed 128 characters. The maximum allowed secret value size is 128
  KB. The maximum number of secrets in a given scope is 1000.

  The input fields "string_value" or "bytes_value" specify the type of the
  secret, which will determine the value returned when the secret value is
  requested. Exactly one must be specified.

  Throws RESOURCE_DOES_NOT_EXIST if no such secret scope exists. Throws
  RESOURCE_LIMIT_EXCEEDED if maximum number of secrets in scope is exceeded.
  Throws INVALID_PARAMETER_VALUE if the key name or value length is invalid.
  Throws PERMISSION_DENIED if the user does not have permission to make this
  API call.`,

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
			err = putSecretJson.Unmarshal(&putSecretReq)
			if err != nil {
				return err
			}
		} else {
			putSecretReq.Scope = args[0]
			putSecretReq.Key = args[1]
		}

		value, err := cmdio.Secret(ctx)
		if err != nil {
			return err
		}

		putSecretReq.StringValue = value

		err = w.Secrets.PutSecret(ctx, putSecretReq)
		if err != nil {
			return err
		}
		return nil
	},
}
