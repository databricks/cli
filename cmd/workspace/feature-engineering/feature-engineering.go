// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package feature_engineering

import (
	"errors"
	"fmt"
	"time"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	features "github.com/databricks/cli/libs/sdktypes/features/v1"
	sdkfeatures "github.com/databricks/sdk-go/features/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature-engineering",
		Short: `*Public Preview* Feature Engineering provides APIs for managing features and materialized features.`,
		Long: `This command is in Public Preview and may change without notice.

Feature Engineering provides APIs for managing features and materialized
  features.`,
		GroupID: "ml",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newBackfillFeatures())
	cmd.AddCommand(newCancelOperation())
	cmd.AddCommand(newCreateFeature())
	cmd.AddCommand(newCreateKafkaConfig())
	cmd.AddCommand(newCreateMaterializedFeature())
	cmd.AddCommand(newCreateStream())
	cmd.AddCommand(newDeleteFeature())
	cmd.AddCommand(newDeleteKafkaConfig())
	cmd.AddCommand(newDeleteMaterializedFeature())
	cmd.AddCommand(newDeleteStream())
	cmd.AddCommand(newGetFeature())
	cmd.AddCommand(newGetKafkaConfig())
	cmd.AddCommand(newGetMaterializedFeature())
	cmd.AddCommand(newGetOperation())
	cmd.AddCommand(newGetStream())
	cmd.AddCommand(newListFeatures())
	cmd.AddCommand(newListKafkaConfigs())
	cmd.AddCommand(newListMaterializedFeatures())
	cmd.AddCommand(newListStreams())
	cmd.AddCommand(newPurgeFeatureEntities())
	cmd.AddCommand(newUpdateFeature())
	cmd.AddCommand(newUpdateKafkaConfig())
	cmd.AddCommand(newUpdateMaterializedFeature())
	cmd.AddCommand(newUpdateStream())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start backfill-features command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var backfillFeaturesOverrides []func(
	*cobra.Command,
	*features.BackfillFeaturesRequest,
)

func newBackfillFeatures() *cobra.Command {
	cmd := &cobra.Command{}
	var backfillFeaturesReq features.BackfillFeaturesRequest

	var backfillFeaturesJson flags.JsonFlag

	var backfillFeaturesSkipWait bool
	var backfillFeaturesTimeout time.Duration

	cmd.Flags().BoolVar(&backfillFeaturesSkipWait, "no-wait", backfillFeaturesSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&backfillFeaturesTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&backfillFeaturesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &backfillFeaturesReq.BudgetPolicyId, "budget-policy-id", `The budget policy ID, in UUID format, used to attribute the serverless compute cost of this backfill.`)
	flags.StringPtrVar(cmd.Flags(), &backfillFeaturesReq.RequestId, "request-id", `Idempotency token for the request.`)
	// TODO: map via StringToStringVar: tags

	cmd.Use = "backfill-features"
	cmd.Short = `Backfill features.`
	cmd.Long = `Backfill features.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := backfillFeaturesJson.Unmarshal(&backfillFeaturesReq)
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
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := features.BackfillFeaturesRequestToSDK(&backfillFeaturesReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case backfillFeaturesSkipWait:
			wait, err := w.Features.BackfillFeatures(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Features.BackfillFeatures(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for backfill-features to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(backfillFeaturesTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := features.BackfillFeaturesResponseFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range backfillFeaturesOverrides {
		fn(cmd, &backfillFeaturesReq)
	}

	return cmd
}

// start cancel-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelOperationOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newCancelOperation() *cobra.Command {
	cmd := &cobra.Command{}
	var cancelOperationReq map[string]any
	cancelOperationReq = map[string]any{}
	var cancelOperationNameRaw string

	cmd.Use = "cancel-operation NAME"
	cmd.Short = `Cancel an operation.`
	cmd.Long = `Cancel an operation.

  Arguments:
    NAME: The name of the operation resource to be cancelled.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		cancelOperationNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/feature-engineering/%s:cancel", fmt.Sprint(cancelOperationNameRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, cancelOperationReq, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelOperationOverrides {
		fn(cmd, &cancelOperationReq)
	}

	return cmd
}

// start create-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createFeatureOverrides []func(
	*cobra.Command,
	*features.CreateFeatureRequest,
)

