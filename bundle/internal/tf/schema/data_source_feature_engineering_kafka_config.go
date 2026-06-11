// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigKeyPasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type DataSourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigKeystorePasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type DataSourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigTruststorePasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type DataSourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfig struct {
	DisableHostnameVerification bool                                                                              `json:"disable_hostname_verification,omitempty"`
	KeyPasswordRef              *DataSourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigKeyPasswordRef        `json:"key_password_ref,omitempty"`
	KeystoreLocation            string                                                                            `json:"keystore_location"`
	KeystorePasswordRef         *DataSourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigKeystorePasswordRef   `json:"keystore_password_ref,omitempty"`
	TruststoreLocation          string                                                                            `json:"truststore_location"`
	TruststorePasswordRef       *DataSourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigTruststorePasswordRef `json:"truststore_password_ref,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigAuthConfig struct {
	MtlsConfig              *DataSourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfig `json:"mtls_config,omitempty"`
	UcServiceCredentialName string                                                       `json:"uc_service_credential_name,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigBackfillSourceDeltaTableSource struct {
	DataframeSchema   string   `json:"dataframe_schema,omitempty"`
	EntityColumns     []string `json:"entity_columns,omitempty"`
	FilterCondition   string   `json:"filter_condition,omitempty"`
	FullName          string   `json:"full_name"`
	TimeseriesColumn  string   `json:"timeseries_column,omitempty"`
	TransformationSql string   `json:"transformation_sql,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigBackfillSource struct {
	DeltaTableName   string                                                                 `json:"delta_table_name,omitempty"`
	DeltaTableSource *DataSourceFeatureEngineeringKafkaConfigBackfillSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigKeySchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type DataSourceFeatureEngineeringKafkaConfigProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
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
	BackfillSource   *DataSourceFeatureEngineeringKafkaConfigBackfillSource   `json:"backfill_source,omitempty"`
	BootstrapServers string                                                   `json:"bootstrap_servers,omitempty"`
	ExtraOptions     map[string]string                                        `json:"extra_options,omitempty"`
	KeySchema        *DataSourceFeatureEngineeringKafkaConfigKeySchema        `json:"key_schema,omitempty"`
	Name             string                                                   `json:"name"`
	ProviderConfig   *DataSourceFeatureEngineeringKafkaConfigProviderConfig   `json:"provider_config,omitempty"`
	SubscriptionMode *DataSourceFeatureEngineeringKafkaConfigSubscriptionMode `json:"subscription_mode,omitempty"`
	ValueSchema      *DataSourceFeatureEngineeringKafkaConfigValueSchema      `json:"value_schema,omitempty"`
}
