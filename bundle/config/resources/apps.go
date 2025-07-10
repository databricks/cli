package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type AppPermissionLevel string

// AppPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any app.
type AppPermission struct {
	Level AppPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type App struct {
	// This is app's name pulled from the state. Usually the same as Name but may be different if Name in the config
	// was changed but the app was not re-deployed yet.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// SourceCodePath is a required field used by DABs to point to Databricks app source code
	// on local disk and to the corresponding workspace path during app deployment.
	SourceCodePath string `json:"source_code_path"`

	// Config is an optional field which allows configuring the app following Databricks app configuration format like in app.yml.
	// When this field is set, DABs read the configuration set in this field and write
	// it to app.yml in the root of the source code folder in Databricks workspace.
	// If there’s app.yml defined locally, DABs will raise an error.
	Config map[string]any `json:"config,omitempty"`

	Permissions    []AppPermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus  `json:"modified_status,omitempty" bundle:"internal"`
	URL            string          `json:"url,omitempty" bundle:"internal"`

	apps.App
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
