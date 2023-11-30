// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package network_policy

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network-policy",
		Short: `Network policy is a set of rules that defines what can be accessed from your Databricks network.`,
		Long: `Network policy is a set of rules that defines what can be accessed from your
  Databricks network. E.g.: You can choose to block your SQL UDF to access
  internet from your Databricks serverless clusters.
  
  There is only one instance of this setting per account. Since this setting has
  a default value, this setting is present on all accounts even though it's
  never set on a given account. Deletion reverts the value of the setting back
  to the default value.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start delete-account-network-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteAccountNetworkPolicyOverrides []func(
	*cobra.Command,
	*settings.DeleteAccountNetworkPolicyRequest,
)

func newDeleteAccountNetworkPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteAccountNetworkPolicyReq settings.DeleteAccountNetworkPolicyRequest

	// TODO: short flags

	cmd.Use = "delete-account-network-policy ETAG"
	cmd.Short = `Delete Account Network Policy.`
	cmd.Long = `Delete Account Network Policy.
  
  Reverts back all the account network policies back to default.

  Arguments:
    ETAG: etag used for versioning. The response is at least as fresh as the eTag
      provided. This is used for optimistic concurrency control as a way to help
      prevent simultaneous writes of a setting overwriting each other. It is
      strongly suggested that systems make use of the etag in the read -> delete
      pattern to perform setting deletions in order to avoid race conditions.
      That is, get an etag from a GET request, and pass it with the DELETE
      request to identify the rule set version you are deleting.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		deleteAccountNetworkPolicyReq.Etag = args[0]

		response, err := a.NetworkPolicy.DeleteAccountNetworkPolicy(ctx, deleteAccountNetworkPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteAccountNetworkPolicyOverrides {
		fn(cmd, &deleteAccountNetworkPolicyReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteAccountNetworkPolicy())
	})
}

// start read-account-network-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var readAccountNetworkPolicyOverrides []func(
	*cobra.Command,
	*settings.ReadAccountNetworkPolicyRequest,
)

func newReadAccountNetworkPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var readAccountNetworkPolicyReq settings.ReadAccountNetworkPolicyRequest

	// TODO: short flags

	cmd.Use = "read-account-network-policy ETAG"
	cmd.Short = `Get Account Network Policy.`
	cmd.Long = `Get Account Network Policy.
  
  Gets the value of Account level Network Policy.

  Arguments:
    ETAG: etag used for versioning. The response is at least as fresh as the eTag
      provided. This is used for optimistic concurrency control as a way to help
      prevent simultaneous writes of a setting overwriting each other. It is
      strongly suggested that systems make use of the etag in the read -> delete
      pattern to perform setting deletions in order to avoid race conditions.
      That is, get an etag from a GET request, and pass it with the DELETE
      request to identify the rule set version you are deleting.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		readAccountNetworkPolicyReq.Etag = args[0]

		response, err := a.NetworkPolicy.ReadAccountNetworkPolicy(ctx, readAccountNetworkPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range readAccountNetworkPolicyOverrides {
		fn(cmd, &readAccountNetworkPolicyReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newReadAccountNetworkPolicy())
	})
}

// start update-account-network-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateAccountNetworkPolicyOverrides []func(
	*cobra.Command,
	*settings.UpdateAccountNetworkPolicyRequest,
)

func newUpdateAccountNetworkPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var updateAccountNetworkPolicyReq settings.UpdateAccountNetworkPolicyRequest
	var updateAccountNetworkPolicyJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateAccountNetworkPolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateAccountNetworkPolicyReq.AllowMissing, "allow-missing", updateAccountNetworkPolicyReq.AllowMissing, `This should always be set to true for Settings RPCs.`)
	// TODO: complex arg: setting

	cmd.Use = "update-account-network-policy"
	cmd.Short = `Update Account Network Policy.`
	cmd.Long = `Update Account Network Policy.
  
  Updates the policy content of Account level Network Policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateAccountNetworkPolicyJson.Unmarshal(&updateAccountNetworkPolicyReq)
			if err != nil {
				return err
			}
		}

		response, err := a.NetworkPolicy.UpdateAccountNetworkPolicy(ctx, updateAccountNetworkPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateAccountNetworkPolicyOverrides {
		fn(cmd, &updateAccountNetworkPolicyReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdateAccountNetworkPolicy())
	})
}

// end service AccountNetworkPolicy