func newCreateFeature() *cobra.Command {
	cmd := &cobra.Command{}
	var createFeatureReq features.CreateFeatureRequest
	createFeatureReq.Feature = &features.Feature{}

	var createFeatureJson flags.JsonFlag

	cmd.Flags().Var(&createFeatureJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createFeatureReq.Feature.Description, "description", `The description of the feature.`)
	// TODO: array: entities
	flags.StringPtrVar(cmd.Flags(), &createFeatureReq.Feature.FilterCondition, "filter-condition", ``)
	// TODO: array: inputs
	// TODO: complex arg: lineage_context
	// TODO: complex arg: time_window
	// TODO: complex arg: timeseries_column

	cmd.Use = "create-feature FULL_NAME SOURCE FUNCTION"
	cmd.Short = `*Public Preview* Create a feature.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a feature.

  Create a Feature.

  Arguments:
    FULL_NAME: The full three-part name (catalog, schema, name) of the feature.
    SOURCE: The data source of the feature.
    FUNCTION: The function by which the feature is computed.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'full_name', 'source', 'function' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createFeatureJson.Unmarshal(&createFeatureReq.Feature)
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
		if !cmd.Flags().Changed("json") {
			createFeatureReq.Feature.FullName = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[1] // TODO: complex arg: source
		}
		if !cmd.Flags().Changed("json") {
			_ = args[2] // TODO: complex arg: function
		}

		sdkReq, err := features.CreateFeatureRequestToSDK(&createFeatureReq)
		if err != nil {
			return err
		}
		response, err := w.Features.CreateFeature(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.FeatureFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createFeatureOverrides {
		fn(cmd, &createFeatureReq)
	}

	return cmd
}

// start create-kafka-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createKafkaConfigOverrides []func(
	*cobra.Command,
	*features.CreateKafkaConfigRequest,
)

func newCreateKafkaConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var createKafkaConfigReq features.CreateKafkaConfigRequest
	createKafkaConfigReq.KafkaConfig = &features.KafkaConfig{}

	var createKafkaConfigJson flags.JsonFlag

	cmd.Flags().Var(&createKafkaConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: backfill_source
	// TODO: map via StringToStringVar: extra_options
	// TODO: complex arg: ingestion_config
	// TODO: complex arg: key_schema
	// TODO: complex arg: value_schema

	cmd.Use = "create-kafka-config NAME BOOTSTRAP_SERVERS SUBSCRIPTION_MODE AUTH_CONFIG"
	cmd.Short = `Create a Kafka config.`
	cmd.Long = `Create a Kafka config.

  Create a Kafka config. During PrPr, Kafka configs can be read and used when
  creating features under the entire metastore. Only the creator of the Kafka
  config can delete it.

  Arguments:
    NAME: Name that uniquely identifies this Kafka config within the metastore. This
      will be the identifier used from the Feature object to reference these
      configs for a feature. Can be distinct from topic name.
    BOOTSTRAP_SERVERS: A comma-separated list of host/port pairs pointing to Kafka cluster.
    SUBSCRIPTION_MODE: Options to configure which Kafka topics to pull data from.
    AUTH_CONFIG: Authentication configuration for connection to topics.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'bootstrap_servers', 'subscription_mode', 'auth_config' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createKafkaConfigJson.Unmarshal(&createKafkaConfigReq.KafkaConfig)
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
		if !cmd.Flags().Changed("json") {
			createKafkaConfigReq.KafkaConfig.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createKafkaConfigReq.KafkaConfig.BootstrapServers = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[2] // TODO: complex arg: subscription_mode
		}
		if !cmd.Flags().Changed("json") {
			_ = args[3] // TODO: complex arg: auth_config
		}

		sdkReq, err := features.CreateKafkaConfigRequestToSDK(&createKafkaConfigReq)
		if err != nil {
			return err
		}
		response, err := w.Features.CreateKafkaConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.KafkaConfigFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createKafkaConfigOverrides {
		fn(cmd, &createKafkaConfigReq)
	}

	return cmd
}

// start create-materialized-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMaterializedFeatureOverrides []func(
	*cobra.Command,
	*features.CreateMaterializedFeatureRequest,
)

