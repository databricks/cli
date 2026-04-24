package resources

import "net/url"

// ExternalLocation is a UC external location. URL + storage credential
// together grant UC access to a specific cloud-storage prefix. Field names
// mirror databricks-sdk-go's catalog.CreateExternalLocation so the direct-
// engine input builder stays a 1:1 copy rather than a mapping layer.
//
// M0 scope: name, url, credential_name, comment, read_only, skip_validation,
// fallback. Encryption details and file-event queue support land later.
//
// Url (lowercase) is the cloud storage path (s3://..., abfss://..., gs://...);
// URL (uppercase) is the workspace console URL populated by the
// initialize_urls mutator. The two are distinct.
type ExternalLocation struct {
	Name           string `json:"name"`
	Url            string `json:"url"`
	CredentialName string `json:"credential_name"`

	Comment        string `json:"comment,omitempty"`
	ReadOnly       bool   `json:"read_only,omitempty"`
	SkipValidation bool   `json:"skip_validation,omitempty"`
	Fallback       bool   `json:"fallback,omitempty"`

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"workspace_url,omitempty" ucm:"readonly"`
}

// InitializeURL sets e.URL iff the external location has been deployed
// (ID is non-empty).
func (e *ExternalLocation) InitializeURL(baseURL url.URL) {
	if e.ID == "" {
		return
	}
	baseURL.Path = "explore/external-locations/" + e.Name
	e.URL = baseURL.String()
}
