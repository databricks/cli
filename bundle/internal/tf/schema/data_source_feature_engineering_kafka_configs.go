// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfigMtlsConfigKeyPasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfigMtlsConfigKeystorePasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfigMtlsConfigTruststorePasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfigMtlsConfig struct {
	DisableHostnameVerification bool                                                                                           `json:"disable_hostname_verification,omitempty"`
	KeyPasswordRef              *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfigMtlsConfigKeyPasswordRef        `json:"key_password_ref,omitempty"`
	KeystoreLocation            string                                                                                         `json:"keystore_location"`
	KeystorePasswordRef         *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfigMtlsConfigKeystorePasswordRef   `json:"keystore_password_ref,omitempty"`
	TruststoreLocation          string                                                                                         `json:"truststore_location"`
	TruststorePasswordRef       *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfigMtlsConfigTruststorePasswordRef `json:"truststore_password_ref,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfig struct {
	MtlsConfig              *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsAuthConfigMtlsConfig `json:"mtls_config,omitempty"`
	UcServiceCredentialName string                                                                    `json:"uc_service_credential_name,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsBackfillSourceDeltaTableSource struct {
	DataframeSchema   string   `json:"dataframe_schema,omitempty"`
	EntityColumns     []string `json:"entity_columns,omitempty"`
	FilterCondition   string   `json:"filter_condition,omitempty"`
	FullName          string   `json:"full_name"`
	TimeseriesColumn  string   `json:"timeseries_column,omitempty"`
	TransformationSql string   `json:"transformation_sql,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsBackfillSource struct {
	DeltaTableName   string                                                                              `json:"delta_table_name,omitempty"`
	DeltaTableSource *DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsBackfillSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsKeySchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigsKafkaConfigsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
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
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigs struct {
	KafkaConfigs   []DataSourceFeatureEngineeringKafkaConfigsKafkaConfigs  `json:"kafka_configs,omitempty"`
	PageSize       int                                                     `json:"page_size,omitempty"`
	ProviderConfig *DataSourceFeatureEngineeringKafkaConfigsProviderConfig `json:"provider_config,omitempty"`
}
