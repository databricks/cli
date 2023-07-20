// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package recipient_activation

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sharing"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "recipient-activation",
		Short:   `Databricks Recipient Activation REST API.`,
		Long:    `Databricks Recipient Activation REST API`,
		GroupID: "sharing",
		Annotations: map[string]string{
			"package": "sharing",
		},
	}

	cmd.AddCommand(newGetActivationUrlInfo())
	cmd.AddCommand(newRetrieveToken())

	return cmd
}

// start get-activation-url-info command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getActivationUrlInfoOverrides []func(
	*cobra.Command,
	*sharing.GetActivationUrlInfoRequest,
)

func newGetActivationUrlInfo() *cobra.Command {
	cmd := &cobra.Command{}

	var getActivationUrlInfoReq sharing.GetActivationUrlInfoRequest

	// TODO: short flags

	cmd.Use = "get-activation-url-info ACTIVATION_URL"
	cmd.Short = `Get a share activation URL.`
	cmd.Long = `Get a share activation URL.
  
  Gets an activation URL for a share.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getActivationUrlInfoReq.ActivationUrl = args[0]

		err = w.RecipientActivation.GetActivationUrlInfo(ctx, getActivationUrlInfoReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getActivationUrlInfoOverrides {
		fn(cmd, &getActivationUrlInfoReq)
	}

	return cmd
}

// start retrieve-token command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var retrieveTokenOverrides []func(
	*cobra.Command,
	*sharing.RetrieveTokenRequest,
)

func newRetrieveToken() *cobra.Command {
	cmd := &cobra.Command{}

	var retrieveTokenReq sharing.RetrieveTokenRequest

	// TODO: short flags

	cmd.Use = "retrieve-token ACTIVATION_URL"
	cmd.Short = `Get an access token.`
	cmd.Long = `Get an access token.
  
  Retrieve access token with an activation url. This is a public API without any
  authentication.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		retrieveTokenReq.ActivationUrl = args[0]

		response, err := w.RecipientActivation.RetrieveToken(ctx, retrieveTokenReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range retrieveTokenOverrides {
		fn(cmd, &retrieveTokenReq)
	}

	return cmd
}

// end service RecipientActivation
