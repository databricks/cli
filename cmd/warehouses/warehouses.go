package warehouses

import (
	"time"

	query_history "github.com/databricks/bricks/cmd/warehouses/query-history"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/warehouses"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "warehouses",
	Short: `A SQL warehouse is a compute resource that lets you run SQL commands on data objects within Databricks SQL.`,
	Long: `A SQL warehouse is a compute resource that lets you run SQL commands on data
  objects within Databricks SQL. Compute resources are infrastructure resources
  that provide processing capabilities in the cloud.`,
}

var createWarehouseReq warehouses.CreateWarehouseRequest
var createWarehouseAndWait bool
var createWarehouseTimeout time.Duration

func init() {
	Cmd.AddCommand(createWarehouseCmd)

	createWarehouseCmd.Flags().BoolVar(&createWarehouseAndWait, "wait", true, `wait to reach RUNNING state`)
	createWarehouseCmd.Flags().DurationVar(&createWarehouseTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	createWarehouseCmd.Flags().IntVar(&createWarehouseReq.AutoStopMins, "auto-stop-mins", createWarehouseReq.AutoStopMins, `The amount of time in minutes that a SQL Endpoint must be idle (i.e., no RUNNING queries) before it is automatically stopped.`)
	// TODO: complex arg: channel
	createWarehouseCmd.Flags().StringVar(&createWarehouseReq.ClusterSize, "cluster-size", createWarehouseReq.ClusterSize, `Size of the clusters allocated for this endpoint.`)
	createWarehouseCmd.Flags().StringVar(&createWarehouseReq.CreatorName, "creator-name", createWarehouseReq.CreatorName, `endpoint creator name.`)
	createWarehouseCmd.Flags().BoolVar(&createWarehouseReq.EnablePhoton, "enable-photon", createWarehouseReq.EnablePhoton, `Configures whether the endpoint should use Photon optimized clusters.`)
	createWarehouseCmd.Flags().BoolVar(&createWarehouseReq.EnableServerlessCompute, "enable-serverless-compute", createWarehouseReq.EnableServerlessCompute, `Configures whether the endpoint should use Serverless Compute (aka Nephos) Defaults to value in global endpoint settings.`)
	createWarehouseCmd.Flags().StringVar(&createWarehouseReq.InstanceProfileArn, "instance-profile-arn", createWarehouseReq.InstanceProfileArn, `Deprecated.`)
	createWarehouseCmd.Flags().IntVar(&createWarehouseReq.MaxNumClusters, "max-num-clusters", createWarehouseReq.MaxNumClusters, `Maximum number of clusters that the autoscaler will create to handle concurrent queries.`)
	createWarehouseCmd.Flags().IntVar(&createWarehouseReq.MinNumClusters, "min-num-clusters", createWarehouseReq.MinNumClusters, `Minimum number of available clusters that will be maintained for this SQL Endpoint.`)
	createWarehouseCmd.Flags().StringVar(&createWarehouseReq.Name, "name", createWarehouseReq.Name, `Logical name for the cluster.`)
	createWarehouseCmd.Flags().Var(&createWarehouseReq.SpotInstancePolicy, "spot-instance-policy", `Configurations whether the endpoint should use spot instances.`)
	// TODO: complex arg: tags
	createWarehouseCmd.Flags().Var(&createWarehouseReq.WarehouseType, "warehouse-type", `Warehouse type (Classic/Pro).`)

}

var createWarehouseCmd = &cobra.Command{
	Use:   "create-warehouse",
	Short: `Create a warehouse.`,
	Long: `Create a warehouse.
  
  Creates a new SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if createWarehouseAndWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.CreateWarehouseAndWait(ctx, createWarehouseReq,
				retries.Timeout[warehouses.GetWarehouseResponse](createWarehouseTimeout),
				func(i *retries.Info[warehouses.GetWarehouseResponse]) {
					spinner.Suffix = i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		response, err := w.Warehouses.CreateWarehouse(ctx, createWarehouseReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteWarehouseReq warehouses.DeleteWarehouse
var deleteWarehouseAndWait bool
var deleteWarehouseTimeout time.Duration

func init() {
	Cmd.AddCommand(deleteWarehouseCmd)

	deleteWarehouseCmd.Flags().BoolVar(&deleteWarehouseAndWait, "wait", true, `wait to reach DELETED state`)
	deleteWarehouseCmd.Flags().DurationVar(&deleteWarehouseTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach DELETED state`)
	// TODO: short flags

	deleteWarehouseCmd.Flags().StringVar(&deleteWarehouseReq.Id, "id", deleteWarehouseReq.Id, `Required.`)

}