func newCreateMaterializedFeature() *cobra.Command {
	cmd := &cobra.Command{}
	var createMaterializedFeatureReq features.CreateMaterializedFeatureRequest
	createMaterializedFeatureReq.MaterializedFeature = &features.MaterializedFeature{}

	var createMaterializedFeatureJson flags.JsonFlag

	cmd.Flags().Var(&createMaterializedFeatureJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createMaterializedFeatureReq.MaterializedFeature.BudgetPolicyId, "budget-policy-id", `The ID of the budget policy used to attribute the serverless compute cost of this materialization.`)
	flags.StringPtrVar(cmd.Flags(), &createMaterializedFeatureReq.MaterializedFeature.CronSchedule, "cron-schedule", ``)
	// TODO: complex arg: cron_schedule_trigger
	flags.StringPtrVar(cmd.Flags(), &createMaterializedFeatureReq.MaterializedFeature.MaterializedFeatureId, "materialized-feature-id", `Server-assigned unique identifier for the materialized feature.`)
	// TODO: complex arg: notifications
	// TODO: complex arg: offline_store_config
	// TODO: complex arg: online_store_config
	flags.EnumVar(cmd.Flags(), &createMaterializedFeatureReq.MaterializedFeature.PipelineScheduleState, "pipeline-schedule-state", `The schedule state of the materialization pipeline. Supported values: [ACTIVE, PAUSED, SNAPSHOT]`, "MaterializedFeaturePipelineScheduleState", "ACTIVE", "PAUSED", "SNAPSHOT")
	// TODO: complex arg: streaming_mode
	// TODO: complex arg: table_trigger
	// TODO: map via StringToStringVar: tags

	cmd.Use = "create-materialized-feature FEATURE_NAME"
	cmd.Short = `*Public Preview* Create a materialized feature.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a materialized feature.

  Arguments:
    FEATURE_NAME: The full name of the feature in Unity Catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'feature_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createMaterializedFeatureJson.Unmarshal(&createMaterializedFeatureReq.MaterializedFeature)
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
		if !cmd.Flags().Changed("json") {
			createMaterializedFeatureReq.MaterializedFeature.FeatureName = new(args[0])
		}

		sdkReq, err := features.CreateMaterializedFeatureRequestToSDK(&createMaterializedFeatureReq)
		if err != nil {
			return err
		}
		response, err := w.Features.CreateMaterializedFeature(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.MaterializedFeatureFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createMaterializedFeatureOverrides {
		fn(cmd, &createMaterializedFeatureReq)
	}

	return cmd
}

// start create-stream command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createStreamOverrides []func(
	*cobra.Command,
	*features.CreateStreamRequest,
)

func newCreateStream() *cobra.Command {
	cmd := &cobra.Command{}
	var createStreamReq features.CreateStreamRequest
	createStreamReq.Stream = &features.Stream{}

	var createStreamJson flags.JsonFlag

	cmd.Flags().Var(&createStreamJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createStreamReq.Stream.Description, "description", `User-provided description.`)
	// TODO: array: excluded_columns
	flags.StringPtrVar(cmd.Flags(), &createStreamReq.Stream.RecordTypeFilter, "record-type-filter", `Optional SQL predicate to filter which record types from a streaming channel (e.g.`)

	cmd.Use = "create-stream NAME SOURCE_CONFIG CONNECTION_CONFIG SCHEMA_CONFIG INGESTION_CONFIG"
	cmd.Short = `*Public Preview* Create a Stream.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a Stream.

  Create a Stream, a governed UC entity representing an external streaming data
  source.

  Arguments:
    NAME: Full three-part (catalog.schema.stream) name of the stream.
    SOURCE_CONFIG: Source-specific configuration. Determines the streaming platform source.
    CONNECTION_CONFIG: Specifies how to connect and authenticate to the stream platform.
    SCHEMA_CONFIG: Schema definitions for the stream, provided either directly on the Stream
      or resolved from an external schema registry through a UC Connection.
    INGESTION_CONFIG: Configuration for streaming data ingestion: the managed table storing an
      offline copy of forward fill data and optional historical backfill.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'source_config', 'connection_config', 'schema_config', 'ingestion_config' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createStreamJson.Unmarshal(&createStreamReq.Stream)
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
		if !cmd.Flags().Changed("json") {
			createStreamReq.Stream.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[1] // TODO: complex arg: source_config
		}
		if !cmd.Flags().Changed("json") {
			_ = args[2] // TODO: complex arg: connection_config
		}
		if !cmd.Flags().Changed("json") {
			_ = args[3] // TODO: complex arg: schema_config
		}
		if !cmd.Flags().Changed("json") {
			_ = args[4] // TODO: complex arg: ingestion_config
		}

		sdkReq, err := features.CreateStreamRequestToSDK(&createStreamReq)
		if err != nil {
			return err
		}
		response, err := w.Features.CreateStream(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.StreamFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createStreamOverrides {
		fn(cmd, &createStreamReq)
	}

	return cmd
}

// start delete-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteFeatureOverrides []func(
	*cobra.Command,
	*sdkfeatures.DeleteFeatureRequest,
)

func newDeleteFeature() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteFeatureReq sdkfeatures.DeleteFeatureRequest

	cmd.Use = "delete-feature FULL_NAME"
	cmd.Short = `*Public Preview* Delete a feature.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a feature.

  Delete a Feature.

  Arguments:
    FULL_NAME: Name of the feature to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteFeatureReq.FullName = new(args[0])

		sdkReq := &deleteFeatureReq
		err = w.Features.DeleteFeature(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteFeatureOverrides {
		fn(cmd, &deleteFeatureReq)
	}

	return cmd
}

// start delete-kafka-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteKafkaConfigOverrides []func(
	*cobra.Command,
	*sdkfeatures.DeleteKafkaConfigRequest,
)

func newDeleteKafkaConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteKafkaConfigReq sdkfeatures.DeleteKafkaConfigRequest

	cmd.Use = "delete-kafka-config NAME"
	cmd.Short = `Delete a Kafka config.`
	cmd.Long = `Delete a Kafka config.

  Delete a Kafka config. During PrPr, Kafka configs can be read and used when
  creating features under the entire metastore. Only the creator of the Kafka
  config can delete it.

  Arguments:
    NAME: Name of the Kafka config to delete.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteKafkaConfigReq.Name = new(args[0])

		sdkReq := &deleteKafkaConfigReq
		err = w.Features.DeleteKafkaConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteKafkaConfigOverrides {
		fn(cmd, &deleteKafkaConfigReq)
	}

	return cmd
}

