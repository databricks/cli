// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfig struct {
	UcServiceCredentialName string `json:"uc_service_credential_name,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsKeySchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
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
	BootstrapServers string                                                                `json:"bootstrap_servers,omitempty"`
	ExtraOptions     map[string]string                                                     `json:"extra_options,omitempty"`
	KeySchema        *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsKeySchema        `json:"key_schema,omitempty"`
	Name             string                                                                `json:"name"`
	SubscriptionMode *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsSubscriptionMode `json:"subscription_mode,omitempty"`
	ValueSchema      *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsValueSchema      `json:"value_schema,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigs struct {
	KafkaConfigs []DataSourceFeatureEngineeringKafkaConfigsKafkaConfigs `json:"kafka_configs,omitempty"`
	PageSize     int                                                    `json:"page_size,omitempty"`
}
