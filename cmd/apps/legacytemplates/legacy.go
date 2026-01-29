package legacytemplates

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed app-template-app-manifests.json
var appTemplateManifestsJSON []byte

// resourceSpec represents a resource specification in the template manifest.
type resourceSpec struct {
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	SQLWarehouseSpec    *map[string]any `json:"sql_warehouse_spec,omitempty"`
	ExperimentSpec      *map[string]any `json:"experiment_spec,omitempty"`
	ServingEndpointSpec *map[string]any `json:"serving_endpoint_spec,omitempty"`
	DatabaseSpec        *map[string]any `json:"database_spec,omitempty"`
	UCSecurableSpec     *map[string]any `json:"uc_securable_spec,omitempty"`
}

// manifest represents the manifest section of a template.
type manifest struct {
	Version       int            `json:"version"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	StartCommand  string         `json:"start_command,omitempty"`
	ResourceSpecs []resourceSpec `json:"resource_specs,omitempty"`
	UserAPIScopes []string       `json:"user_api_scopes,omitempty"`
}

// AppTemplateManifest represents a single app template from the manifests JSON file.
type AppTemplateManifest struct {
	Path     string   `json:"path"`
	GitRepo  string   `json:"git_repo"`
	Manifest manifest `json:"manifest"`
}

// appTemplateManifests holds all app templates.
type appTemplateManifests struct {
	Templates []AppTemplateManifest `json:"appTemplateAppManifests"`
}

// LoadLegacyTemplates loads the legacy app templates from the embedded JSON file.
func LoadLegacyTemplates() ([]AppTemplateManifest, error) {
	var manifests appTemplateManifests
	if err := json.Unmarshal(appTemplateManifestsJSON, &manifests); err != nil {
		return nil, fmt.Errorf("failed to load app template manifests: %w", err)
	}
	return manifests.Templates, nil
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
	for _, spec := range tmpl.Manifest.ResourceSpecs {
		if checker(&spec) {
			return true
		}
	}
	return false
}

// RequiresSQLWarehouse checks if the template requires a SQL warehouse.
func RequiresSQLWarehouse(tmpl *AppTemplateManifest) bool {
	return hasResourceSpec(tmpl, func(s *resourceSpec) bool { return s.SQLWarehouseSpec != nil })
}

// RequiresServingEndpoint checks if the template requires a serving endpoint.
func RequiresServingEndpoint(tmpl *AppTemplateManifest) bool {
	return hasResourceSpec(tmpl, func(s *resourceSpec) bool { return s.ServingEndpointSpec != nil })
}

// RequiresExperiment checks if the template requires an experiment.
func RequiresExperiment(tmpl *AppTemplateManifest) bool {
	return hasResourceSpec(tmpl, func(s *resourceSpec) bool { return s.ExperimentSpec != nil })
}

// RequiresDatabase checks if the template requires a database.
func RequiresDatabase(tmpl *AppTemplateManifest) bool {
	return hasResourceSpec(tmpl, func(s *resourceSpec) bool { return s.DatabaseSpec != nil })
}

// RequiresUCVolume checks if the template requires a UC volume.
func RequiresUCVolume(tmpl *AppTemplateManifest) bool {
	return hasResourceSpec(tmpl, func(s *resourceSpec) bool { return s.UCSecurableSpec != nil })
}
