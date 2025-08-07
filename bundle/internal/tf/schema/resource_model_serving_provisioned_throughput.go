// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceModelServingProvisionedThroughputAiGatewayFallbackConfig struct {
	Enabled bool `json:"enabled"`
}

type ResourceModelServingProvisionedThroughputAiGatewayGuardrailsInputPii struct {
	Behavior string `json:"behavior,omitempty"`
}

type ResourceModelServingProvisionedThroughputAiGatewayGuardrailsInput struct {
	InvalidKeywords []string                                                              `json:"invalid_keywords,omitempty"`
	Safety          bool                                                                  `json:"safety,omitempty"`
	ValidTopics     []string                                                              `json:"valid_topics,omitempty"`
	Pii             *ResourceModelServingProvisionedThroughputAiGatewayGuardrailsInputPii `json:"pii,omitempty"`
}

type ResourceModelServingProvisionedThroughputAiGatewayGuardrailsOutputPii struct {
	Behavior string `json:"behavior,omitempty"`
}

type ResourceModelServingProvisionedThroughputAiGatewayGuardrailsOutput struct {
	InvalidKeywords []string                                                               `json:"invalid_keywords,omitempty"`
	Safety          bool                                                                   `json:"safety,omitempty"`
	ValidTopics     []string                                                               `json:"valid_topics,omitempty"`
	Pii             *ResourceModelServingProvisionedThroughputAiGatewayGuardrailsOutputPii `json:"pii,omitempty"`
}

type ResourceModelServingProvisionedThroughputAiGatewayGuardrails struct {
	Input  *ResourceModelServingProvisionedThroughputAiGatewayGuardrailsInput  `json:"input,omitempty"`
	Output *ResourceModelServingProvisionedThroughputAiGatewayGuardrailsOutput `json:"output,omitempty"`
}

type ResourceModelServingProvisionedThroughputAiGatewayInferenceTableConfig struct {
	CatalogName     string `json:"catalog_name,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`
	SchemaName      string `json:"schema_name,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type ResourceModelServingProvisionedThroughputAiGatewayRateLimits struct {
	Calls         int    `json:"calls,omitempty"`
	Key           string `json:"key,omitempty"`
	Principal     string `json:"principal,omitempty"`
	RenewalPeriod string `json:"renewal_period"`
}

type ResourceModelServingProvisionedThroughputAiGatewayUsageTrackingConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ResourceModelServingProvisionedThroughputAiGateway struct {
	FallbackConfig       *ResourceModelServingProvisionedThroughputAiGatewayFallbackConfig       `json:"fallback_config,omitempty"`
	Guardrails           *ResourceModelServingProvisionedThroughputAiGatewayGuardrails           `json:"guardrails,omitempty"`
	InferenceTableConfig *ResourceModelServingProvisionedThroughputAiGatewayInferenceTableConfig `json:"inference_table_config,omitempty"`
	RateLimits           []ResourceModelServingProvisionedThroughputAiGatewayRateLimits          `json:"rate_limits,omitempty"`
	UsageTrackingConfig  *ResourceModelServingProvisionedThroughputAiGatewayUsageTrackingConfig  `json:"usage_tracking_config,omitempty"`
}

type ResourceModelServingProvisionedThroughputConfigServedEntities struct {
	EntityName            string `json:"entity_name"`
	EntityVersion         string `json:"entity_version"`
	Name                  string `json:"name,omitempty"`
	ProvisionedModelUnits int    `json:"provisioned_model_units"`
}

type ResourceModelServingProvisionedThroughputConfigTrafficConfigRoutes struct {
	ServedEntityName  string `json:"served_entity_name,omitempty"`
	ServedModelName   string `json:"served_model_name,omitempty"`
	TrafficPercentage int    `json:"traffic_percentage"`
}

type ResourceModelServingProvisionedThroughputConfigTrafficConfig struct {
	Routes []ResourceModelServingProvisionedThroughputConfigTrafficConfigRoutes `json:"routes,omitempty"`
}

type ResourceModelServingProvisionedThroughputConfig struct {
	ServedEntities []ResourceModelServingProvisionedThroughputConfigServedEntities `json:"served_entities,omitempty"`
	TrafficConfig  *ResourceModelServingProvisionedThroughputConfigTrafficConfig   `json:"traffic_config,omitempty"`
}

type ResourceModelServingProvisionedThroughputTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type ResourceModelServingProvisionedThroughput struct {
	BudgetPolicyId    string                                              `json:"budget_policy_id,omitempty"`
	Id                string                                              `json:"id,omitempty"`
	Name              string                                              `json:"name"`
	ServingEndpointId string                                              `json:"serving_endpoint_id,omitempty"`
	AiGateway         *ResourceModelServingProvisionedThroughputAiGateway `json:"ai_gateway,omitempty"`
	Config            *ResourceModelServingProvisionedThroughputConfig    `json:"config,omitempty"`
	Tags              []ResourceModelServingProvisionedThroughputTags     `json:"tags,omitempty"`
}
