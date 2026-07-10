// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigKeyPasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type ResourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigKeystorePasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type ResourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigTruststorePasswordRef struct {
	Key   string `json:"key"`
	Scope string `json:"scope"`
}

type ResourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfig struct {
	DisableHostnameVerification bool                                                                            `json:"disable_hostname_verification,omitempty"`
	KeyPasswordRef              *ResourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigKeyPasswordRef        `json:"key_password_ref,omitempty"`
	KeystoreLocation            string                                                                          `json:"keystore_location"`
	KeystorePasswordRef         *ResourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigKeystorePasswordRef   `json:"keystore_password_ref,omitempty"`
	TruststoreLocation          string                                                                          `json:"truststore_location"`
	TruststorePasswordRef       *ResourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfigTruststorePasswordRef `json:"truststore_password_ref,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigAuthConfig struct {
	MtlsConfig              *ResourceFeatureEngineeringKafkaConfigAuthConfigMtlsConfig `json:"mtls_config,omitempty"`
	UcServiceCredentialName string                                                     `json:"uc_service_credential_name,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigBackfillSourceDeltaTableSource struct {
	DataframeSchema   string   `json:"dataframe_schema,omitempty"`
	EntityColumns     []string `json:"entity_columns,omitempty"`
	FilterCondition   string   `json:"filter_condition,omitempty"`
	FullName          string   `json:"full_name"`
	TimeseriesColumn  string   `json:"timeseries_column,omitempty"`
	TransformationSql string   `json:"transformation_sql,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigBackfillSource struct {
	DeltaTableName   string                                                               `json:"delta_table_name,omitempty"`
	DeltaTableSource *ResourceFeatureEngineeringKafkaConfigBackfillSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigIngestionConfigBackfillSourceDeltaTableSource struct {
	DataframeSchema   string   `json:"dataframe_schema,omitempty"`
	EntityColumns     []string `json:"entity_columns,omitempty"`
	FilterCondition   string   `json:"filter_condition,omitempty"`
	FullName          string   `json:"full_name"`
	TimeseriesColumn  string   `json:"timeseries_column,omitempty"`
	TransformationSql string   `json:"transformation_sql,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigIngestionConfigBackfillSource struct {
	DeltaTableName   string                                                                              `json:"delta_table_name,omitempty"`
	DeltaTableSource *ResourceFeatureEngineeringKafkaConfigIngestionConfigBackfillSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigIngestionConfigIngestionDestination struct {
	DeltaTableName string `json:"delta_table_name,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigIngestionConfig struct {
	BackfillJobId        int                                                                       `json:"backfill_job_id,omitempty"`
	BackfillSource       *ResourceFeatureEngineeringKafkaConfigIngestionConfigBackfillSource       `json:"backfill_source,omitempty"`
	DeduplicationColumns []string                                                                  `json:"deduplication_columns,omitempty"`
	IngestionDestination *ResourceFeatureEngineeringKafkaConfigIngestionConfigIngestionDestination `json:"ingestion_destination,omitempty"`
	IngestionJobId       int                                                                       `json:"ingestion_job_id,omitempty"`
	IngestionPipelineId  string                                                                    `json:"ingestion_pipeline_id,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigKeySchema struct {
	JsonSchema string `json:"json_schema,omitempty"`
}

type ResourceFeatureEngineeringKafkaConfigProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
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
	IngestionConfig  *ResourceFeatureEngineeringKafkaConfigIngestionConfig  `json:"ingestion_config,omitempty"`
	KeySchema        *ResourceFeatureEngineeringKafkaConfigKeySchema        `json:"key_schema,omitempty"`
	Name             string                                                 `json:"name,omitempty"`
	ProviderConfig   *ResourceFeatureEngineeringKafkaConfigProviderConfig   `json:"provider_config,omitempty"`
	SubscriptionMode *ResourceFeatureEngineeringKafkaConfigSubscriptionMode `json:"subscription_mode,omitempty"`
	ValueSchema      *ResourceFeatureEngineeringKafkaConfigValueSchema      `json:"value_schema,omitempty"`
}
