package resources

import (
	"context"
	"fmt"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type App struct {
	// This represents the id which is the name of the app that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// SourceCodePath is a required field used by DABs to point databricks app source code
	// on local disk and use it to point to this source code in the app deployment
	SourceCodePath string `json:"source_code_path"`

	// Config is an optional field which allows configuring the app following Databricks app configuration format like in app.yml.
	// When this field is set, DABs read the configuration set in this field and write
	// it to app.yml in the root of the source code folder in Databricks workspace.
	// If there’s app.yml defined already, it will be overridden.
	Config map[string]interface{} `json:"config,omitempty"`

	Permissions    []Permission   `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`

	*apps.App
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

func (a *App) TerraformResourceName() string {
	return "databricks_cluster"
}

func (a *App) InitializeURL(baseURL url.URL) {
	if a.ID == "" {
		return
	}
	baseURL.Path = fmt.Sprintf("apps/%s", a.ID)
	a.URL = baseURL.String()
}

func (a *App) GetName() string {
	return a.Name
}

func (a *App) GetURL() string {
	return a.URL
}

func (a *App) IsNil() bool {
	return a.App == nil
}
