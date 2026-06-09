package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

type GenieSpaceConfig struct {
	// Description of the Genie Space
	Description string `json:"description,omitempty"`
	// Etag for change detection. The bundle persists the value the backend
	// returned on the last Create/Update and uses it both as an If-Match for
	// the next Update and as the signal for `bundle plan` to detect remote
	// drift (see OverrideChangeDesc in bundle/direct/dresources/genie_space.go).
	// Mirrors dashboards.DashboardConfig.Etag.
	Etag string `json:"etag,omitempty"`
	// Title of the Genie Space
	Title string `json:"title,omitempty"`
	// Warehouse associated with the Genie Space
	WarehouseId string `json:"warehouse_id,omitempty"`
	// Parent folder path where the space will be registered
	ParentPath string `json:"parent_path,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`

	// ==============================================
	// === overrides over [dashboards.GenieSpace] ===
	// ==============================================

	// SerializedSpace holds the contents of the Genie Space in serialized JSON form.
	// Even though the SDK represents this as a string, we override it as any to allow for inlining as YAML.
	// If the value is a string, it is used as is.
	// If it is not a string, its contents is marshalled as JSON.
	SerializedSpace any `json:"serialized_space,omitempty"`
}

func (c *GenieSpaceConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c GenieSpaceConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type GenieSpace struct {
	BaseResource
	GenieSpaceConfig

	Permissions []Permission `json:"permissions,omitempty"`

	// FilePath points to the local `.geniespace.json` file containing the Genie Space definition.
	// This is inlined into serialized_space during deployment. The file_path is kept around
	// as metadata which is needed for `databricks bundle generate genie-space --resource <key>` to work.
	// This is not part of GenieSpaceConfig because we don't need to store this in the resource state.
	FilePath string `json:"file_path,omitempty"`
}

func (*GenieSpace) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Genie.GetSpace(ctx, dashboards.GenieGetSpaceRequest{
		SpaceId: id,
	})
	if err != nil {
		log.Debugf(ctx, "genie space %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*GenieSpace) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "genie_space",
		PluralName:    "genie_spaces",
		SingularTitle: "Genie Space",
		PluralTitle:   "Genie Spaces",
	}
}

func (r *GenieSpace) InitializeURL(baseURL url.URL) {
	if r.ID == "" {
		return
	}

	r.URL = workspaceurls.ResourceURL(baseURL, "genie_spaces", r.ID)
}

func (r *GenieSpace) GetName() string {
	return r.Title
}

func (r *GenieSpace) GetURL() string {
	return r.URL
}
