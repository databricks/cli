// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringKafkaConfigAuthConfig struct {
	UcServiceCredentialName string `json:"uc_service_credential_name,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigKeySchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigSubscriptionMode struct {
	Assign           string `json:"assign,omitempty"`
	Subscribe        string `json:"subscribe,omitempty"`
	SubscribePattern string `json:"subscribe_pattern,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigValueSchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfig struct {
	AuthConfig       *DataSourceFeatureEngineeringKafkaConfigAuthConfig       `json:"auth_config,omitempty"`
	BootstrapServers string                                                   `json:"bootstrap_servers,omitempty"`
	ExtraOptions     map[string]string                                        `json:"extra_options,omitempty"`
	KeySchema        *DataSourceFeatureEngineeringKafkaConfigKeySchema        `json:"key_schema,omitempty"`
	Name             string                                                   `json:"name"`
	SubscriptionMode *DataSourceFeatureEngineeringKafkaConfigSubscriptionMode `json:"subscription_mode,omitempty"`
	ValueSchema      *DataSourceFeatureEngineeringKafkaConfigValueSchema      `json:"value_schema,omitempty"`
}
