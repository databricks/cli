package resources

import (
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Volume struct {
	// List of grants to apply on this volume.
	Grants []Grant `json:"grants,omitempty"`

	// Full name of the volume (catalog_name.schema_name.volume_name). This value is read from
	// the terraform state after deployment succeeds.
	ID string `json:"id,omitempty" bundle:"readonly"`

	*catalog.CreateVolumeRequestContent

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
}

func (v *Volume) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, v)
}

func (v Volume) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(v)
}
