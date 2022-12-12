// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMlflowWebhookHttpUrlSpec struct {
	Authorization         string `json:"authorization,omitempty"`
	EnableSslVerification bool   `json:"enable_ssl_verification,omitempty"`
	Secret                string `json:"secret,omitempty"`
	Url                   string `json:"url"`
}

type ResourceMlflowWebhookJobSpec struct {
	AccessToken  string `json:"access_token"`
	JobId        string `json:"job_id"`
	WorkspaceUrl string `json:"workspace_url,omitempty"`
}

type ResourceMlflowWebhook struct {
	Description string                            `json:"description,omitempty"`
	Events      []string                          `json:"events"`
	Id          string                            `json:"id,omitempty"`
	ModelName   string                            `json:"model_name,omitempty"`
	Status      string                            `json:"status,omitempty"`
	HttpUrlSpec *ResourceMlflowWebhookHttpUrlSpec `json:"http_url_spec,omitempty"`
	JobSpec     *ResourceMlflowWebhookJobSpec     `json:"job_spec,omitempty"`
}
