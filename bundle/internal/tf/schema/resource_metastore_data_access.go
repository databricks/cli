// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceMetastoreDataAccessAwsIamRole struct {
	RoleArn string `json:"role_arn"`
}

type ResourceMetastoreDataAccessAzureManagedIdentity struct {
	AccessConnectorId string `json:"access_connector_id"`
}

type ResourceMetastoreDataAccessAzureServicePrincipal struct {
	ApplicationId string `json:"application_id"`
	ClientSecret  string `json:"client_secret"`
	DirectoryId   string `json:"directory_id"`
}

type ResourceMetastoreDataAccess struct {
	ConfigurationType     string                                            `json:"configuration_type,omitempty"`
	Id                    string                                            `json:"id,omitempty"`
	IsDefault             bool                                              `json:"is_default,omitempty"`
	MetastoreId           string                                            `json:"metastore_id"`
	Name                  string                                            `json:"name"`
	AwsIamRole            *ResourceMetastoreDataAccessAwsIamRole            `json:"aws_iam_role,omitempty"`
	AzureManagedIdentity  *ResourceMetastoreDataAccessAzureManagedIdentity  `json:"azure_managed_identity,omitempty"`
	AzureServicePrincipal *ResourceMetastoreDataAccessAzureServicePrincipal `json:"azure_service_principal,omitempty"`
}
