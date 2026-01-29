package legacytemplates

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed apps-templates.json
var appTemplatesJSON []byte

// resourceSpec represents a resource specification in the template.
type resourceSpec struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	SQLWarehouse    *map[string]any `json:"sql_warehouse,omitempty"`
	Experiment      *map[string]any `json:"experiment,omitempty"`
	ServingEndpoint *map[string]any `json:"serving_endpoint,omitempty"`
	Database        *map[string]any `json:"database,omitempty"`
	UCSecurable     *map[string]any `json:"uc_securable,omitempty"`
}

// AppTemplateManifest represents a single app template from the templates JSON file.
type AppTemplateManifest struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Resources     []resourceSpec `json:"resources,omitempty"`
	GitRepo       string         `json:"git_repo"`
	Path          string         `json:"path"`
	GitProvider   string         `json:"git_provider"`
	UseCase       string         `json:"use_case,omitempty"`
	FrameworkType string         `json:"framework_type,omitempty"`
	UserAPIScopes []string       `json:"user_api_scopes,omitempty"`
}

// appTemplates holds all app templates.
type appTemplates struct {
	Templates []AppTemplateManifest `json:"templates"`
}

// LoadLegacyTemplates loads the legacy app templates from the embedded JSON file.
func LoadLegacyTemplates() ([]AppTemplateManifest, error) {
	var templates appTemplates
	if err := json.Unmarshal(appTemplatesJSON, &templates); err != nil {
		return nil, fmt.Errorf("failed to load app templates: %w", err)
	}
	return templates.Templates, nil
}

// FindLegacyTemplateByPath finds a legacy template by its path identifier.
// Returns nil if no matching template is found.
func FindLegacyTemplateByPath(templates []AppTemplateManifest, path string) *AppTemplateManifest {
	for i := range templates {
		if templates[i].Path == path {
			return &templates[i]
		}
	}
	return nil
}

// resourceSpecChecker is a function that checks if a resourceSpec has a specific field.
type resourceSpecChecker func(*resourceSpec) bool

// hasResourceSpec checks if a template requires a resource based on a spec checker.
func hasResourceSpec(tmpl *AppTemplateManifest, checker resourceSpecChecker) bool {
	for _, spec := range tmpl.Resources {
		if checker(&spec) {
			return true
		}
	}
	return false
}
