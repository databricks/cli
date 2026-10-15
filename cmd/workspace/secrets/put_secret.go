package secrets

import (
	"errors"
	"io"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	secrets "github.com/databricks/cli/libs/sdktypes/secrets/v1"
	"github.com/spf13/cobra"
)

func newPutSecret() *cobra.Command {
	cmd := &cobra.Command{}

	var putSecretReq secrets.PutSecretRequest
	var putSecretJson flags.JsonFlag
	var bytesValue, stringValue string

	cmd.Flags().Var(&putSecretJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&bytesValue, "bytes-value", bytesValue, `If specified, value will be stored as bytes.`)
	cmd.Flags().StringVar(&stringValue, "string-value", stringValue, `If specified, note that the value will be stored in UTF-8 (MB4) form.`)

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
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			putSecretReq.Scope = new(args[0])
			putSecretReq.Key = new(args[1])

			// bytes_value is base64 on the wire; the request type encodes it.
			switch {
			case bytesValueChanged:
				putSecretReq.BytesValue = []byte(bytesValue)
			case stringValueChanged:
				putSecretReq.StringValue = new(stringValue)
			default:
				// Neither is specified; read secret value from stdin.
				bytes, err := promptSecret(cmd)
				if err != nil {
					return err
				}
				putSecretReq.BytesValue = bytes
			}
		}

		sdkReq, err := secrets.PutSecretRequestToSDK(&putSecretReq)
		if err != nil {
			return err
		}
		_, err = w.Secrets.PutSecret(ctx, *sdkReq)
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
	// If prompting is not supported, read secret from stdin.
	if !cmdio.IsPromptSupported(cmd.Context()) {
		return io.ReadAll(cmd.InOrStdin())
	}

	value, err := cmdio.Secret(cmd.Context(), "Please enter your secret value")
	if err != nil {
		return nil, err
	}

	return []byte(value), nil
}
