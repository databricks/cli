package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

// GenieSpaceConfig holds configuration for a Genie Space resource.
// This is persisted in the deployment state.
type GenieSpaceConfig struct {
	// SpaceId is the UUID of the Genie space (output only).
	SpaceId string `json:"space_id,omitempty"`

	// Title is the display title of the Genie space.
	Title string `json:"title,omitempty"`

	// Description is an optional description for the Genie space.
	Description string `json:"description,omitempty"`

	// ParentPath is the workspace folder path where the space is registered.
	// Example: /Workspace/Users/user@example.com
	ParentPath string `json:"parent_path,omitempty"`

	// WarehouseId is the SQL warehouse ID to associate with this space.
	// This is required for creating a Genie space.
	WarehouseId string `json:"warehouse_id,omitempty"`

	// SerializedSpace holds the contents of the Genie space in serialized JSON form.
	// Even though the SDK represents this as a string, we override it as any to allow
	// for inlining as YAML. If the value is a string, it is used as is.
	// If it is not a string, its contents is marshalled as JSON.
	//
	// The JSON structure includes instructions, sample questions, and data sources.
	// Example:
	//   {
	//     "version": 1,
	//     "config": {
	//       "sample_questions": [{"id": "...", "question": ["Show orders by date"]}],
	//       "instructions": [{"id": "...", "content": "Use MM/DD/YYYY date format"}]
	//     },
	//     "data_sources": {
	//       "tables": [{"identifier": "catalog.schema.table_name"}]
	//     }
	//   }
	SerializedSpace any `json:"serialized_space,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

func (c *GenieSpaceConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c GenieSpaceConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

// GenieSpace represents a Genie Space resource in a Databricks Asset Bundle.
type GenieSpace struct {
	BaseResource
	GenieSpaceConfig

	// FilePath points to the local JSON file containing the Genie space definition.
	// This is inlined into serialized_space during deployment.
	// The file should contain the JSON structure with instructions, sample questions,
	// and data sources that define the Genie space.
	// This is not part of GenieSpaceConfig because we don't need to store this in state.
	FilePath string `json:"file_path,omitempty"`

	Permissions []GenieSpacePermission `json:"permissions,omitempty"`
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
	baseURL.Path = "genie/rooms/" + r.ID
	r.URL = baseURL.String()
}

func (r *GenieSpace) GetName() string {
	return r.Title
}

func (r *GenieSpace) GetURL() string {
	return r.URL
}
