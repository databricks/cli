package resources

import (
	"context"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/utils"
	sdkexternallocations "github.com/databricks/sdk-go/uc/externallocations/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
)

type ExternalLocation struct {
	// Manually include BaseResource fields to avoid URL field conflict
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	// Note: We intentionally don't include BaseResource.URL here to avoid conflict with Url field below
	Lifecycle Lifecycle `json:"lifecycle,omitempty"`

	CreateExternalLocation

	// List of grants to apply on this external location.
	Grants []grants.PrivilegeAssignment `json:"grants,omitempty"`
}

func (e *ExternalLocation) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.ExternalLocations.GetExternalLocation(ctx, sdkexternallocations.GetExternalLocationRequest{NameArg: new(name)})
	if err != nil {
		log.Debugf(ctx, "external location with name %s does not exist: %v", name, err)

		if apierrs.IsMissing(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (*ExternalLocation) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "external_location",
		PluralName:    "external_locations",
		SingularTitle: "External Location",
		PluralTitle:   "External Locations",
	}
}

func (e *ExternalLocation) InitializeURL(baseURL url.URL) {
	// External locations don't have a workspace URL
	// The Url field is for the storage path (s3://...), not a workspace URL
}

func (e *ExternalLocation) GetURL() (string, bool) {
	// External locations don't have a workspace URL
	return "", false
}

func (e *ExternalLocation) GetName() string {
	return utils.Deref(e.Name)
}

// GetLifecycle returns the lifecycle settings for the resource.
func (e *ExternalLocation) GetLifecycle() LifecycleConfig {
	return e.Lifecycle
}
