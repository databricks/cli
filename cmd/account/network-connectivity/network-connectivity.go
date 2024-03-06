// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package network_connectivity

import (
	"fmt"

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
		Use:   "network-connectivity",
		Short: `These APIs provide configurations for the network connectivity of your workspaces for serverless compute resources.`,
		Long: `These APIs provide configurations for the network connectivity of your
  workspaces for serverless compute resources.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-network-connectivity-configuration command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createNetworkConnectivityConfigurationOverrides []func(
	*cobra.Command,
	*settings.CreateNetworkConnectivityConfigRequest,
)

func newCreateNetworkConnectivityConfiguration() *cobra.Command {
	cmd := &cobra.Command{}

	var createNetworkConnectivityConfigurationReq settings.CreateNetworkConnectivityConfigRequest
	var createNetworkConnectivityConfigurationJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createNetworkConnectivityConfigurationJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-network-connectivity-configuration NAME REGION"
	cmd.Short = `Create a network connectivity configuration.`
	cmd.Long = `Create a network connectivity configuration.

  Arguments:
    NAME: The name of the network connectivity configuration. The name can contain
      alphanumeric characters, hyphens, and underscores. The length must be
      between 3 and 30 characters. The name must match the regular expression
      ^[0-9a-zA-Z-_]{3,30}$.
    REGION: The region for the network connectivity configuration. Only workspaces in
      the same region can be attached to the network connectivity configuration.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'region' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createNetworkConnectivityConfigurationJson.Unmarshal(&createNetworkConnectivityConfigurationReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			createNetworkConnectivityConfigurationReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createNetworkConnectivityConfigurationReq.Region = args[1]
		}

		response, err := a.NetworkConnectivity.CreateNetworkConnectivityConfiguration(ctx, createNetworkConnectivityConfigurationReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createNetworkConnectivityConfigurationOverrides {
		fn(cmd, &createNetworkConnectivityConfigurationReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreateNetworkConnectivityConfiguration())
	})
}

// start create-private-endpoint-rule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createPrivateEndpointRuleOverrides []func(
	*cobra.Command,
	*settings.CreatePrivateEndpointRuleRequest,
)

func newCreatePrivateEndpointRule() *cobra.Command {
	cmd := &cobra.Command{}

	var createPrivateEndpointRuleReq settings.CreatePrivateEndpointRuleRequest
	var createPrivateEndpointRuleJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createPrivateEndpointRuleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-private-endpoint-rule NETWORK_CONNECTIVITY_CONFIG_ID RESOURCE_ID GROUP_ID"
	cmd.Short = `Create a private endpoint rule.`
	cmd.Long = `Create a private endpoint rule.
  
  Create a private endpoint rule for the specified network connectivity config
  object. Once the object is created, Databricks asynchronously provisions a new
  Azure private endpoint to your specified Azure resource.
  
  **IMPORTANT**: You must use Azure portal or other Azure tools to approve the
  private endpoint to complete the connection. To get the information of the
  private endpoint created, make a GET request on the new private endpoint
  rule. See [serverless private link].
  
  [serverless private link]: https://learn.microsoft.com/azure/databricks/security/network/serverless-network-security/serverless-private-link

  Arguments:
    NETWORK_CONNECTIVITY_CONFIG_ID: Your Network Connectvity Configuration ID.
    RESOURCE_ID: The Azure resource ID of the target resource.
    GROUP_ID: The sub-resource type (group ID) of the target resource. Note that to
      connect to workspace root storage (root DBFS), you need two endpoints, one
      for blob and one for dfs.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NETWORK_CONNECTIVITY_CONFIG_ID as positional arguments. Provide 'resource_id', 'group_id' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createPrivateEndpointRuleJson.Unmarshal(&createPrivateEndpointRuleReq)
			if err != nil {
				return err
			}
		}
		createPrivateEndpointRuleReq.NetworkConnectivityConfigId = args[0]
		if !cmd.Flags().Changed("json") {
			createPrivateEndpointRuleReq.ResourceId = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createPrivateEndpointRuleReq.GroupId)
			if err != nil {
				return fmt.Errorf("invalid GROUP_ID: %s", args[2])
			}
		}

		response, err := a.NetworkConnectivity.CreatePrivateEndpointRule(ctx, createPrivateEndpointRuleReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createPrivateEndpointRuleOverrides {
		fn(cmd, &createPrivateEndpointRuleReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreatePrivateEndpointRule())
	})
}

// start delete-network-connectivity-configuration command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteNetworkConnectivityConfigurationOverrides []func(
	*cobra.Command,
	*settings.DeleteNetworkConnectivityConfigurationRequest,
)

func newDeleteNetworkConnectivityConfiguration() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteNetworkConnectivityConfigurationReq settings.DeleteNetworkConnectivityConfigurationRequest

	// TODO: short flags

	cmd.Use = "delete-network-connectivity-configuration NETWORK_CONNECTIVITY_CONFIG_ID"
	cmd.Short = `Delete a network connectivity configuration.`
	cmd.Long = `Delete a network connectivity configuration.
  
  Deletes a network connectivity configuration.

  Arguments:
    NETWORK_CONNECTIVITY_CONFIG_ID: Your Network Connectvity Configuration ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		deleteNetworkConnectivityConfigurationReq.NetworkConnectivityConfigId = args[0]

		err = a.NetworkConnectivity.DeleteNetworkConnectivityConfiguration(ctx, deleteNetworkConnectivityConfigurationReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteNetworkConnectivityConfigurationOverrides {
		fn(cmd, &deleteNetworkConnectivityConfigurationReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteNetworkConnectivityConfiguration())
	})
}

