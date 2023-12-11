// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceStorageCredentialAwsIamRole struct {
	ExternalId         string `json:"external_id,omitempty"`
	RoleArn            string `json:"role_arn"`
	UnityCatalogIamArn string `json:"unity_catalog_iam_arn,omitempty"`
}

type ResourceStorageCredentialAzureManagedIdentity struct {
	AccessConnectorId string `json:"access_connector_id"`
	CredentialId      string `json:"credential_id,omitempty"`
	ManagedIdentityId string `json:"managed_identity_id,omitempty"`
}

type ResourceStorageCredentialAzureServicePrincipal struct {
	ApplicationId string `json:"application_id"`
	ClientSecret  string `json:"client_secret"`
	DirectoryId   string `json:"directory_id"`
}

type ResourceStorageCredentialDatabricksGcpServiceAccount struct {
	CredentialId string `json:"credential_id,omitempty"`
	Email        string `json:"email,omitempty"`
}

type ResourceStorageCredentialGcpServiceAccountKey struct {
	Email        string `json:"email"`
	PrivateKey   string `json:"private_key"`
	PrivateKeyId string `json:"private_key_id"`
}

type ResourceStorageCredential struct {
	Comment                     string                                                `json:"comment,omitempty"`
	ForceDestroy                bool                                                  `json:"force_destroy,omitempty"`
	Id                          string                                                `json:"id,omitempty"`
	MetastoreId                 string                                                `json:"metastore_id,omitempty"`
	Name                        string                                                `json:"name"`
	Owner                       string                                                `json:"owner,omitempty"`
	ReadOnly                    bool                                                  `json:"read_only,omitempty"`
	AwsIamRole                  *ResourceStorageCredentialAwsIamRole                  `json:"aws_iam_role,omitempty"`
	AzureManagedIdentity        *ResourceStorageCredentialAzureManagedIdentity        `json:"azure_managed_identity,omitempty"`
	AzureServicePrincipal       *ResourceStorageCredentialAzureServicePrincipal       `json:"azure_service_principal,omitempty"`
	DatabricksGcpServiceAccount *ResourceStorageCredentialDatabricksGcpServiceAccount `json:"databricks_gcp_service_account,omitempty"`
	GcpServiceAccountKey        *ResourceStorageCredentialGcpServiceAccountKey        `json:"gcp_service_account_key,omitempty"`
}
