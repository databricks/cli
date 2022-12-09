package warehouses

import (
	"time"

	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "warehouses",
	Short: `A SQL warehouse is a compute resource that lets you run SQL commands on data objects within Databricks SQL.`,
	Long: `A SQL warehouse is a compute resource that lets you run SQL commands on data
  objects within Databricks SQL. Compute resources are infrastructure resources
  that provide processing capabilities in the cloud.`,
}

// start create command

var createReq sql.CreateWarehouseRequest
var createJson jsonflag.JsonFlag
var createNoWait bool
var createTimeout time.Duration

func init() {
	Cmd.AddCommand(createCmd)

	createCmd.Flags().BoolVar(&createNoWait, "no-wait", createNoWait, `do not wait to reach RUNNING state`)
	createCmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().IntVar(&createReq.AutoStopMins, "auto-stop-mins", createReq.AutoStopMins, `The amount of time in minutes that a SQL Endpoint must be idle (i.e., no RUNNING queries) before it is automatically stopped.`)
	// TODO: complex arg: channel
	createCmd.Flags().StringVar(&createReq.ClusterSize, "cluster-size", createReq.ClusterSize, `Size of the clusters allocated for this endpoint.`)
	createCmd.Flags().StringVar(&createReq.CreatorName, "creator-name", createReq.CreatorName, `endpoint creator name.`)
	createCmd.Flags().BoolVar(&createReq.EnablePhoton, "enable-photon", createReq.EnablePhoton, `Configures whether the endpoint should use Photon optimized clusters.`)
	createCmd.Flags().BoolVar(&createReq.EnableServerlessCompute, "enable-serverless-compute", createReq.EnableServerlessCompute, `Configures whether the endpoint should use Serverless Compute (aka Nephos) Defaults to value in global endpoint settings.`)
	createCmd.Flags().StringVar(&createReq.InstanceProfileArn, "instance-profile-arn", createReq.InstanceProfileArn, `Deprecated.`)
	createCmd.Flags().IntVar(&createReq.MaxNumClusters, "max-num-clusters", createReq.MaxNumClusters, `Maximum number of clusters that the autoscaler will create to handle concurrent queries.`)
	createCmd.Flags().IntVar(&createReq.MinNumClusters, "min-num-clusters", createReq.MinNumClusters, `Minimum number of available clusters that will be maintained for this SQL Endpoint.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `Logical name for the cluster.`)
	createCmd.Flags().Var(&createReq.SpotInstancePolicy, "spot-instance-policy", `Configurations whether the endpoint should use spot instances.`)
	// TODO: complex arg: tags
	createCmd.Flags().Var(&createReq.WarehouseType, "warehouse-type", `Warehouse type (Classic/Pro).`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a warehouse.`,
	Long: `Create a warehouse.
  
  Creates a new SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if !createNoWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.CreateAndWait(ctx, createReq,
				retries.Timeout[sql.GetWarehouseResponse](createTimeout),
				func(i *retries.Info[sql.GetWarehouseResponse]) {
					spinner.Suffix = " " + i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		response, err := w.Warehouses.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq sql.DeleteWarehouseRequest

var deleteNoWait bool
var deleteTimeout time.Duration

func init() {
	Cmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVar(&deleteNoWait, "no-wait", deleteNoWait, `do not wait to reach DELETED state`)
	deleteCmd.Flags().DurationVar(&deleteTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach DELETED state`)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", deleteReq.Id, `Required.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a warehouse.`,
	Long: `Delete a warehouse.
  
  Deletes a SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if !deleteNoWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.DeleteAndWait(ctx, deleteReq,
				retries.Timeout[sql.GetWarehouseResponse](deleteTimeout),
				func(i *retries.Info[sql.GetWarehouseResponse]) {
					spinner.Suffix = " " + i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err = w.Warehouses.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start edit command

var editReq sql.EditWarehouseRequest
var editJson jsonflag.JsonFlag
var editNoWait bool
var editTimeout time.Duration

func init() {
	Cmd.AddCommand(editCmd)

	editCmd.Flags().BoolVar(&editNoWait, "no-wait", editNoWait, `do not wait to reach RUNNING state`)
	editCmd.Flags().DurationVar(&editTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags
	editCmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	editCmd.Flags().IntVar(&editReq.AutoStopMins, "auto-stop-mins", editReq.AutoStopMins, `The amount of time in minutes that a SQL Endpoint must be idle (i.e., no RUNNING queries) before it is automatically stopped.`)
	// TODO: complex arg: channel
	editCmd.Flags().StringVar(&editReq.ClusterSize, "cluster-size", editReq.ClusterSize, `Size of the clusters allocated for this endpoint.`)
	editCmd.Flags().StringVar(&editReq.CreatorName, "creator-name", editReq.CreatorName, `endpoint creator name.`)
	editCmd.Flags().BoolVar(&editReq.EnableDatabricksCompute, "enable-databricks-compute", editReq.EnableDatabricksCompute, `Configures whether the endpoint should use Databricks Compute (aka Nephos) Deprecated: Use enable_serverless_compute TODO(SC-79930): Remove the field once clients are updated.`)
	editCmd.Flags().BoolVar(&editReq.EnablePhoton, "enable-photon", editReq.EnablePhoton, `Configures whether the endpoint should use Photon optimized clusters.`)
	editCmd.Flags().BoolVar(&editReq.EnableServerlessCompute, "enable-serverless-compute", editReq.EnableServerlessCompute, `Configures whether the endpoint should use Serverless Compute (aka Nephos) Defaults to value in global endpoint settings.`)
	editCmd.Flags().StringVar(&editReq.Id, "id", editReq.Id, `Required.`)
	editCmd.Flags().StringVar(&editReq.InstanceProfileArn, "instance-profile-arn", editReq.InstanceProfileArn, `Deprecated.`)
	editCmd.Flags().IntVar(&editReq.MaxNumClusters, "max-num-clusters", editReq.MaxNumClusters, `Maximum number of clusters that the autoscaler will create to handle concurrent queries.`)
	editCmd.Flags().IntVar(&editReq.MinNumClusters, "min-num-clusters", editReq.MinNumClusters, `Minimum number of available clusters that will be maintained for this SQL Endpoint.`)
	editCmd.Flags().StringVar(&editReq.Name, "name", editReq.Name, `Logical name for the cluster.`)
	editCmd.Flags().Var(&editReq.SpotInstancePolicy, "spot-instance-policy", `Configurations whether the endpoint should use spot instances.`)
	// TODO: complex arg: tags
	editCmd.Flags().Var(&editReq.WarehouseType, "warehouse-type", `Warehouse type (Classic/Pro).`)

}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: `Update a warehouse.`,
	Long: `Update a warehouse.
  
  Updates the configuration for a SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = editJson.Unmarshall(&editReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if !editNoWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.EditAndWait(ctx, editReq,
				retries.Timeout[sql.GetWarehouseResponse](editTimeout),
				func(i *retries.Info[sql.GetWarehouseResponse]) {
					spinner.Suffix = " " + i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err = w.Warehouses.Edit(ctx, editReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq sql.GetWarehouseRequest

var getNoWait bool
var getTimeout time.Duration

func init() {
	Cmd.AddCommand(getCmd)

	getCmd.Flags().BoolVar(&getNoWait, "no-wait", getNoWait, `do not wait to reach RUNNING state`)
	getCmd.Flags().DurationVar(&getTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Id, "id", getReq.Id, `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get warehouse info.`,
	Long: `Get warehouse info.
  
  Gets the information for a single SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if !getNoWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.GetAndWait(ctx, getReq,
				retries.Timeout[sql.GetWarehouseResponse](getTimeout),
				func(i *retries.Info[sql.GetWarehouseResponse]) {
					spinner.Suffix = " " + i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		response, err := w.Warehouses.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-workspace-warehouse-config command

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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Warehouses.GetWorkspaceWarehouseConfig(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq sql.ListWarehousesRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().IntVar(&listReq.RunAsUserId, "run-as-user-id", listReq.RunAsUserId, `Service Principal which will be used to fetch the list of endpoints.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List warehouses.`,
	Long: `List warehouses.
  
  Lists all SQL warehouses that a user has manager permissions on.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Warehouses.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start set-workspace-warehouse-config command

var setWorkspaceWarehouseConfigReq sql.SetWorkspaceWarehouseConfigRequest
var setWorkspaceWarehouseConfigJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(setWorkspaceWarehouseConfigCmd)
	// TODO: short flags
	setWorkspaceWarehouseConfigCmd.Flags().Var(&setWorkspaceWarehouseConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = setWorkspaceWarehouseConfigJson.Unmarshall(&setWorkspaceWarehouseConfigReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Warehouses.SetWorkspaceWarehouseConfig(ctx, setWorkspaceWarehouseConfigReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start start command

var startReq sql.StartRequest

var startNoWait bool
var startTimeout time.Duration

func init() {
	Cmd.AddCommand(startCmd)

	startCmd.Flags().BoolVar(&startNoWait, "no-wait", startNoWait, `do not wait to reach RUNNING state`)
	startCmd.Flags().DurationVar(&startTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	startCmd.Flags().StringVar(&startReq.Id, "id", startReq.Id, `Required.`)

}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: `Start a warehouse.`,
	Long: `Start a warehouse.
  
  Starts a SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if !startNoWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.StartAndWait(ctx, startReq,
				retries.Timeout[sql.GetWarehouseResponse](startTimeout),
				func(i *retries.Info[sql.GetWarehouseResponse]) {
					spinner.Suffix = " " + i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err = w.Warehouses.Start(ctx, startReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start stop command

var stopReq sql.StopRequest

var stopNoWait bool
var stopTimeout time.Duration

func init() {
	Cmd.AddCommand(stopCmd)

	stopCmd.Flags().BoolVar(&stopNoWait, "no-wait", stopNoWait, `do not wait to reach STOPPED state`)
	stopCmd.Flags().DurationVar(&stopTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach STOPPED state`)
	// TODO: short flags

	stopCmd.Flags().StringVar(&stopReq.Id, "id", stopReq.Id, `Required.`)

}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: `Stop a warehouse.`,
	Long: `Stop a warehouse.
  
  Stops a SQL warehouse.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if !stopNoWait {
			spinner := ui.StartSpinner()
			info, err := w.Warehouses.StopAndWait(ctx, stopReq,
				retries.Timeout[sql.GetWarehouseResponse](stopTimeout),
				func(i *retries.Info[sql.GetWarehouseResponse]) {
					spinner.Suffix = " " + i.Info.Health.Summary
				})
			spinner.Stop()
			if err != nil {
				return err
			}
			return ui.Render(cmd, info)
		}
		err = w.Warehouses.Stop(ctx, stopReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Warehouses
