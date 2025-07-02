// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package network_policies

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		Use:   "network-policies",
		Short: `These APIs manage network policies for this account.`,
		Long: `These APIs manage network policies for this account. Network policies control
  which network destinations can be accessed from the Databricks environment.
  Each Databricks account includes a default policy named 'default-policy'.
  'default-policy' is associated with any workspace lacking an explicit network
  policy assignment, and is automatically associated with each newly created
  workspace. 'default-policy' is reserved and cannot be deleted, but it can be
  updated to customize the default network access rules for your account.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateNetworkPolicyRpc())
	cmd.AddCommand(newDeleteNetworkPolicyRpc())
	cmd.AddCommand(newGetNetworkPolicyRpc())
	cmd.AddCommand(newListNetworkPoliciesRpc())
	cmd.AddCommand(newUpdateNetworkPolicyRpc())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-network-policy-rpc command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createNetworkPolicyRpcOverrides []func(
	*cobra.Command,
	*settings.CreateNetworkPolicyRequest,
)

func newCreateNetworkPolicyRpc() *cobra.Command {
	cmd := &cobra.Command{}

	var createNetworkPolicyRpcReq settings.CreateNetworkPolicyRequest
	createNetworkPolicyRpcReq.NetworkPolicy = settings.AccountNetworkPolicy{}
	var createNetworkPolicyRpcJson flags.JsonFlag

	cmd.Flags().Var(&createNetworkPolicyRpcJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createNetworkPolicyRpcReq.NetworkPolicy.AccountId, "account-id", createNetworkPolicyRpcReq.NetworkPolicy.AccountId, `The associated account ID for this Network Policy object.`)
	// TODO: complex arg: egress
	cmd.Flags().StringVar(&createNetworkPolicyRpcReq.NetworkPolicy.NetworkPolicyId, "network-policy-id", createNetworkPolicyRpcReq.NetworkPolicy.NetworkPolicyId, `The unique identifier for the network policy.`)

	cmd.Use = "create-network-policy-rpc"
	cmd.Short = `Create a network policy.`
	cmd.Long = `Create a network policy.
  
  Creates a new network policy to manage which network destinations can be
  accessed from the Databricks environment.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createNetworkPolicyRpcJson.Unmarshal(&createNetworkPolicyRpcReq.NetworkPolicy)
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

		response, err := a.NetworkPolicies.CreateNetworkPolicyRpc(ctx, createNetworkPolicyRpcReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createNetworkPolicyRpcOverrides {
		fn(cmd, &createNetworkPolicyRpcReq)
	}

	return cmd
}

// start delete-network-policy-rpc command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteNetworkPolicyRpcOverrides []func(
	*cobra.Command,
	*settings.DeleteNetworkPolicyRequest,
)

func newDeleteNetworkPolicyRpc() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteNetworkPolicyRpcReq settings.DeleteNetworkPolicyRequest

	cmd.Use = "delete-network-policy-rpc NETWORK_POLICY_ID"
	cmd.Short = `Delete a network policy.`
	cmd.Long = `Delete a network policy.
  
  Deletes a network policy. Cannot be called on 'default-policy'.

  Arguments:
    NETWORK_POLICY_ID: The unique identifier of the network policy to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deleteNetworkPolicyRpcReq.NetworkPolicyId = args[0]

		err = a.NetworkPolicies.DeleteNetworkPolicyRpc(ctx, deleteNetworkPolicyRpcReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteNetworkPolicyRpcOverrides {
		fn(cmd, &deleteNetworkPolicyRpcReq)
	}

	return cmd
}

// start get-network-policy-rpc command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getNetworkPolicyRpcOverrides []func(
	*cobra.Command,
	*settings.GetNetworkPolicyRequest,
)

func newGetNetworkPolicyRpc() *cobra.Command {
	cmd := &cobra.Command{}

	var getNetworkPolicyRpcReq settings.GetNetworkPolicyRequest

	cmd.Use = "get-network-policy-rpc NETWORK_POLICY_ID"
	cmd.Short = `Get a network policy.`
	cmd.Long = `Get a network policy.
  
  Gets a network policy.

  Arguments:
    NETWORK_POLICY_ID: The unique identifier of the network policy to retrieve.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getNetworkPolicyRpcReq.NetworkPolicyId = args[0]

		response, err := a.NetworkPolicies.GetNetworkPolicyRpc(ctx, getNetworkPolicyRpcReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getNetworkPolicyRpcOverrides {
		fn(cmd, &getNetworkPolicyRpcReq)
	}

	return cmd
}

// start list-network-policies-rpc command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listNetworkPoliciesRpcOverrides []func(
	*cobra.Command,
	*settings.ListNetworkPoliciesRequest,
)

func newListNetworkPoliciesRpc() *cobra.Command {
	cmd := &cobra.Command{}

	var listNetworkPoliciesRpcReq settings.ListNetworkPoliciesRequest

	cmd.Flags().StringVar(&listNetworkPoliciesRpcReq.PageToken, "page-token", listNetworkPoliciesRpcReq.PageToken, `Pagination token to go to next page based on previous query.`)

	cmd.Use = "list-network-policies-rpc"
	cmd.Short = `List network policies.`
	cmd.Long = `List network policies.
  
  Gets an array of network policies.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response := a.NetworkPolicies.ListNetworkPoliciesRpc(ctx, listNetworkPoliciesRpcReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listNetworkPoliciesRpcOverrides {
		fn(cmd, &listNetworkPoliciesRpcReq)
	}

	return cmd
}

// start update-network-policy-rpc command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateNetworkPolicyRpcOverrides []func(
	*cobra.Command,
	*settings.UpdateNetworkPolicyRequest,
)

func newUpdateNetworkPolicyRpc() *cobra.Command {
	cmd := &cobra.Command{}

	var updateNetworkPolicyRpcReq settings.UpdateNetworkPolicyRequest
	updateNetworkPolicyRpcReq.NetworkPolicy = settings.AccountNetworkPolicy{}
	var updateNetworkPolicyRpcJson flags.JsonFlag

	cmd.Flags().Var(&updateNetworkPolicyRpcJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateNetworkPolicyRpcReq.NetworkPolicy.AccountId, "account-id", updateNetworkPolicyRpcReq.NetworkPolicy.AccountId, `The associated account ID for this Network Policy object.`)
	// TODO: complex arg: egress
	cmd.Flags().StringVar(&updateNetworkPolicyRpcReq.NetworkPolicy.NetworkPolicyId, "network-policy-id", updateNetworkPolicyRpcReq.NetworkPolicy.NetworkPolicyId, `The unique identifier for the network policy.`)

	cmd.Use = "update-network-policy-rpc NETWORK_POLICY_ID"
	cmd.Short = `Update a network policy.`
	cmd.Long = `Update a network policy.
  
  Updates a network policy. This allows you to modify the configuration of a
  network policy.

  Arguments:
    NETWORK_POLICY_ID: The unique identifier for the network policy.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateNetworkPolicyRpcJson.Unmarshal(&updateNetworkPolicyRpcReq.NetworkPolicy)
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
		updateNetworkPolicyRpcReq.NetworkPolicyId = args[0]

		response, err := a.NetworkPolicies.UpdateNetworkPolicyRpc(ctx, updateNetworkPolicyRpcReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateNetworkPolicyRpcOverrides {
		fn(cmd, &updateNetworkPolicyRpcReq)
	}

	return cmd
}

// end service NetworkPolicies
