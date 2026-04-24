package resources

import "net/url"

// StorageCredential is a UC storage credential. Exactly one of the cloud
// identity fields (AwsIamRole, AzureManagedIdentity, AzureServicePrincipal,
// DatabricksGcpServiceAccount) must be set. Field shape mirrors
// databricks-sdk-go's catalog.CreateStorageCredential so the direct-engine
// input builder is a 1:1 copy rather than a mapping layer.
type StorageCredential struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`

	AwsIamRole                  *AwsIamRole                  `json:"aws_iam_role,omitempty"`
	AzureManagedIdentity        *AzureManagedIdentity        `json:"azure_managed_identity,omitempty"`
	AzureServicePrincipal       *AzureServicePrincipal       `json:"azure_service_principal,omitempty"`
	DatabricksGcpServiceAccount *DatabricksGcpServiceAccount `json:"databricks_gcp_service_account,omitempty"`

	ReadOnly       bool `json:"read_only,omitempty"`
	SkipValidation bool `json:"skip_validation,omitempty"`

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

// InitializeURL sets s.URL iff the storage credential has been deployed
// (ID is non-empty).
func (s *StorageCredential) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	baseURL.Path = "explore/storage-credentials/" + s.Name
	s.URL = baseURL.String()
}

// AwsIamRole is the AWS IAM role UC assumes to vend temporary credentials.
type AwsIamRole struct {
	RoleArn string `json:"role_arn"`
}

// AzureManagedIdentity identifies an Azure managed identity by access connector.
type AzureManagedIdentity struct {
	AccessConnectorId string `json:"access_connector_id"`
	ManagedIdentityId string `json:"managed_identity_id,omitempty"`
}

// AzureServicePrincipal holds an Azure AD service principal reference.
type AzureServicePrincipal struct {
	DirectoryId   string `json:"directory_id"`
	ApplicationId string `json:"application_id"`
	ClientSecret  string `json:"client_secret"`
}

// DatabricksGcpServiceAccount toggles the Databricks-managed GCP identity
// shape. Presence alone is meaningful; there are no user-supplied fields.
type DatabricksGcpServiceAccount struct{}
