// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type Config struct {
	AccountId                string `json:"account_id,omitempty"`
	AuthType                 string `json:"auth_type,omitempty"`
	AzureClientId            string `json:"azure_client_id,omitempty"`
	AzureClientSecret        string `json:"azure_client_secret,omitempty"`
	AzureEnvironment         string `json:"azure_environment,omitempty"`
	AzureLoginAppId          string `json:"azure_login_app_id,omitempty"`
	AzureTenantId            string `json:"azure_tenant_id,omitempty"`
	AzureUseMsi              bool   `json:"azure_use_msi,omitempty"`
	AzureWorkspaceResourceId string `json:"azure_workspace_resource_id,omitempty"`
	BricksCliPath            string `json:"bricks_cli_path,omitempty"`
	ClientId                 string `json:"client_id,omitempty"`
	ClientSecret             string `json:"client_secret,omitempty"`
	ConfigFile               string `json:"config_file,omitempty"`
	DebugHeaders             bool   `json:"debug_headers,omitempty"`
	DebugTruncateBytes       int    `json:"debug_truncate_bytes,omitempty"`
	GoogleCredentials        string `json:"google_credentials,omitempty"`
	GoogleServiceAccount     string `json:"google_service_account,omitempty"`
	Host                     string `json:"host,omitempty"`
	HttpTimeoutSeconds       int    `json:"http_timeout_seconds,omitempty"`
	Password                 string `json:"password,omitempty"`
	Profile                  string `json:"profile,omitempty"`
	RateLimit                int    `json:"rate_limit,omitempty"`
	RetryTimeoutSeconds      int    `json:"retry_timeout_seconds,omitempty"`
	SkipVerify               bool   `json:"skip_verify,omitempty"`
	Token                    string `json:"token,omitempty"`
	Username                 string `json:"username,omitempty"`
}
