package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type App struct {
	BaseResource
	apps.App // nolint App struct also defines Id and URL field with the same json tag "id" and "url"

	// SourceCodePath is a required field used by DABs to point to Databricks app source code
	// on local disk and to the corresponding workspace path during app deployment.
	SourceCodePath string `json:"source_code_path"`

	Permissions []AppPermission `json:"permissions,omitempty"`
}

func (a *App) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, a)
}

func (a App) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(a)
}

func (a *App) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Apps.GetByName(ctx, name)
	if err != nil {
		log.Debugf(ctx, "app %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (*App) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "app",
		PluralName:    "apps",
		SingularTitle: "App",
		PluralTitle:   "Apps",
	}
}

func (a *App) InitializeURL(baseURL url.URL) {
	if a.ModifiedStatus == "" || a.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	baseURL.Path = "apps/" + a.GetName()
	a.URL = baseURL.String()
}

func (a *App) GetName() string {
	// Prefer name from the state - that is what is actually deployed
	if a.ID != "" {
		return a.ID
	}
	return a.Name
}

func (a *App) GetURL() string {
	return a.URL
}