// start delete-materialized-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteMaterializedFeatureOverrides []func(
	*cobra.Command,
	*sdkfeatures.DeleteMaterializedFeatureRequest,
)

func newDeleteMaterializedFeature() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteMaterializedFeatureReq sdkfeatures.DeleteMaterializedFeatureRequest

	cmd.Use = "delete-materialized-feature MATERIALIZED_FEATURE_ID"
	cmd.Short = `*Public Preview* Delete a materialized feature.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a materialized feature.

  Arguments:
    MATERIALIZED_FEATURE_ID: The ID of the materialized feature to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteMaterializedFeatureReq.MaterializedFeatureId = new(args[0])

		sdkReq := &deleteMaterializedFeatureReq
		err = w.Features.DeleteMaterializedFeature(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteMaterializedFeatureOverrides {
		fn(cmd, &deleteMaterializedFeatureReq)
	}

	return cmd
}

// start delete-stream command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteStreamOverrides []func(
	*cobra.Command,
	*sdkfeatures.DeleteStreamRequest,
)

func newDeleteStream() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteStreamReq sdkfeatures.DeleteStreamRequest

	cmd.Use = "delete-stream NAME"
	cmd.Short = `*Public Preview* Delete a Stream.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a Stream.

  Delete a Stream by its full three-part name (catalog.schema.stream).

  Arguments:
    NAME: Full three-part name (catalog.schema.stream) of the Stream to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteStreamReq.Name = new(args[0])

		sdkReq := &deleteStreamReq
		err = w.Features.DeleteStream(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteStreamOverrides {
		fn(cmd, &deleteStreamReq)
	}

	return cmd
}

// start get-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getFeatureOverrides []func(
	*cobra.Command,
	*sdkfeatures.GetFeatureRequest,
)

func newGetFeature() *cobra.Command {
	cmd := &cobra.Command{}
	var getFeatureReq sdkfeatures.GetFeatureRequest

	cmd.Use = "get-feature FULL_NAME"
	cmd.Short = `*Public Preview* Get a feature.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a feature.

  Get a Feature.

  Arguments:
    FULL_NAME: Name of the feature to get.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getFeatureReq.FullName = new(args[0])

		sdkReq := &getFeatureReq
		response, err := w.Features.GetFeature(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.FeatureFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getFeatureOverrides {
		fn(cmd, &getFeatureReq)
	}

	return cmd
}

// start get-kafka-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getKafkaConfigOverrides []func(
	*cobra.Command,
	*sdkfeatures.GetKafkaConfigRequest,
)

func newGetKafkaConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var getKafkaConfigReq sdkfeatures.GetKafkaConfigRequest

	cmd.Use = "get-kafka-config NAME"
	cmd.Short = `Get a Kafka config.`
	cmd.Long = `Get a Kafka config.

  Get a Kafka config. During PrPr, Kafka configs can be read and used when
  creating features under the entire metastore. Only the creator of the Kafka
  config can delete it.

  Arguments:
    NAME: Name of the Kafka config to get.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getKafkaConfigReq.Name = new(args[0])

		sdkReq := &getKafkaConfigReq
		response, err := w.Features.GetKafkaConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.KafkaConfigFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getKafkaConfigOverrides {
		fn(cmd, &getKafkaConfigReq)
	}

	return cmd
}

// start get-materialized-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMaterializedFeatureOverrides []func(
	*cobra.Command,
	*sdkfeatures.GetMaterializedFeatureRequest,
)

