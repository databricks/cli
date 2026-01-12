// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAppsSettingsCustomTemplateManifestResourceSpecsExperimentSpec struct {
	Permission string `json:"permission"`
}

type ResourceAppsSettingsCustomTemplateManifestResourceSpecsJobSpec struct {
	Permission string `json:"permission"`
}

type ResourceAppsSettingsCustomTemplateManifestResourceSpecsSecretSpec struct {
	Permission string `json:"permission"`
}

type ResourceAppsSettingsCustomTemplateManifestResourceSpecsServingEndpointSpec struct {
	Permission string `json:"permission"`
}

type ResourceAppsSettingsCustomTemplateManifestResourceSpecsSqlWarehouseSpec struct {
	Permission string `json:"permission"`
}

type ResourceAppsSettingsCustomTemplateManifestResourceSpecsUcSecurableSpec struct {
	Permission    string `json:"permission"`
	SecurableType string `json:"securable_type"`
}

type ResourceAppsSettingsCustomTemplateManifestResourceSpecs struct {
	Description         string                                                                      `json:"description,omitempty"`
	ExperimentSpec      *ResourceAppsSettingsCustomTemplateManifestResourceSpecsExperimentSpec      `json:"experiment_spec,omitempty"`
	JobSpec             *ResourceAppsSettingsCustomTemplateManifestResourceSpecsJobSpec             `json:"job_spec,omitempty"`
	Name                string                                                                      `json:"name"`
	SecretSpec          *ResourceAppsSettingsCustomTemplateManifestResourceSpecsSecretSpec          `json:"secret_spec,omitempty"`
	ServingEndpointSpec *ResourceAppsSettingsCustomTemplateManifestResourceSpecsServingEndpointSpec `json:"serving_endpoint_spec,omitempty"`
	SqlWarehouseSpec    *ResourceAppsSettingsCustomTemplateManifestResourceSpecsSqlWarehouseSpec    `json:"sql_warehouse_spec,omitempty"`
	UcSecurableSpec     *ResourceAppsSettingsCustomTemplateManifestResourceSpecsUcSecurableSpec     `json:"uc_securable_spec,omitempty"`
}

type ResourceAppsSettingsCustomTemplateManifest struct {
	Description   string                                                    `json:"description,omitempty"`
	Name          string                                                    `json:"name"`
	ResourceSpecs []ResourceAppsSettingsCustomTemplateManifestResourceSpecs `json:"resource_specs,omitempty"`
	Version       int                                                       `json:"version"`
}

type ResourceAppsSettingsCustomTemplate struct {
	Creator     string                                      `json:"creator,omitempty"`
	Description string                                      `json:"description,omitempty"`
	GitProvider string                                      `json:"git_provider"`
	GitRepo     string                                      `json:"git_repo"`
	Manifest    *ResourceAppsSettingsCustomTemplateManifest `json:"manifest,omitempty"`
	Name        string                                      `json:"name"`
	Path        string                                      `json:"path"`
}
