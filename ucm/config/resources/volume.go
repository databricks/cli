package resources

import "net/url"

// Volume is a UC volume (managed or external). Field names mirror
// databricks-sdk-go's catalog.CreateVolumeRequestContent so the direct-
// engine input builder stays a 1:1 copy.
//
// VolumeType is "MANAGED" (UC provisions the underlying storage) or
// "EXTERNAL" (points at a cloud path under an external_location).
// StorageLocation is required for EXTERNAL and unset for MANAGED.
type Volume struct {
	Name            string `json:"name"`
	CatalogName     string `json:"catalog_name"`
	SchemaName      string `json:"schema_name"`
	VolumeType      string `json:"volume_type"`
	StorageLocation string `json:"storage_location,omitempty"`
	Comment         string `json:"comment,omitempty"`

	// ID is the deployed resource's terraform-state ID. Populated by
	// statemgmt.Load from the local tfstate; never written from ucm.yml.
	ID string `json:"id,omitempty" ucm:"readonly"`

	// URL is populated by the initialize_urls mutator.
	URL string `json:"url,omitempty" ucm:"readonly"`
}

// InitializeURL sets v.URL iff the volume has been deployed (ID is non-empty).
func (v *Volume) InitializeURL(baseURL url.URL) {
	if v.ID == "" {
		return
	}
	baseURL.Path = "explore/data/" + v.CatalogName + "/" + v.SchemaName + "/" + v.Name
	v.URL = baseURL.String()
}
