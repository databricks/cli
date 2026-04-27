package resources

import (
	"net/url"

	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// StorageCredential is a UC storage credential. Embeds the SDK's
// CreateStorageCredential for full attribute coverage; one of the SDK
// identity fields (AwsIamRole, AzureManagedIdentity, AzureServicePrincipal,
// DatabricksGcpServiceAccount, CloudflareApiToken) must be set.
type StorageCredential struct {
	catalog.CreateStorageCredential

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

func (s *StorageCredential) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s StorageCredential) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}
