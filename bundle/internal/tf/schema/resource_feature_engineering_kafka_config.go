// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceFeatureEngineeringKafkaConfigAuthConfig struct {
	UcServiceCredentialName string `json:"uc_service_credential_name,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigBackfillSourceDeltaTableSource struct {
	EntityColumns    []string `json:"entity_columns"`
	FullName         string   `json:"full_name"`
	TimeseriesColumn string   `json:"timeseries_column"`
}

type ResourceFeatureEngineeringKafkaConfigBackfillSource struct {
	DeltaTableSource *ResourceFeatureEngineeringKafkaConfigBackfillSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigKeySchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigSubscriptionMode struct {
	Assign           string `json:"assign,omitempty"`
	Subscribe        string `json:"subscribe,omitempty"`
	SubscribePattern string `json:"subscribe_pattern,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigValueSchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfig struct {
	AuthConfig       *ResourceFeatureEngineeringKafkaConfigAuthConfig       `json:"auth_config,omitempty"`
	BackfillSource   *ResourceFeatureEngineeringKafkaConfigBackfillSource   `json:"backfill_source,omitempty"`
	BootstrapServers string                                                 `json:"bootstrap_servers"`
	ExtraOptions     map[string]string                                      `json:"extra_options,omitempty"`
	KeySchema        *ResourceFeatureEngineeringKafkaConfigKeySchema        `json:"key_schema,omitempty"`
	Name             string                                                 `json:"name,omitempty"`
	SubscriptionMode *ResourceFeatureEngineeringKafkaConfigSubscriptionMode `json:"subscription_mode,omitempty"`
	ValueSchema      *ResourceFeatureEngineeringKafkaConfigValueSchema      `json:"value_schema,omitempty"`
}
