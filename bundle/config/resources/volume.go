package resources

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Volume struct {
	BaseResource
	catalog.CreateVolumeRequestContent

	// VolumePath is /Volumes/{catalog}/{schema}/{name}. Populated during initialize; not user-configurable.
	VolumePath string `json:"volume_path,omitempty" bundle:"readonly"`

	// List of grants to apply on this volume.
	Grants []catalog.PrivilegeAssignment `json:"grants,omitempty"`
}

func (v *Volume) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, v)
}

func (v Volume) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(v)
}

func (v *Volume) Exists(ctx context.Context, w *databricks.WorkspaceClient, fullyQualifiedName string) (bool, error) {
	_, err := w.Volumes.Read(ctx, catalog.ReadVolumeRequest{
		Name: fullyQualifiedName,
	})
	if err != nil {
		log.Debugf(ctx, "volume with fully qualified name %s does not exist: %v", fullyQualifiedName, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (*Volume) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "volume",
		PluralName:    "volumes",
		SingularTitle: "Volume",
		PluralTitle:   "Volumes",
	}
}

func (v *Volume) InitializeURL(baseURL url.URL) {
	if v.ID == "" {
		return
	}
	v.URL = workspaceurls.ResourceURL(baseURL, "volumes", v.ID)
}

func (v *Volume) GetURL() string {
	return v.URL
}

func (v *Volume) GetName() string {
	return v.Name
}

// ComputeVolumePath returns the Unity Catalog volume path when catalog, schema, and name are set and resolved.
func (v *Volume) ComputeVolumePath() string {
	if v.CatalogName == "" || v.SchemaName == "" || v.Name == "" {
		return ""
	}
	// Bail out if any component still contains a "${...}" reference. We check for the
	// literal "${" rather than a well-formed reference so malformed references (which
	// would otherwise leak verbatim into the path) are also rejected.
	if strings.Contains(v.CatalogName, "${") ||
		strings.Contains(v.SchemaName, "${") ||
		strings.Contains(v.Name, "${") {
		return ""
	}
	return fmt.Sprintf("/Volumes/%s/%s/%s", v.CatalogName, v.SchemaName, v.Name)
}
