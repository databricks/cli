// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceStorageCredentialStorageCredentialInfoAwsIamRole struct {
	ExternalId         string `json:"external_id,omitempty"`
	RoleArn            string `json:"role_arn"`
	UnityCatalogIamArn string `json:"unity_catalog_iam_arn,omitempty"`
}

type DataSourceStorageCredentialStorageCredentialInfoAzureManagedIdentity struct {
	AccessConnectorId string `json:"access_connector_id"`
	CredentialId      string `json:"credential_id,omitempty"`
	ManagedIdentityId string `json:"managed_identity_id,omitempty"`
}

type DataSourceStorageCredentialStorageCredentialInfoAzureServicePrincipal struct {
	ApplicationId string `json:"application_id"`
	ClientSecret  string `json:"client_secret"`
	DirectoryId   string `json:"directory_id"`
}

type DataSourceStorageCredentialStorageCredentialInfoCloudflareApiToken struct {
	AccessKeyId     string `json:"access_key_id"`
	AccountId       string `json:"account_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

type DataSourceStorageCredentialStorageCredentialInfoDatabricksGcpServiceAccount struct {
	CredentialId string `json:"credential_id,omitempty"`
	Email        string `json:"email,omitempty"`
}

type DataSourceStorageCredentialStorageCredentialInfo struct {
	Comment                     string                                                                       `json:"comment,omitempty"`
	CreatedAt                   int                                                                          `json:"created_at,omitempty"`
	CreatedBy                   string                                                                       `json:"created_by,omitempty"`
	FullName                    string                                                                       `json:"full_name,omitempty"`
	Id                          string                                                                       `json:"id,omitempty"`
	IsolationMode               string                                                                       `json:"isolation_mode,omitempty"`
	MetastoreId                 string                                                                       `json:"metastore_id,omitempty"`
	Name                        string                                                                       `json:"name,omitempty"`
	Owner                       string                                                                       `json:"owner,omitempty"`
	ReadOnly                    bool                                                                         `json:"read_only,omitempty"`
	UpdatedAt                   int                                                                          `json:"updated_at,omitempty"`
	UpdatedBy                   string                                                                       `json:"updated_by,omitempty"`
	UsedForManagedStorage       bool                                                                         `json:"used_for_managed_storage,omitempty"`
	AwsIamRole                  *DataSourceStorageCredentialStorageCredentialInfoAwsIamRole                  `json:"aws_iam_role,omitempty"`
	AzureManagedIdentity        *DataSourceStorageCredentialStorageCredentialInfoAzureManagedIdentity        `json:"azure_managed_identity,omitempty"`
	AzureServicePrincipal       *DataSourceStorageCredentialStorageCredentialInfoAzureServicePrincipal       `json:"azure_service_principal,omitempty"`
	CloudflareApiToken          *DataSourceStorageCredentialStorageCredentialInfoCloudflareApiToken          `json:"cloudflare_api_token,omitempty"`
	DatabricksGcpServiceAccount *DataSourceStorageCredentialStorageCredentialInfoDatabricksGcpServiceAccount `json:"databricks_gcp_service_account,omitempty"`
}

type DataSourceStorageCredential struct {
	Id                    string                                            `json:"id,omitempty"`
	Name                  string                                            `json:"name"`
	StorageCredentialInfo *DataSourceStorageCredentialStorageCredentialInfo `json:"storage_credential_info,omitempty"`
}