func newGetMaterializedFeature() *cobra.Command {
	cmd := &cobra.Command{}
	var getMaterializedFeatureReq sdkfeatures.GetMaterializedFeatureRequest

	cmd.Use = "get-materialized-feature MATERIALIZED_FEATURE_ID"
	cmd.Short = `*Public Preview* Get a materialized feature.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a materialized feature.

  Arguments:
    MATERIALIZED_FEATURE_ID: The ID of the materialized feature.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getMaterializedFeatureReq.MaterializedFeatureId = new(args[0])

		sdkReq := &getMaterializedFeatureReq
		response, err := w.Features.GetMaterializedFeature(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.MaterializedFeatureFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMaterializedFeatureOverrides {
		fn(cmd, &getMaterializedFeatureReq)
	}

	return cmd
}

// start get-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOperationOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newGetOperation() *cobra.Command {
	cmd := &cobra.Command{}
	var getOperationReq map[string]any
	getOperationReq = map[string]any{}
	var getOperationNameRaw string

	cmd.Use = "get-operation NAME"
	cmd.Short = `Get an operation.`
	cmd.Long = `Get an operation.

  Arguments:
    NAME: The name of the operation resource.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getOperationNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/feature-engineering/%s", fmt.Sprint(getOperationNameRaw))

		maps.Copy(rawQuery, getOperationReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOperationOverrides {
		fn(cmd, &getOperationReq)
	}

	return cmd
}

// start get-stream command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getStreamOverrides []func(
	*cobra.Command,
	*sdkfeatures.GetStreamRequest,
)

