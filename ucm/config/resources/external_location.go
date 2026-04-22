package resources

// ExternalLocation is a UC external location. URL + storage credential
// together grant UC access to a specific cloud-storage prefix. Field names
// mirror databricks-sdk-go's catalog.CreateExternalLocation so the direct-
// engine input builder stays a 1:1 copy rather than a mapping layer.
//
// M0 scope: name, url, credential_name, comment, read_only, skip_validation,
// fallback. Encryption details and file-event queue support land later.
type ExternalLocation struct {
	Name           string `json:"name"`
	Url            string `json:"url"`
	CredentialName string `json:"credential_name"`

	Comment        string `json:"comment,omitempty"`
	ReadOnly       bool   `json:"read_only,omitempty"`
	SkipValidation bool   `json:"skip_validation,omitempty"`
	Fallback       bool   `json:"fallback,omitempty"`
}
