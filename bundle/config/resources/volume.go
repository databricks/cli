package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type Volume struct {
	BaseResource
	catalog.CreateVolumeRequestContent

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
	baseURL.Path = "explore/data/volumes/" + strings.ReplaceAll(v.ID, ".", "/")
	v.URL = baseURL.String()
}

func (v *Volume) GetURL() string {
	return v.URL
}

func (v *Volume) GetName() string {
	return v.Name
}
