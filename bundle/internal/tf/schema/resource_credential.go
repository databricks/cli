// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceCredentialAwsIamRole struct {
	ExternalId         string `json:"external_id,omitempty"`
	RoleArn            string `json:"role_arn,omitempty"`
	UnityCatalogIamArn string `json:"unity_catalog_iam_arn,omitempty"`
}

type ResourceCredentialAzureManagedIdentity struct {
	AccessConnectorId string `json:"access_connector_id"`
	CredentialId      string `json:"credential_id,omitempty"`
	ManagedIdentityId string `json:"managed_identity_id,omitempty"`
}

type ResourceCredentialAzureServicePrincipal struct {
	ApplicationId string `json:"application_id"`
	ClientSecret  string `json:"client_secret"`
	DirectoryId   string `json:"directory_id"`
}

type ResourceCredentialDatabricksGcpServiceAccount struct {
	CredentialId string `json:"credential_id,omitempty"`
	Email        string `json:"email,omitempty"`
	PrivateKeyId string `json:"private_key_id,omitempty"`
}

type ResourceCredential struct {
	Comment                     string                                         `json:"comment,omitempty"`
	CreatedAt                   int                                            `json:"created_at,omitempty"`
	CreatedBy                   string                                         `json:"created_by,omitempty"`
	CredentialId                string                                         `json:"credential_id,omitempty"`
	ForceDestroy                bool                                           `json:"force_destroy,omitempty"`
	ForceUpdate                 bool                                           `json:"force_update,omitempty"`
	FullName                    string                                         `json:"full_name,omitempty"`
	Id                          string                                         `json:"id,omitempty"`
	IsolationMode               string                                         `json:"isolation_mode,omitempty"`
	MetastoreId                 string                                         `json:"metastore_id,omitempty"`
	Name                        string                                         `json:"name"`
	Owner                       string                                         `json:"owner,omitempty"`
	Purpose                     string                                         `json:"purpose"`
	ReadOnly                    bool                                           `json:"read_only,omitempty"`
	SkipValidation              bool                                           `json:"skip_validation,omitempty"`
	UpdatedAt                   int                                            `json:"updated_at,omitempty"`
	UpdatedBy                   string                                         `json:"updated_by,omitempty"`
	UsedForManagedStorage       bool                                           `json:"used_for_managed_storage,omitempty"`
	AwsIamRole                  *ResourceCredentialAwsIamRole                  `json:"aws_iam_role,omitempty"`
	AzureManagedIdentity        *ResourceCredentialAzureManagedIdentity        `json:"azure_managed_identity,omitempty"`
	AzureServicePrincipal       *ResourceCredentialAzureServicePrincipal       `json:"azure_service_principal,omitempty"`
	DatabricksGcpServiceAccount *ResourceCredentialDatabricksGcpServiceAccount `json:"databricks_gcp_service_account,omitempty"`
}
