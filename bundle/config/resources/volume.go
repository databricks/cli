package resources

import (
	"context"
	"fmt"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/utils"
	sdkvolumes "github.com/databricks/sdk-go/uc/volumes/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type Volume struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	CreateVolumeRequestContent

	// VolumePath is /Volumes/{catalog}/{schema}/{name}. Populated during initialize; not user-configurable.
	VolumePath string `json:"volume_path,omitempty" bundle:"readonly"`

	// List of grants to apply on this volume.
	Grants []grants.PrivilegeAssignment `json:"grants,omitempty"`
}

func (v *Volume) Exists(ctx context.Context, w *sdkclients.Clients, fullyQualifiedName string) (bool, error) {
	_, err := w.Volumes.GetVolume(ctx, sdkvolumes.GetVolumeRequest{
		FullNameArg: new(fullyQualifiedName),
	})
	if err != nil {
		log.Debugf(ctx, "volume with fully qualified name %s does not exist: %v", fullyQualifiedName, err)

		if apierrs.IsMissing(err) {
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

func (v *Volume) GetName() string {
	return utils.Deref(v.Name)
}

// ComputeVolumePath returns the Unity Catalog volume path /Volumes/{catalog}/{schema}/{name}.
//
// Components are used as-is: a ${...} reference is embedded verbatim and resolved later during plan
// or deploy. Missing required fields and malformed references are reported by validation and the
// plan/deploy backends, so this function does not guard against them.
func (v *Volume) ComputeVolumePath() string {
	return fmt.Sprintf("/Volumes/%s/%s/%s", v.CatalogName, v.SchemaName, v.Name)
}