// start delete-private-endpoint-rule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deletePrivateEndpointRuleOverrides []func(
	*cobra.Command,
	*settings.DeletePrivateEndpointRuleRequest,
)

func newDeletePrivateEndpointRule() *cobra.Command {
	cmd := &cobra.Command{}

	var deletePrivateEndpointRuleReq settings.DeletePrivateEndpointRuleRequest

	// TODO: short flags

	cmd.Use = "delete-private-endpoint-rule NETWORK_CONNECTIVITY_CONFIG_ID PRIVATE_ENDPOINT_RULE_ID"
	cmd.Short = `Delete a private endpoint rule.`
	cmd.Long = `Delete a private endpoint rule.
  
  Initiates deleting a private endpoint rule. The private endpoint will be
  deactivated and will be purged after seven days of deactivation. When a
  private endpoint is in deactivated state, deactivated field is set to true
  and the private endpoint is not available to your serverless compute
  resources.

  Arguments:
    NETWORK_CONNECTIVITY_CONFIG_ID: Your Network Connectvity Configuration ID.
    PRIVATE_ENDPOINT_RULE_ID: Your private endpoint rule ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		deletePrivateEndpointRuleReq.NetworkConnectivityConfigId = args[0]
		deletePrivateEndpointRuleReq.PrivateEndpointRuleId = args[1]

		response, err := a.NetworkConnectivity.DeletePrivateEndpointRule(ctx, deletePrivateEndpointRuleReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deletePrivateEndpointRuleOverrides {
		fn(cmd, &deletePrivateEndpointRuleReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeletePrivateEndpointRule())
	})
}

// start get-network-connectivity-configuration command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getNetworkConnectivityConfigurationOverrides []func(
	*cobra.Command,
	*settings.GetNetworkConnectivityConfigurationRequest,
)

func newGetNetworkConnectivityConfiguration() *cobra.Command {
	cmd := &cobra.Command{}

	var getNetworkConnectivityConfigurationReq settings.GetNetworkConnectivityConfigurationRequest

	// TODO: short flags

	cmd.Use = "get-network-connectivity-configuration NETWORK_CONNECTIVITY_CONFIG_ID"
	cmd.Short = `Get a network connectivity configuration.`
	cmd.Long = `Get a network connectivity configuration.
  
  Gets a network connectivity configuration.

  Arguments:
    NETWORK_CONNECTIVITY_CONFIG_ID: Your Network Connectvity Configuration ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		getNetworkConnectivityConfigurationReq.NetworkConnectivityConfigId = args[0]

		response, err := a.NetworkConnectivity.GetNetworkConnectivityConfiguration(ctx, getNetworkConnectivityConfigurationReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getNetworkConnectivityConfigurationOverrides {
		fn(cmd, &getNetworkConnectivityConfigurationReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetNetworkConnectivityConfiguration())
	})
}

