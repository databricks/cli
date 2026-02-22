// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfig struct {
	UcServiceCredentialName string `json:"uc_service_credential_name,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsBackfillSourceDeltaTableSource struct {
	EntityColumns    []string `json:"entity_columns"`
	FullName         string   `json:"full_name"`
	TimeseriesColumn string   `json:"timeseries_column"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsBackfillSource struct {
	DeltaTableSource *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsBackfillSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsKeySchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsSubscriptionMode struct {
	Assign           string `json:"assign,omitempty"`
	Subscribe        string `json:"subscribe,omitempty"`
	SubscribePattern string `json:"subscribe_pattern,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsValueSchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigs struct {
	AuthConfig       *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfig       `json:"auth_config,omitempty"`
	BackfillSource   *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsBackfillSource   `json:"backfill_source,omitempty"`
	BootstrapServers string                                                                `json:"bootstrap_servers,omitempty"`
	ExtraOptions     map[string]string                                                     `json:"extra_options,omitempty"`
	KeySchema        *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsKeySchema        `json:"key_schema,omitempty"`
	Name             string                                                                `json:"name"`
	ProviderConfig   *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsProviderConfig   `json:"provider_config,omitempty"`
	SubscriptionMode *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsSubscriptionMode `json:"subscription_mode,omitempty"`
	ValueSchema      *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsValueSchema      `json:"value_schema,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceFeatureEngineeringKafkaConfigs struct {
	KafkaConfigs   []DataSourceFeatureEngineeringKafkaConfigsKafkaConfigs  `json:"kafka_configs,omitempty"`
	PageSize       int                                                     `json:"page_size,omitempty"`
	ProviderConfig *DataSourceFeatureEngineeringKafkaConfigsProviderConfig `json:"provider_config,omitempty"`
}