func newGetStream() *cobra.Command {
	cmd := &cobra.Command{}
	var getStreamReq sdkfeatures.GetStreamRequest

	cmd.Use = "get-stream NAME"
	cmd.Short = `*Public Preview* Get a Stream.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a Stream.

  Get a Stream by its full three-part name (catalog.schema.stream).

  Arguments:
    NAME: Full three-part name (catalog.schema.stream) of the Stream to get.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getStreamReq.Name = new(args[0])

		sdkReq := &getStreamReq
		response, err := w.Features.GetStream(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.StreamFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getStreamOverrides {
		fn(cmd, &getStreamReq)
	}

	return cmd
}

// start list-features command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listFeaturesOverrides []func(
	*cobra.Command,
	*features.ListFeaturesRequest,
)

func newListFeatures() *cobra.Command {
	cmd := &cobra.Command{}
	var listFeaturesReq features.ListFeaturesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listFeaturesLimit int

	flags.IntPtrVar(cmd.Flags(), &listFeaturesReq.PageSize, "page-size", `The maximum number of results to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listFeaturesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listFeaturesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-features CATALOG_NAME SCHEMA_NAME"
	cmd.Short = `*Public Preview* List features.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List features.

  List Features.

  Arguments:
    CATALOG_NAME: Name of parent catalog for features of interest.
    SCHEMA_NAME: Name of parent schema relative to its parent catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listFeaturesReq.CatalogName = new(args[0])
		listFeaturesReq.SchemaName = new(args[1])

		sdkReq, err := features.ListFeaturesRequestToSDK(&listFeaturesReq)
		if err != nil {
			return err
		}
		if listFeaturesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listFeaturesLimit)
		}
		if listFeaturesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listFeaturesLimit)
		}
		response := w.Features.ListFeaturesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, features.FeatureFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listFeaturesOverrides {
		fn(cmd, &listFeaturesReq)
	}

	return cmd
}

// start list-kafka-configs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listKafkaConfigsOverrides []func(
	*cobra.Command,
	*features.ListKafkaConfigsRequest,
)

func newListKafkaConfigs() *cobra.Command {
	cmd := &cobra.Command{}
	var listKafkaConfigsReq features.ListKafkaConfigsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listKafkaConfigsLimit int

	flags.IntPtrVar(cmd.Flags(), &listKafkaConfigsReq.PageSize, "page-size", `The maximum number of results to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listKafkaConfigsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listKafkaConfigsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-kafka-configs"
	cmd.Short = `List Kafka configs.`
	cmd.Long = `List Kafka configs.

  List Kafka configs. During PrPr, Kafka configs can be read and used when
  creating features under the entire metastore. Only the creator of the Kafka
  config can delete it.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := features.ListKafkaConfigsRequestToSDK(&listKafkaConfigsReq)
		if err != nil {
			return err
		}
		if listKafkaConfigsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listKafkaConfigsLimit)
		}
		if listKafkaConfigsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listKafkaConfigsLimit)
		}
		response := w.Features.ListKafkaConfigsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, features.KafkaConfigFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listKafkaConfigsOverrides {
		fn(cmd, &listKafkaConfigsReq)
	}

	return cmd
}

// start list-materialized-features command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMaterializedFeaturesOverrides []func(
	*cobra.Command,
	*features.ListMaterializedFeaturesRequest,
)

func newListMaterializedFeatures() *cobra.Command {
	cmd := &cobra.Command{}
	var listMaterializedFeaturesReq features.ListMaterializedFeaturesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMaterializedFeaturesLimit int

	flags.StringPtrVar(cmd.Flags(), &listMaterializedFeaturesReq.FeatureName, "feature-name", `Filter by feature name.`)
	flags.IntPtrVar(cmd.Flags(), &listMaterializedFeaturesReq.PageSize, "page-size", `The maximum number of results to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMaterializedFeaturesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listMaterializedFeaturesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-materialized-features"
	cmd.Short = `*Public Preview* List materialized features.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List materialized features.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := features.ListMaterializedFeaturesRequestToSDK(&listMaterializedFeaturesReq)
		if err != nil {
			return err
		}
		if listMaterializedFeaturesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listMaterializedFeaturesLimit)
		}
		if listMaterializedFeaturesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listMaterializedFeaturesLimit)
		}
		response := w.Features.ListMaterializedFeaturesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, features.MaterializedFeatureFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listMaterializedFeaturesOverrides {
		fn(cmd, &listMaterializedFeaturesReq)
	}

	return cmd
}

// start list-streams command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listStreamsOverrides []func(
	*cobra.Command,
	*features.ListStreamsRequest,
)

func newListStreams() *cobra.Command {
	cmd := &cobra.Command{}
	var listStreamsReq features.ListStreamsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listStreamsLimit int

	flags.IntPtrVar(cmd.Flags(), &listStreamsReq.PageSize, "page-size", `The maximum number of results to return.`)
	flags.StringPtrVar(cmd.Flags(), &listStreamsReq.Parent, "parent", `Two-part name (catalog.schema) of the parent under which to list Streams.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listStreamsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listStreamsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-streams"
	cmd.Short = `*Public Preview* List Streams.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List Streams.

  List Streams under a given catalog.schema parent.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := features.ListStreamsRequestToSDK(&listStreamsReq)
		if err != nil {
			return err
		}
		if listStreamsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listStreamsLimit)
		}
		if listStreamsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listStreamsLimit)
		}
		response := w.Features.ListStreamsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, features.StreamFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listStreamsOverrides {
		fn(cmd, &listStreamsReq)
	}

	return cmd
}

// start purge-feature-entities command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var purgeFeatureEntitiesOverrides []func(
	*cobra.Command,
	*features.PurgeFeatureEntitiesRequest,
)

func newPurgeFeatureEntities() *cobra.Command {
	cmd := &cobra.Command{}
	var purgeFeatureEntitiesReq features.PurgeFeatureEntitiesRequest

	var purgeFeatureEntitiesJson flags.JsonFlag

	var purgeFeatureEntitiesSkipWait bool
	var purgeFeatureEntitiesTimeout time.Duration

	cmd.Flags().BoolVar(&purgeFeatureEntitiesSkipWait, "no-wait", purgeFeatureEntitiesSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&purgeFeatureEntitiesTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&purgeFeatureEntitiesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &purgeFeatureEntitiesReq.BudgetPolicyId, "budget-policy-id", `The budget policy ID, in UUID format, used to attribute the serverless compute cost of this purge.`)
	flags.StringPtrVar(cmd.Flags(), &purgeFeatureEntitiesReq.RequestId, "request-id", `Optional UUID4 idempotency token for the request.`)
	// TODO: map via StringToStringVar: tags

	cmd.Use = "purge-feature-entities"
	cmd.Short = `Purge feature values for entities.`
	cmd.Long = `Purge feature values for entities.

  Purge materialized feature values for specified entities.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := purgeFeatureEntitiesJson.Unmarshal(&purgeFeatureEntitiesReq)
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
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := features.PurgeFeatureEntitiesRequestToSDK(&purgeFeatureEntitiesReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case purgeFeatureEntitiesSkipWait:
			wait, err := w.Features.PurgeFeatureEntities(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Features.PurgeFeatureEntities(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for purge-feature-entities to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(purgeFeatureEntitiesTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := features.PurgeFeatureEntitiesResponseFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range purgeFeatureEntitiesOverrides {
		fn(cmd, &purgeFeatureEntitiesReq)
	}

	return cmd
}

// start update-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateFeatureOverrides []func(
	*cobra.Command,
	*features.UpdateFeatureRequest,
)

func newUpdateFeature() *cobra.Command {
	cmd := &cobra.Command{}
	var updateFeatureReq features.UpdateFeatureRequest
	updateFeatureReq.Feature = &features.Feature{}

	var updateFeatureJson flags.JsonFlag

	cmd.Flags().Var(&updateFeatureJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateFeatureReq.Feature.Description, "description", `The description of the feature.`)
	// TODO: array: entities
	flags.StringPtrVar(cmd.Flags(), &updateFeatureReq.Feature.FilterCondition, "filter-condition", ``)
	// TODO: array: inputs
	// TODO: complex arg: lineage_context
	// TODO: complex arg: time_window
	// TODO: complex arg: timeseries_column

	cmd.Use = "update-feature FULL_NAME UPDATE_MASK SOURCE FUNCTION"
	cmd.Short = `*Public Preview* Update a feature's description (all other fields are immutable).`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update a feature's description (all other fields are immutable).

  Update a Feature.

  Arguments:
    FULL_NAME: The full three-part name (catalog, schema, name) of the feature.
    UPDATE_MASK: Fields to update. The only supported path is description.
    SOURCE: The data source of the feature.
    FUNCTION: The function by which the feature is computed.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only FULL_NAME, UPDATE_MASK as positional arguments. Provide 'full_name', 'source', 'function' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateFeatureJson.Unmarshal(&updateFeatureReq.Feature)
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
		updateFeatureReq.Feature.FullName = new(args[0])
		updateFeatureReq.UpdateMask = new(args[1])
		if !cmd.Flags().Changed("json") {
			_ = args[2] // TODO: complex arg: source
		}
		if !cmd.Flags().Changed("json") {
			_ = args[3] // TODO: complex arg: function
		}

		sdkReq, err := features.UpdateFeatureRequestToSDK(&updateFeatureReq)
		if err != nil {
			return err
		}
		response, err := w.Features.UpdateFeature(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.FeatureFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateFeatureOverrides {
		fn(cmd, &updateFeatureReq)
	}

	return cmd
}

// start update-kafka-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateKafkaConfigOverrides []func(
	*cobra.Command,
	*features.UpdateKafkaConfigRequest,
)

func newUpdateKafkaConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var updateKafkaConfigReq features.UpdateKafkaConfigRequest
	updateKafkaConfigReq.KafkaConfig = &features.KafkaConfig{}

	var updateKafkaConfigJson flags.JsonFlag

	cmd.Flags().Var(&updateKafkaConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: backfill_source
	// TODO: map via StringToStringVar: extra_options
	// TODO: complex arg: ingestion_config
	// TODO: complex arg: key_schema
	// TODO: complex arg: value_schema

	cmd.Use = "update-kafka-config NAME UPDATE_MASK BOOTSTRAP_SERVERS SUBSCRIPTION_MODE AUTH_CONFIG"
	cmd.Short = `Update a Kafka config.`
	cmd.Long = `Update a Kafka config.

  Update a Kafka config. During PrPr, Kafka configs can be read and used when
  creating features under the entire metastore. Only the creator of the Kafka
  config can delete it.

  Arguments:
    NAME: Name that uniquely identifies this Kafka config within the metastore. This
      will be the identifier used from the Feature object to reference these
      configs for a feature. Can be distinct from topic name.
    UPDATE_MASK: The list of fields to update.
    BOOTSTRAP_SERVERS: A comma-separated list of host/port pairs pointing to Kafka cluster.
    SUBSCRIPTION_MODE: Options to configure which Kafka topics to pull data from.
    AUTH_CONFIG: Authentication configuration for connection to topics.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'name', 'bootstrap_servers', 'subscription_mode', 'auth_config' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateKafkaConfigJson.Unmarshal(&updateKafkaConfigReq.KafkaConfig)
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
		updateKafkaConfigReq.KafkaConfig.Name = new(args[0])
		if args[1] != "" {
			updateKafkaConfigReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateKafkaConfigReq.KafkaConfig.BootstrapServers = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[3] // TODO: complex arg: subscription_mode
		}
		if !cmd.Flags().Changed("json") {
			_ = args[4] // TODO: complex arg: auth_config
		}

		sdkReq, err := features.UpdateKafkaConfigRequestToSDK(&updateKafkaConfigReq)
		if err != nil {
			return err
		}
		response, err := w.Features.UpdateKafkaConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.KafkaConfigFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateKafkaConfigOverrides {
		fn(cmd, &updateKafkaConfigReq)
	}

	return cmd
}

// start update-materialized-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateMaterializedFeatureOverrides []func(
	*cobra.Command,
	*features.UpdateMaterializedFeatureRequest,
)

func newUpdateMaterializedFeature() *cobra.Command {
	cmd := &cobra.Command{}
	var updateMaterializedFeatureReq features.UpdateMaterializedFeatureRequest
	updateMaterializedFeatureReq.MaterializedFeature = &features.MaterializedFeature{}

	var updateMaterializedFeatureJson flags.JsonFlag

	cmd.Flags().Var(&updateMaterializedFeatureJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateMaterializedFeatureReq.MaterializedFeature.BudgetPolicyId, "budget-policy-id", `The ID of the budget policy used to attribute the serverless compute cost of this materialization.`)
	flags.StringPtrVar(cmd.Flags(), &updateMaterializedFeatureReq.MaterializedFeature.CronSchedule, "cron-schedule", ``)
	// TODO: complex arg: cron_schedule_trigger
	flags.StringPtrVar(cmd.Flags(), &updateMaterializedFeatureReq.MaterializedFeature.MaterializedFeatureId, "materialized-feature-id", `Server-assigned unique identifier for the materialized feature.`)
	// TODO: complex arg: notifications
	// TODO: complex arg: offline_store_config
	// TODO: complex arg: online_store_config
	flags.EnumVar(cmd.Flags(), &updateMaterializedFeatureReq.MaterializedFeature.PipelineScheduleState, "pipeline-schedule-state", `The schedule state of the materialization pipeline. Supported values: [ACTIVE, PAUSED, SNAPSHOT]`, "MaterializedFeaturePipelineScheduleState", "ACTIVE", "PAUSED", "SNAPSHOT")
	// TODO: complex arg: streaming_mode
	// TODO: complex arg: table_trigger
	// TODO: map via StringToStringVar: tags

	cmd.Use = "update-materialized-feature MATERIALIZED_FEATURE_ID UPDATE_MASK FEATURE_NAME"
	cmd.Short = `*Public Preview* Update a materialized feature.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update a materialized feature.

  Update a materialized feature (pause/resume).

  Arguments:
    MATERIALIZED_FEATURE_ID: Server-assigned unique identifier for the materialized feature.
    UPDATE_MASK: Provide the materialization feature fields which should be updated.
      Currently, only the pipeline_state field can be updated.
    FEATURE_NAME: The full name of the feature in Unity Catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only MATERIALIZED_FEATURE_ID, UPDATE_MASK as positional arguments. Provide 'feature_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateMaterializedFeatureJson.Unmarshal(&updateMaterializedFeatureReq.MaterializedFeature)
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
		updateMaterializedFeatureReq.MaterializedFeature.MaterializedFeatureId = new(args[0])
		updateMaterializedFeatureReq.UpdateMask = new(args[1])
		if !cmd.Flags().Changed("json") {
			updateMaterializedFeatureReq.MaterializedFeature.FeatureName = new(args[2])
		}

		sdkReq, err := features.UpdateMaterializedFeatureRequestToSDK(&updateMaterializedFeatureReq)
		if err != nil {
			return err
		}
		response, err := w.Features.UpdateMaterializedFeature(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.MaterializedFeatureFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateMaterializedFeatureOverrides {
		fn(cmd, &updateMaterializedFeatureReq)
	}

	return cmd
}

// start update-stream command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateStreamOverrides []func(
	*cobra.Command,
	*features.UpdateStreamRequest,
)

func newUpdateStream() *cobra.Command {
	cmd := &cobra.Command{}
	var updateStreamReq features.UpdateStreamRequest
	updateStreamReq.Stream = &features.Stream{}

	var updateStreamJson flags.JsonFlag

	cmd.Flags().Var(&updateStreamJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateStreamReq.Stream.Description, "description", `User-provided description.`)
	// TODO: array: excluded_columns
	flags.StringPtrVar(cmd.Flags(), &updateStreamReq.Stream.RecordTypeFilter, "record-type-filter", `Optional SQL predicate to filter which record types from a streaming channel (e.g.`)

	cmd.Use = "update-stream NAME UPDATE_MASK SOURCE_CONFIG CONNECTION_CONFIG SCHEMA_CONFIG INGESTION_CONFIG"
	cmd.Short = `*Public Preview* Update a Stream.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update a Stream.

  Update a Stream. Only fields listed in update_mask are mutated.

  Arguments:
    NAME: Full three-part (catalog.schema.stream) name of the stream.
    UPDATE_MASK: The list of fields to update.
    SOURCE_CONFIG: Source-specific configuration. Determines the streaming platform source.
    CONNECTION_CONFIG: Specifies how to connect and authenticate to the stream platform.
    SCHEMA_CONFIG: Schema definitions for the stream, provided either directly on the Stream
      or resolved from an external schema registry through a UC Connection.
    INGESTION_CONFIG: Configuration for streaming data ingestion: the managed table storing an
      offline copy of forward fill data and optional historical backfill.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'name', 'source_config', 'connection_config', 'schema_config', 'ingestion_config' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateStreamJson.Unmarshal(&updateStreamReq.Stream)
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
		updateStreamReq.Stream.Name = new(args[0])
		if args[1] != "" {
			updateStreamReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[2] // TODO: complex arg: source_config
		}
		if !cmd.Flags().Changed("json") {
			_ = args[3] // TODO: complex arg: connection_config
		}
		if !cmd.Flags().Changed("json") {
			_ = args[4] // TODO: complex arg: schema_config
		}
		if !cmd.Flags().Changed("json") {
			_ = args[5] // TODO: complex arg: ingestion_config
		}

		sdkReq, err := features.UpdateStreamRequestToSDK(&updateStreamReq)
		if err != nil {
			return err
		}
		response, err := w.Features.UpdateStream(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := features.StreamFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateStreamOverrides {
		fn(cmd, &updateStreamReq)
	}

	return cmd
}

// end service FeatureEngineering
