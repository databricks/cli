package serving_endpoints

import (
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

// The generated help for these commands only lists --json, and most of their
// request body has no flags, so users and agents had to guess the JSON shape or
// read the SDK source. These examples were each checked against the API.

const createExamples = `

  Example: serve a Unity Catalog model version on CPU.

    databricks serving-endpoints create --json '{
      "name": "my-endpoint",
      "config": {
        "served_entities": [{
          "entity_name": "main.default.my_model",
          "entity_version": "1",
          "workload_size": "Small",
          "scale_to_zero_enabled": true
        }]
      }
    }'`

const updateConfigExamples = `

  Example: serve two model versions with an 80/20 traffic split. Each route's
  served_entity_name refers to a served entity's "name".

    databricks serving-endpoints update-config my-endpoint --json '{
      "served_entities": [
        {"name": "v1", "entity_name": "main.default.my_model", "entity_version": "1",
         "workload_size": "Small", "scale_to_zero_enabled": true},
        {"name": "v2", "entity_name": "main.default.my_model", "entity_version": "2",
         "workload_size": "Small", "scale_to_zero_enabled": true}
      ],
      "traffic_config": {"routes": [
        {"served_entity_name": "v1", "traffic_percentage": 80},
        {"served_entity_name": "v2", "traffic_percentage": 20}
      ]}
    }'`

const putAiGatewayExamples = `

  Example: usage tracking, an inference table, a per-user rate limit and a PII
  input guardrail. Settings left out of the request are removed, except
  inference_table_config: to turn the inference table off, send it with
  "enabled": false. Guardrails are only supported on some endpoint types, such
  as external models.

    databricks serving-endpoints put-ai-gateway my-endpoint --json '{
      "usage_tracking_config": {"enabled": true},
      "inference_table_config": {
        "enabled": true,
        "catalog_name": "main",
        "schema_name": "default",
        "table_name_prefix": "my_endpoint"
      },
      "rate_limits": [{"calls": 10, "key": "user", "renewal_period": "minute"}],
      "guardrails": {"input": {"pii": {"behavior": "BLOCK"}}}
    }'`

const patchTelemetryConfigExamples = `

  Example: export logs, traces and metrics to Unity Catalog tables. Leave out
  enabled_telemetry_features to enable all of them.

    databricks serving-endpoints patch-telemetry-config my-endpoint --json '{
      "telemetry_config": {
        "enabled_telemetry_features": [
          "TELEMETRY_FEATURE_LOGS",
          "TELEMETRY_FEATURE_TRACES",
          "TELEMETRY_FEATURE_METRICS"
        ],
        "table_names": {
          "logs_table": "main.default.my_endpoint_logs",
          "traces_table": "main.default.my_endpoint_traces",
          "metrics_table": "main.default.my_endpoint_metrics"
        }
      }
    }'`

const queryExamples = `

  Examples: the request body depends on the endpoint.

    # A custom model, one row per record (column names from the model signature)
    databricks serving-endpoints query my-endpoint --json '{"dataframe_records": [{"x": 1.0, "y": 2.0}]}'

    # The same rows in split format
    databricks serving-endpoints query my-endpoint --json '{"dataframe_split": {"columns": ["x", "y"], "data": [[1.0, 2.0]]}}'

    # The same rows as "inputs", keyed by column name
    databricks serving-endpoints query my-endpoint --json '{"inputs": {"x": [1.0], "y": [2.0]}}'

    # A model with a tensor signature (no column names)
    databricks serving-endpoints query my-endpoint --json '{"inputs": [[1.0, 2.0]]}'

    # A chat model
    databricks serving-endpoints query my-chat-endpoint --json '{"messages": [{"role": "user", "content": "Hello"}], "max_tokens": 100}'

  Use "serving-endpoints get-open-api NAME" to see the request schema of an
  endpoint.`

// appendHelp returns an override that appends text to a command's help.
func appendHelp[T any](text string) func(*cobra.Command, T) {
	return func(cmd *cobra.Command, _ T) {
		cmd.Long += text
	}
}

func init() {
	createOverrides = append(createOverrides, appendHelp[*serving.CreateServingEndpoint](createExamples))
	updateConfigOverrides = append(updateConfigOverrides, appendHelp[*serving.EndpointCoreConfigInput](updateConfigExamples))
	putAiGatewayOverrides = append(putAiGatewayOverrides, appendHelp[*serving.PutAiGatewayRequest](putAiGatewayExamples))
	patchTelemetryConfigOverrides = append(patchTelemetryConfigOverrides, appendHelp[*serving.PatchTelemetryConfigRequest](patchTelemetryConfigExamples))
	queryOverrides = append(queryOverrides, appendHelp[*serving.QueryEndpointInput](queryExamples))
}
