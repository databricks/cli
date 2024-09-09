package resources

import (
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Volume struct {
	// List of grants to apply on this schema.
	Grants []Grant `json:"grants,omitempty"`

	// TODO: Confirm the accuracy of this comment.
	// Full name of the schema (catalog_name.schema_name.volume_name). This value is read from
	// the terraform state after deployment succeeds.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// TODO: Are there fields in the edit API or terraform that are not in this struct?
	// If so call it out in the PR.
	*catalog.CreateVolumeRequestContent

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
}

func (v *Volume) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, v)
}

func (v Volume) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(v)
}
