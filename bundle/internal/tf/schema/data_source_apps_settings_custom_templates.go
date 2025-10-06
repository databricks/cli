// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsJobSpec struct {
	Permission string `json:"permission"`
}

type DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsSecretSpec struct {
	Permission string `json:"permission"`
}

type DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsServingEndpointSpec struct {
	Permission string `json:"permission"`
}

type DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsSqlWarehouseSpec struct {
	Permission string `json:"permission"`
}

type DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsUcSecurableSpec struct {
	Permission    string `json:"permission"`
	SecurableType string `json:"securable_type"`
}

type DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecs struct {
	Description         string                                                                                  `json:"description,omitempty"`
	JobSpec             *DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsJobSpec             `json:"job_spec,omitempty"`
	Name                string                                                                                  `json:"name"`
	SecretSpec          *DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsSecretSpec          `json:"secret_spec,omitempty"`
	ServingEndpointSpec *DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsServingEndpointSpec `json:"serving_endpoint_spec,omitempty"`
	SqlWarehouseSpec    *DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsSqlWarehouseSpec    `json:"sql_warehouse_spec,omitempty"`
	UcSecurableSpec     *DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecsUcSecurableSpec     `json:"uc_securable_spec,omitempty"`
}

type DataSourceAppsSettingsCustomTemplatesTemplatesManifest struct {
	Description   string                                                                `json:"description,omitempty"`
	Name          string                                                                `json:"name"`
	ResourceSpecs []DataSourceAppsSettingsCustomTemplatesTemplatesManifestResourceSpecs `json:"resource_specs,omitempty"`
	Version       int                                                                   `json:"version"`
}

type DataSourceAppsSettingsCustomTemplatesTemplates struct {
	Creator     string                                                  `json:"creator,omitempty"`
	Description string                                                  `json:"description,omitempty"`
	GitProvider string                                                  `json:"git_provider"`
	GitRepo     string                                                  `json:"git_repo"`
	Manifest    *DataSourceAppsSettingsCustomTemplatesTemplatesManifest `json:"manifest,omitempty"`
	Name        string                                                  `json:"name"`
	Path        string                                                  `json:"path"`
}

type DataSourceAppsSettingsCustomTemplates struct {
	Templates   []DataSourceAppsSettingsCustomTemplatesTemplates `json:"templates,omitempty"`
	WorkspaceId string                                           `json:"workspace_id,omitempty"`
}