var deleteWarehouseCmd = &cobra.Command{
	Use:   "delete-warehouse",
	Short: `Delete a warehouse.`,
	Long: `Delete a warehouse.
  
  Deletes a SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if deleteWarehouseAndWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.DeleteWarehouseAndWait(ctx, deleteWarehouseReq,
				retries.Timeout[warehouses.GetWarehouseResponse](deleteWarehouseTimeout),
				func(i *retries.Info[warehouses.GetWarehouseResponse]) {
					spinner.Suffix = i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err := w.Warehouses.DeleteWarehouse(ctx, deleteWarehouseReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var editWarehouseReq warehouses.EditWarehouseRequest
var editWarehouseAndWait bool
var editWarehouseTimeout time.Duration

func init() {
	Cmd.AddCommand(editWarehouseCmd)

	editWarehouseCmd.Flags().BoolVar(&editWarehouseAndWait, "wait", true, `wait to reach RUNNING state`)
	editWarehouseCmd.Flags().DurationVar(&editWarehouseTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	editWarehouseCmd.Flags().IntVar(&editWarehouseReq.AutoStopMins, "auto-stop-mins", editWarehouseReq.AutoStopMins, `The amount of time in minutes that a SQL Endpoint must be idle (i.e., no RUNNING queries) before it is automatically stopped.`)
	// TODO: complex arg: channel
	editWarehouseCmd.Flags().StringVar(&editWarehouseReq.ClusterSize, "cluster-size", editWarehouseReq.ClusterSize, `Size of the clusters allocated for this endpoint.`)
	editWarehouseCmd.Flags().StringVar(&editWarehouseReq.CreatorName, "creator-name", editWarehouseReq.CreatorName, `endpoint creator name.`)
	editWarehouseCmd.Flags().BoolVar(&editWarehouseReq.EnableDatabricksCompute, "enable-databricks-compute", editWarehouseReq.EnableDatabricksCompute, `Configures whether the endpoint should use Databricks Compute (aka Nephos) Deprecated: Use enable_serverless_compute TODO(SC-79930): Remove the field once clients are updated.`)
	editWarehouseCmd.Flags().BoolVar(&editWarehouseReq.EnablePhoton, "enable-photon", editWarehouseReq.EnablePhoton, `Configures whether the endpoint should use Photon optimized clusters.`)
	editWarehouseCmd.Flags().BoolVar(&editWarehouseReq.EnableServerlessCompute, "enable-serverless-compute", editWarehouseReq.EnableServerlessCompute, `Configures whether the endpoint should use Serverless Compute (aka Nephos) Defaults to value in global endpoint settings.`)
	editWarehouseCmd.Flags().StringVar(&editWarehouseReq.Id, "id", editWarehouseReq.Id, `Required.`)
	editWarehouseCmd.Flags().StringVar(&editWarehouseReq.InstanceProfileArn, "instance-profile-arn", editWarehouseReq.InstanceProfileArn, `Deprecated.`)
	editWarehouseCmd.Flags().IntVar(&editWarehouseReq.MaxNumClusters, "max-num-clusters", editWarehouseReq.MaxNumClusters, `Maximum number of clusters that the autoscaler will create to handle concurrent queries.`)
	editWarehouseCmd.Flags().IntVar(&editWarehouseReq.MinNumClusters, "min-num-clusters", editWarehouseReq.MinNumClusters, `Minimum number of available clusters that will be maintained for this SQL Endpoint.`)
	editWarehouseCmd.Flags().StringVar(&editWarehouseReq.Name, "name", editWarehouseReq.Name, `Logical name for the cluster.`)
	editWarehouseCmd.Flags().Var(&editWarehouseReq.SpotInstancePolicy, "spot-instance-policy", `Configurations whether the endpoint should use spot instances.`)
	// TODO: complex arg: tags
	editWarehouseCmd.Flags().Var(&editWarehouseReq.WarehouseType, "warehouse-type", `Warehouse type (Classic/Pro).`)

}

var editWarehouseCmd = &cobra.Command{
	Use:   "edit-warehouse",
	Short: `Update a warehouse.`,
	Long: `Update a warehouse.
  
  Updates the configuration for a SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if editWarehouseAndWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.EditWarehouseAndWait(ctx, editWarehouseReq,
				retries.Timeout[warehouses.GetWarehouseResponse](editWarehouseTimeout),
				func(i *retries.Info[warehouses.GetWarehouseResponse]) {
					spinner.Suffix = i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err := w.Warehouses.EditWarehouse(ctx, editWarehouseReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var getWarehouseReq warehouses.GetWarehouse
var getWarehouseAndWait bool
var getWarehouseTimeout time.Duration

func init() {
	Cmd.AddCommand(getWarehouseCmd)

	getWarehouseCmd.Flags().BoolVar(&getWarehouseAndWait, "wait", true, `wait to reach RUNNING state`)
	getWarehouseCmd.Flags().DurationVar(&getWarehouseTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	getWarehouseCmd.Flags().StringVar(&getWarehouseReq.Id, "id", getWarehouseReq.Id, `Required.`)

}

var getWarehouseCmd = &cobra.Command{
	Use:   "get-warehouse",
	Short: `Get warehouse info.`,
	Long: `Get warehouse info.
  
  Gets the information for a single SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if getWarehouseAndWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.GetWarehouseAndWait(ctx, getWarehouseReq,
				retries.Timeout[warehouses.GetWarehouseResponse](getWarehouseTimeout),
				func(i *retries.Info[warehouses.GetWarehouseResponse]) {
					spinner.Suffix = i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		response, err := w.Warehouses.GetWarehouse(ctx, getWarehouseReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

func init() {
	Cmd.AddCommand(getWorkspaceWarehouseConfigCmd)

}

var getWorkspaceWarehouseConfigCmd = &cobra.Command{
	Use:   "get-workspace-warehouse-config",
	Short: `Get the workspace configuration.`,
	Long: `Get the workspace configuration.
  
  Gets the workspace level configuration that is shared by all SQL warehouses in
  a workspace.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Warehouses.GetWorkspaceWarehouseConfig(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var listWarehousesReq warehouses.ListWarehouses

func init() {
	Cmd.AddCommand(listWarehousesCmd)
	// TODO: short flags

	listWarehousesCmd.Flags().IntVar(&listWarehousesReq.RunAsUserId, "run-as-user-id", listWarehousesReq.RunAsUserId, `Service Principal which will be used to fetch the list of endpoints.`)

}

var listWarehousesCmd = &cobra.Command{
	Use:   "list-warehouses",
	Short: `List warehouses.`,
	Long: `List warehouses.
  
  Lists all SQL warehouses that a user has manager permissions on.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Warehouses.ListWarehousesAll(ctx, listWarehousesReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var setWorkspaceWarehouseConfigReq warehouses.SetWorkspaceWarehouseConfigRequest

func init() {
	Cmd.AddCommand(setWorkspaceWarehouseConfigCmd)
	// TODO: short flags

	// TODO: complex arg: channel
	// TODO: complex arg: config_param
	// TODO: array: data_access_config
	setWorkspaceWarehouseConfigCmd.Flags().BoolVar(&setWorkspaceWarehouseConfigReq.EnableDatabricksCompute, "enable-databricks-compute", setWorkspaceWarehouseConfigReq.EnableDatabricksCompute, `Enable Serverless compute for SQL Endpoints Deprecated: Use enable_serverless_compute TODO(SC-79930): Remove the field once clients are updated.`)
	setWorkspaceWarehouseConfigCmd.Flags().BoolVar(&setWorkspaceWarehouseConfigReq.EnableServerlessCompute, "enable-serverless-compute", setWorkspaceWarehouseConfigReq.EnableServerlessCompute, `Enable Serverless compute for SQL Endpoints.`)
	// TODO: array: enabled_warehouse_types
	// TODO: complex arg: global_param
	setWorkspaceWarehouseConfigCmd.Flags().StringVar(&setWorkspaceWarehouseConfigReq.GoogleServiceAccount, "google-service-account", setWorkspaceWarehouseConfigReq.GoogleServiceAccount, `GCP only: Google Service Account used to pass to cluster to access Google Cloud Storage.`)
	setWorkspaceWarehouseConfigCmd.Flags().StringVar(&setWorkspaceWarehouseConfigReq.InstanceProfileArn, "instance-profile-arn", setWorkspaceWarehouseConfigReq.InstanceProfileArn, `AWS Only: Instance profile used to pass IAM role to the cluster.`)
	setWorkspaceWarehouseConfigCmd.Flags().Var(&setWorkspaceWarehouseConfigReq.SecurityPolicy, "security-policy", `Security policy for endpoints.`)
	setWorkspaceWarehouseConfigCmd.Flags().BoolVar(&setWorkspaceWarehouseConfigReq.ServerlessAgreement, "serverless-agreement", setWorkspaceWarehouseConfigReq.ServerlessAgreement, `Internal.`)
	// TODO: complex arg: sql_configuration_parameters

}

var setWorkspaceWarehouseConfigCmd = &cobra.Command{
	Use:   "set-workspace-warehouse-config",
	Short: `Set the workspace configuration.`,
	Long: `Set the workspace configuration.
  
  Sets the workspace level configuration that is shared by all SQL warehouses in
  a workspace.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Warehouses.SetWorkspaceWarehouseConfig(ctx, setWorkspaceWarehouseConfigReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var startWarehouseReq warehouses.StartWarehouse
var startWarehouseAndWait bool
var startWarehouseTimeout time.Duration

func init() {
	Cmd.AddCommand(startWarehouseCmd)

	startWarehouseCmd.Flags().BoolVar(&startWarehouseAndWait, "wait", true, `wait to reach RUNNING state`)
	startWarehouseCmd.Flags().DurationVar(&startWarehouseTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	startWarehouseCmd.Flags().StringVar(&startWarehouseReq.Id, "id", startWarehouseReq.Id, `Required.`)

}

var startWarehouseCmd = &cobra.Command{
	Use:   "start-warehouse",
	Short: `Start a warehouse.`,
	Long: `Start a warehouse.
  
  Starts a SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if startWarehouseAndWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.StartWarehouseAndWait(ctx, startWarehouseReq,
				retries.Timeout[warehouses.GetWarehouseResponse](startWarehouseTimeout),
				func(i *retries.Info[warehouses.GetWarehouseResponse]) {
					spinner.Suffix = i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err := w.Warehouses.StartWarehouse(ctx, startWarehouseReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var stopWarehouseReq warehouses.StopWarehouse
var stopWarehouseAndWait bool
var stopWarehouseTimeout time.Duration

func init() {
	Cmd.AddCommand(stopWarehouseCmd)

	stopWarehouseCmd.Flags().BoolVar(&stopWarehouseAndWait, "wait", true, `wait to reach STOPPED state`)
	stopWarehouseCmd.Flags().DurationVar(&stopWarehouseTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach STOPPED state`)
	// TODO: short flags

	stopWarehouseCmd.Flags().StringVar(&stopWarehouseReq.Id, "id", stopWarehouseReq.Id, `Required.`)

}

var stopWarehouseCmd = &cobra.Command{
	Use:   "stop-warehouse",
	Short: `Stop a warehouse.`,
	Long: `Stop a warehouse.
  
  Stops a SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if stopWarehouseAndWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.StopWarehouseAndWait(ctx, stopWarehouseReq,
				retries.Timeout[warehouses.GetWarehouseResponse](stopWarehouseTimeout),
				func(i *retries.Info[warehouses.GetWarehouseResponse]) {
					spinner.Suffix = i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err := w.Warehouses.StopWarehouse(ctx, stopWarehouseReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Warehouses

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(query_history.Cmd)
}
