// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package network_policies

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	networking "github.com/databricks/cli/libs/sdktypes/networking/v1"
	sdknetworking "github.com/databricks/sdk-go/networking/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*networking.CreateNetworkPolicyRequest,
)

func newCreateNetworkPolicyRpc() *cobra.Command {
	cmd := &cobra.Command{}
	var createNetworkPolicyRpcReq networking.CreateNetworkPolicyRequest
	createNetworkPolicyRpcReq.NetworkPolicy = &networking.AccountNetworkPolicy{}

	var createNetworkPolicyRpcJson flags.JsonFlag

	cmd.Flags().Var(&createNetworkPolicyRpcJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: egress
	// TODO: complex arg: ingress
	// TODO: complex arg: ingress_dry_run
	flags.StringPtrVar(cmd.Flags(), &createNetworkPolicyRpcReq.NetworkPolicy.NetworkPolicyId, "network-policy-id", `The unique identifier for the network policy.`)

	cmd.Use = "create-network-policy-rpc"
	cmd.Short = `Create a network policy.`
	cmd.Long = `Create a network policy.

  Creates a new network policy to manage which network destinations can be
  accessed from the Databricks environment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createNetworkPolicyRpcJson.Unmarshal(&createNetworkPolicyRpcReq.NetworkPolicy)
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

		sdkReq, err := networking.CreateNetworkPolicyRequestToSDK(&createNetworkPolicyRpcReq)
		if err != nil {
			return err
		}
		response, err := a.Networking.CreateNetworkPolicyRpc(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.AccountNetworkPolicyFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdknetworking.DeleteNetworkPolicyRequest,
)

func newDeleteNetworkPolicyRpc() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteNetworkPolicyRpcReq sdknetworking.DeleteNetworkPolicyRequest

	cmd.Use = "delete-network-policy-rpc NETWORK_POLICY_ID"
	cmd.Short = `Delete a network policy.`
	cmd.Long = `Delete a network policy.

  Deletes a network policy. Cannot be called on 'default-policy'.

  Arguments:
    NETWORK_POLICY_ID: The unique identifier of the network policy to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteNetworkPolicyRpcReq.NetworkPolicyId = new(args[0])

		sdkReq := &deleteNetworkPolicyRpcReq
		err = a.Networking.DeleteNetworkPolicyRpc(ctx, *sdkReq)
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
	*sdknetworking.GetNetworkPolicyRequest,
)

func newGetNetworkPolicyRpc() *cobra.Command {
	cmd := &cobra.Command{}
	var getNetworkPolicyRpcReq sdknetworking.GetNetworkPolicyRequest

	cmd.Use = "get-network-policy-rpc NETWORK_POLICY_ID"
	cmd.Short = `Get a network policy.`
	cmd.Long = `Get a network policy.

  Gets a network policy.

  Arguments:
    NETWORK_POLICY_ID: The unique identifier of the network policy to retrieve.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getNetworkPolicyRpcReq.NetworkPolicyId = new(args[0])

		sdkReq := &getNetworkPolicyRpcReq
		response, err := a.Networking.GetNetworkPolicyRpc(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.AccountNetworkPolicyFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*networking.ListNetworkPoliciesRequest,
)

func newListNetworkPoliciesRpc() *cobra.Command {
	cmd := &cobra.Command{}
	var listNetworkPoliciesRpcReq networking.ListNetworkPoliciesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listNetworkPoliciesRpcLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listNetworkPoliciesRpcLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listNetworkPoliciesRpcReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-network-policies-rpc"
	cmd.Short = `List network policies.`
	cmd.Long = `List network policies.

  Gets an array of network policies.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq, err := networking.ListNetworkPoliciesRequestToSDK(&listNetworkPoliciesRpcReq)
		if err != nil {
			return err
		}
		if listNetworkPoliciesRpcLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listNetworkPoliciesRpcLimit)
		}
		if listNetworkPoliciesRpcLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listNetworkPoliciesRpcLimit)
		}
		response := a.Networking.ListNetworkPoliciesRpcIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, networking.AccountNetworkPolicyFromSDK))
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
	*networking.UpdateNetworkPolicyRequest,
)

func newUpdateNetworkPolicyRpc() *cobra.Command {
	cmd := &cobra.Command{}
	var updateNetworkPolicyRpcReq networking.UpdateNetworkPolicyRequest
	updateNetworkPolicyRpcReq.NetworkPolicy = &networking.AccountNetworkPolicy{}

	var updateNetworkPolicyRpcJson flags.JsonFlag

	cmd.Flags().Var(&updateNetworkPolicyRpcJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: egress
	// TODO: complex arg: ingress
	// TODO: complex arg: ingress_dry_run
	flags.StringPtrVar(cmd.Flags(), &updateNetworkPolicyRpcReq.NetworkPolicyId, "network-policy-id", `The unique identifier for the network policy.`)

	cmd.Use = "update-network-policy-rpc NETWORK_POLICY_ID"
	cmd.Short = `Update a network policy.`
	cmd.Long = `Update a network policy.

  Updates a network policy. This allows you to modify the configuration of a
  network policy.

  Arguments:
    NETWORK_POLICY_ID: The unique identifier for the network policy.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateNetworkPolicyRpcJson.Unmarshal(&updateNetworkPolicyRpcReq.NetworkPolicy)
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
		updateNetworkPolicyRpcReq.NetworkPolicyId = new(args[0])

		sdkReq, err := networking.UpdateNetworkPolicyRequestToSDK(&updateNetworkPolicyRpcReq)
		if err != nil {
			return err
		}
		response, err := a.Networking.UpdateNetworkPolicyRpc(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := networking.AccountNetworkPolicyFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
