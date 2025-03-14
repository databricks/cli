package secrets

import (
	"encoding/base64"
	"errors"
	"io"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func newPutSecret() *cobra.Command {
	cmd := &cobra.Command{}

	var putSecretReq workspace.PutSecret
	var putSecretJson flags.JsonFlag

	cmd.Flags().Var(&putSecretJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&putSecretReq.BytesValue, "bytes-value", putSecretReq.BytesValue, `If specified, value will be stored as bytes.`)
	cmd.Flags().StringVar(&putSecretReq.StringValue, "string-value", putSecretReq.StringValue, `If specified, note that the value will be stored in UTF-8 (MB4) form.`)

	cmd.Use = "put-secret SCOPE KEY"
	cmd.Short = `Add a secret.`
	cmd.Long = `Add a secret.

  Inserts a secret under the provided scope with the given name. If a secret
  already exists with the same name, this command overwrites the existing
  secret's value. The server encrypts the secret using the secret scope's
  encryption settings before storing it.

  You must have WRITE or MANAGE permission on the secret scope. The secret
  key must consist of alphanumeric characters, dashes, underscores, and periods,
  and cannot exceed 128 characters. The maximum allowed secret value size is 128
  KB. The maximum number of secrets in a given scope is 1000.

  The arguments "string-value" or "bytes-value" specify the type of the secret,
  which will determine the value returned when the secret value is requested.

  You can specify the secret value in one of three ways:
  * Specify the value as a string using the --string-value flag.
  * Input the secret when prompted interactively (single-line secrets).
  * Pass the secret via standard input (multi-line secrets).
  `

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = root.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		var diags diag.Diagnostics
		w := cmdctx.WorkspaceClient(ctx)

		bytesValueChanged := cmd.Flags().Changed("bytes-value")
		stringValueChanged := cmd.Flags().Changed("string-value")
		if bytesValueChanged && stringValueChanged {
			return errors.New("cannot specify both --bytes-value and --string-value")
		}

		if cmd.Flags().Changed("json") {
			diags = putSecretJson.Unmarshal(&putSecretReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			putSecretReq.Scope = args[0]
			putSecretReq.Key = args[1]

			switch {
			case bytesValueChanged:
				// Bytes value set; encode as base64.
				putSecretReq.BytesValue = base64.StdEncoding.EncodeToString([]byte(putSecretReq.BytesValue))
			case stringValueChanged:
				// String value set; nothing to do.
			default:
				// Neither is specified; read secret value from stdin.
				bytes, err := promptSecret(cmd)
				if err != nil {
					return err
				}
				putSecretReq.BytesValue = base64.StdEncoding.EncodeToString(bytes)
			}
		}

		err = w.Secrets.PutSecret(ctx, putSecretReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Potential future follow up to auto complete secret scopes for the first argument.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}

func promptSecret(cmd *cobra.Command) ([]byte, error) {
	// If stdin is a TTY, prompt for the secret.
	if !cmdio.IsInTTY(cmd.Context()) {
		return io.ReadAll(os.Stdin)
	}

	value, err := cmdio.Secret(cmd.Context(), "Please enter your secret value")
	if err != nil {
		return nil, err
	}

	return []byte(value), nil
}
