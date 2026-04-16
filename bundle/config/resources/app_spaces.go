package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type AppSpace struct {
	BaseResource
	apps.Space
}

func (s *AppSpace) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s AppSpace) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *AppSpace) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Apps.GetSpace(ctx, apps.GetSpaceRequest{Name: id})
	if err != nil {
		log.Debugf(ctx, "app space %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*AppSpace) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "app_space",
		PluralName:    "app_spaces",
		SingularTitle: "App Space",
		PluralTitle:   "App Spaces",
	}
}

func (s *AppSpace) InitializeURL(baseURL url.URL) {
	if s.ModifiedStatus == "" || s.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	baseURL.Path = "apps/spaces/" + s.GetName()
	s.URL = baseURL.String()
}

func (s *AppSpace) GetName() string {
	if s.ID != "" {
		return s.ID
	}
	return s.Name
}

func (s *AppSpace) GetURL() string {
	return s.URL
}