// start get-private-endpoint-rule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPrivateEndpointRuleOverrides []func(
	*cobra.Command,
	*settings.GetPrivateEndpointRuleRequest,
)

func newGetPrivateEndpointRule() *cobra.Command {
	cmd := &cobra.Command{}

	var getPrivateEndpointRuleReq settings.GetPrivateEndpointRuleRequest

	// TODO: short flags

	cmd.Use = "get-private-endpoint-rule NETWORK_CONNECTIVITY_CONFIG_ID PRIVATE_ENDPOINT_RULE_ID"
	cmd.Short = `Get a private endpoint rule.`
	cmd.Long = `Get a private endpoint rule.
  
  Gets the private endpoint rule.

  Arguments:
    NETWORK_CONNECTIVITY_CONFIG_ID: Your Network Connectvity Configuration ID.
    PRIVATE_ENDPOINT_RULE_ID: Your private endpoint rule ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		getPrivateEndpointRuleReq.NetworkConnectivityConfigId = args[0]
		getPrivateEndpointRuleReq.PrivateEndpointRuleId = args[1]

		response, err := a.NetworkConnectivity.GetPrivateEndpointRule(ctx, getPrivateEndpointRuleReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPrivateEndpointRuleOverrides {
		fn(cmd, &getPrivateEndpointRuleReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetPrivateEndpointRule())
	})
}

// start list-network-connectivity-configurations command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listNetworkConnectivityConfigurationsOverrides []func(
	*cobra.Command,
	*settings.ListNetworkConnectivityConfigurationsRequest,
)

func newListNetworkConnectivityConfigurations() *cobra.Command {
	cmd := &cobra.Command{}

	var listNetworkConnectivityConfigurationsReq settings.ListNetworkConnectivityConfigurationsRequest

	// TODO: short flags

	cmd.Flags().StringVar(&listNetworkConnectivityConfigurationsReq.PageToken, "page-token", listNetworkConnectivityConfigurationsReq.PageToken, `Pagination token to go to next page based on previous query.`)

	cmd.Use = "list-network-connectivity-configurations"
	cmd.Short = `List network connectivity configurations.`
	cmd.Long = `List network connectivity configurations.
  
  Gets an array of network connectivity configurations.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		response := a.NetworkConnectivity.ListNetworkConnectivityConfigurations(ctx, listNetworkConnectivityConfigurationsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listNetworkConnectivityConfigurationsOverrides {
		fn(cmd, &listNetworkConnectivityConfigurationsReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newListNetworkConnectivityConfigurations())
	})
}

// start list-private-endpoint-rules command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listPrivateEndpointRulesOverrides []func(
	*cobra.Command,
	*settings.ListPrivateEndpointRulesRequest,
)

func newListPrivateEndpointRules() *cobra.Command {
	cmd := &cobra.Command{}

	var listPrivateEndpointRulesReq settings.ListPrivateEndpointRulesRequest

	// TODO: short flags

	cmd.Flags().StringVar(&listPrivateEndpointRulesReq.PageToken, "page-token", listPrivateEndpointRulesReq.PageToken, `Pagination token to go to next page based on previous query.`)

	cmd.Use = "list-private-endpoint-rules NETWORK_CONNECTIVITY_CONFIG_ID"
	cmd.Short = `List private endpoint rules.`
	cmd.Long = `List private endpoint rules.
  
  Gets an array of private endpoint rules.

  Arguments:
    NETWORK_CONNECTIVITY_CONFIG_ID: Your Network Connectvity Configuration ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		listPrivateEndpointRulesReq.NetworkConnectivityConfigId = args[0]

		response := a.NetworkConnectivity.ListPrivateEndpointRules(ctx, listPrivateEndpointRulesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listPrivateEndpointRulesOverrides {
		fn(cmd, &listPrivateEndpointRulesReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newListPrivateEndpointRules())
	})
}

// end service NetworkConnectivity
