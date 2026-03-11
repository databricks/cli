// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema


type ResourceDataClassificationCatalogConfigAutoTagConfigs struct {
    AutoTaggingMode string `json:"auto_tagging_mode"`
    ClassificationTag string `json:"classification_tag"`
}

type ResourceDataClassificationCatalogConfigIncludedSchemas struct {
    Names []string `json:"names"`
}

type ResourceDataClassificationCatalogConfigProviderConfig struct {
    WorkspaceId string `json:"workspace_id"`
}

type ResourceDataClassificationCatalogConfig struct {
    AutoTagConfigs []ResourceDataClassificationCatalogConfigAutoTagConfigs `json:"auto_tag_configs,omitempty"`
    IncludedSchemas *ResourceDataClassificationCatalogConfigIncludedSchemas `json:"included_schemas,omitempty"`
    Name string `json:"name,omitempty"`
    Parent string `json:"parent"`
    ProviderConfig *ResourceDataClassificationCatalogConfigProviderConfig `json:"provider_config,omitempty"`
}
