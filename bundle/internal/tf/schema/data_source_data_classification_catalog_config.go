// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDataClassificationCatalogConfigAutoTagConfigs struct {
	AutoTaggingMode   string `json:"auto_tagging_mode"`
	ClassificationTag string `json:"classification_tag"`
}

type DataSourceDataClassificationCatalogConfigIncludedSchemas struct {
	Names []string `json:"names"`
}

type DataSourceDataClassificationCatalogConfigProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceDataClassificationCatalogConfig struct {
	AutoTagConfigs  []DataSourceDataClassificationCatalogConfigAutoTagConfigs `json:"auto_tag_configs,omitempty"`
	IncludedSchemas *DataSourceDataClassificationCatalogConfigIncludedSchemas `json:"included_schemas,omitempty"`
	Name            string                                                    `json:"name"`
	ProviderConfig  *DataSourceDataClassificationCatalogConfigProviderConfig  `json:"provider_config,omitempty"`
}
