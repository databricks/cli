// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppsSettingsCustomTemplateManifestResourceSpecsJobSpec struct {
	Permission string `json:"permission"`
}

type DataSourceAppsSettingsCustomTemplateManifestResourceSpecsSecretSpec struct {
	Permission string `json:"permission"`
}

type DataSourceAppsSettingsCustomTemplateManifestResourceSpecsServingEndpointSpec struct {
	Permission string `json:"permission"`
}

type DataSourceAppsSettingsCustomTemplateManifestResourceSpecsSqlWarehouseSpec struct {
	Permission string `json:"permission"`
}

type DataSourceAppsSettingsCustomTemplateManifestResourceSpecsUcSecurableSpec struct {
	Permission    string `json:"permission"`
	SecurableType string `json:"securable_type"`
}

type DataSourceAppsSettingsCustomTemplateManifestResourceSpecs struct {
	Description         string                                                                        `json:"description,omitempty"`
	JobSpec             *DataSourceAppsSettingsCustomTemplateManifestResourceSpecsJobSpec             `json:"job_spec,omitempty"`
	Name                string                                                                        `json:"name"`
	SecretSpec          *DataSourceAppsSettingsCustomTemplateManifestResourceSpecsSecretSpec          `json:"secret_spec,omitempty"`
	ServingEndpointSpec *DataSourceAppsSettingsCustomTemplateManifestResourceSpecsServingEndpointSpec `json:"serving_endpoint_spec,omitempty"`
	SqlWarehouseSpec    *DataSourceAppsSettingsCustomTemplateManifestResourceSpecsSqlWarehouseSpec    `json:"sql_warehouse_spec,omitempty"`
	UcSecurableSpec     *DataSourceAppsSettingsCustomTemplateManifestResourceSpecsUcSecurableSpec     `json:"uc_securable_spec,omitempty"`
}

type DataSourceAppsSettingsCustomTemplateManifest struct {
	Description   string                                                      `json:"description,omitempty"`
	Name          string                                                      `json:"name"`
	ResourceSpecs []DataSourceAppsSettingsCustomTemplateManifestResourceSpecs `json:"resource_specs,omitempty"`
	Version       int                                                         `json:"version"`
}

type DataSourceAppsSettingsCustomTemplate struct {
	Creator     string                                        `json:"creator,omitempty"`
	Description string                                        `json:"description,omitempty"`
	GitProvider string                                        `json:"git_provider"`
	GitRepo     string                                        `json:"git_repo"`
	Manifest    *DataSourceAppsSettingsCustomTemplateManifest `json:"manifest,omitempty"`
	Name        string                                        `json:"name"`
	Path        string                                        `json:"path"`
	WorkspaceId string                                        `json:"workspace_id,omitempty"`
}
