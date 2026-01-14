package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

// AppConfig represents the inline app.yaml configuration structure.
// This matches the structure of an app.yaml file that can be used to configure how the app runs.
type AppConfig struct {
	// Command specifies the command to run the app (e.g., ["streamlit", "run", "app.py"])
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`

	// Env contains environment variables to set for the app
	Env []AppEnvVar `json:"env,omitempty" yaml:"env,omitempty"`
}

// AppEnvVar represents an environment variable configuration for an app
type AppEnvVar struct {
	// Name is the environment variable name
	Name string `json:"name" yaml:"name"`

	// Value is the environment variable value
	Value string `json:"value" yaml:"value"`
}

type App struct {
	BaseResource
	apps.App // nolint App struct also defines Id and URL field with the same json tag "id" and "url"

	// SourceCodePath is a required field used by DABs to point to Databricks app source code
	// on local disk and to the corresponding workspace path during app deployment.
	SourceCodePath string `json:"source_code_path"`

	// Config represents inline app.yaml configuration for the app.
	// When specified, this configuration is written to an app.yaml file in the source code path during deployment.
	// This allows users to define app configuration directly in the bundle YAML instead of maintaining a separate app.yaml file.
	Config *AppConfig `json:"config,omitempty"`

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
