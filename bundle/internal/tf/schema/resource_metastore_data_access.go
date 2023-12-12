// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMetastoreDataAccessAwsIamRole struct {
	ExternalId         string `json:"external_id,omitempty"`
	RoleArn            string `json:"role_arn"`
	UnityCatalogIamArn string `json:"unity_catalog_iam_arn,omitempty"`
}

type ResourceMetastoreDataAccessAzureManagedIdentity struct {
	AccessConnectorId string `json:"access_connector_id"`
	CredentialId      string `json:"credential_id,omitempty"`
	ManagedIdentityId string `json:"managed_identity_id,omitempty"`
}

type ResourceMetastoreDataAccessAzureServicePrincipal struct {
	ApplicationId string `json:"application_id"`
	ClientSecret  string `json:"client_secret"`
	DirectoryId   string `json:"directory_id"`
}

type ResourceMetastoreDataAccessDatabricksGcpServiceAccount struct {
	CredentialId string `json:"credential_id,omitempty"`
	Email        string `json:"email,omitempty"`
}

type ResourceMetastoreDataAccessGcpServiceAccountKey struct {
	Email        string `json:"email"`
	PrivateKey   string `json:"private_key"`
	PrivateKeyId string `json:"private_key_id"`
}

type ResourceMetastoreDataAccess struct {
	Comment                     string                                                  `json:"comment,omitempty"`
	ForceDestroy                bool                                                    `json:"force_destroy,omitempty"`
	Id                          string                                                  `json:"id,omitempty"`
	IsDefault                   bool                                                    `json:"is_default,omitempty"`
	MetastoreId                 string                                                  `json:"metastore_id,omitempty"`
	Name                        string                                                  `json:"name"`
	Owner                       string                                                  `json:"owner,omitempty"`
	ReadOnly                    bool                                                    `json:"read_only,omitempty"`
	AwsIamRole                  *ResourceMetastoreDataAccessAwsIamRole                  `json:"aws_iam_role,omitempty"`
	AzureManagedIdentity        *ResourceMetastoreDataAccessAzureManagedIdentity        `json:"azure_managed_identity,omitempty"`
	AzureServicePrincipal       *ResourceMetastoreDataAccessAzureServicePrincipal       `json:"azure_service_principal,omitempty"`
	DatabricksGcpServiceAccount *ResourceMetastoreDataAccessDatabricksGcpServiceAccount `json:"databricks_gcp_service_account,omitempty"`
	GcpServiceAccountKey        *ResourceMetastoreDataAccessGcpServiceAccountKey        `json:"gcp_service_account_key,omitempty"`
}
