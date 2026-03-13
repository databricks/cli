package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/dyn"
)

// ResourceInfo holds deployment info for a resource.
type ResourceInfo struct {
	ID   string
	URL  string
	Name string
}

// LoadResourceState reads the local deployment state to get resource IDs.
// It tries direct engine state first, then terraform state.
// Returns a map from resource path (e.g., "resources.jobs.my_job") to ResourceInfo.
func LoadResourceState(bundleRoot, target string) map[string]ResourceInfo {
	result := make(map[string]ResourceInfo)

	// Try direct engine state first.
	directPath := filepath.Join(bundleRoot, ".databricks", "bundle", target, "resources.json")
	if info := loadDirectState(directPath); info != nil {
		for k, v := range info {
			result[k] = v
		}
		return result
	}

	// Try terraform state.
	tfPath := filepath.Join(bundleRoot, ".databricks", "bundle", target, "terraform", "terraform.tfstate")
	if info := loadTerraformState(tfPath); info != nil {
		for k, v := range info {
			result[k] = v
		}
	}

	return result
}

type directState struct {
	State map[string]directResourceEntry `json:"state"`
}

type directResourceEntry struct {
	ID string `json:"__id__"`
}

func loadDirectState(path string) map[string]ResourceInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var state directState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}

	result := make(map[string]ResourceInfo)
	for key, entry := range state.State {
		result[key] = ResourceInfo{ID: entry.ID}
	}
	return result
}

type terraformState struct {
	Version   int                 `json:"version"`
	Resources []terraformResource `json:"resources"`
}

type terraformResource struct {
	Type      string              `json:"type"`
	Name      string              `json:"name"`
	Mode      string              `json:"mode"`
	Instances []terraformInstance `json:"instances"`
}

type terraformInstance struct {
	Attributes terraformAttributes `json:"attributes"`
}

type terraformAttributes struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// terraformToGroup maps terraform resource types to bundle resource group names.
var terraformToGroup = map[string]string{
	"databricks_job":               "jobs",
	"databricks_pipeline":          "pipelines",
	"databricks_mlflow_model":      "models",
	"databricks_mlflow_experiment": "experiments",
	"databricks_model_serving":     "model_serving_endpoints",
	"databricks_registered_model":  "registered_models",
	"databricks_quality_monitor":   "quality_monitors",
	"databricks_catalog":           "catalogs",
	"databricks_schema":            "schemas",
	"databricks_volume":            "volumes",
	"databricks_cluster":           "clusters",
	"databricks_sql_dashboard":     "dashboards",
	"databricks_dashboard":         "dashboards",
	"databricks_app":               "apps",
	"databricks_secret_scope":      "secret_scopes",
	"databricks_sql_alert":         "alerts",
}

func loadTerraformState(path string) map[string]ResourceInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var state terraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}

	if state.Version != 4 {
		return nil
	}

	result := make(map[string]ResourceInfo)
	for _, r := range state.Resources {
		if r.Mode != "managed" {
			continue
		}
		group, ok := terraformToGroup[r.Type]
		if !ok {
			continue
		}
		if len(r.Instances) == 0 {
			continue
		}

		id := r.Instances[0].Attributes.ID
		name := r.Instances[0].Attributes.Name
		key := "resources." + group + "." + r.Name

		result[key] = ResourceInfo{
			ID:   id,
			Name: name,
		}
	}

	return result
}

// LoadWorkspaceHost extracts the workspace host from the bundle YAML config.
func LoadWorkspaceHost(v dyn.Value) string {
	host := v.Get("workspace").Get("host")
	s, ok := host.AsString()
	if !ok {
		return ""
	}
	// Skip interpolation references like ${var.host}.
	if strings.Contains(s, "${") {
		return ""
	}
	return s
}

// LoadTarget extracts the default target name from the bundle config.
func LoadTarget(v dyn.Value) string {
	targets := v.Get("targets")
	if targets.Kind() != dyn.KindMap {
		return "default"
	}
	tm, ok := targets.AsMap()
	if !ok {
		return "default"
	}

	// Look for a target with default: true.
	for _, pair := range tm.Pairs() {
		d := pair.Value.Get("default")
		if b, ok := d.AsBool(); ok && b {
			return pair.Key.MustString()
		}
	}

	// Return first target if none marked as default.
	if tm.Len() > 0 {
		return tm.Pairs()[0].Key.MustString()
	}
	return "default"
}

// LoadAllTargets returns all target names from the parsed config.
func LoadAllTargets(v dyn.Value) []string {
	targets := v.Get("targets")
	if targets.Kind() != dyn.KindMap {
		return nil
	}
	tm, ok := targets.AsMap()
	if !ok {
		return nil
	}
	var names []string
	for _, pair := range tm.Pairs() {
		names = append(names, pair.Key.MustString())
	}
	return names
}

// LoadTargetWorkspaceHost extracts workspace host from a specific target override,
// falling back to the root-level workspace host.
func LoadTargetWorkspaceHost(v dyn.Value, target string) string {
	// Try target-specific override first.
	host := v.Get("targets").Get(target).Get("workspace").Get("host")
	if s, ok := host.AsString(); ok && !strings.Contains(s, "${") {
		return s
	}
	// Fall back to root-level.
	return LoadWorkspaceHost(v)
}

// BuildResourceURL constructs the workspace URL for a resource.
func BuildResourceURL(host, resourceType, id string) string {
	if host == "" || id == "" {
		return ""
	}
	host = strings.TrimRight(host, "/")

	switch resourceType {
	case "jobs":
		return host + "/jobs/" + id
	case "pipelines":
		return host + "/pipelines/" + id
	case "dashboards":
		return host + "/dashboardsv3/" + id + "/published"
	case "model_serving_endpoints":
		return host + "/ml/endpoints/" + id
	case "experiments":
		return host + "/ml/experiments/" + id
	case "models":
		return host + "/ml/models/" + id
	case "clusters":
		return host + "/compute/clusters/" + id
	case "apps":
		return host + "/apps/" + id
	case "alerts":
		return host + "/sql/alerts-v2/" + id
	case "sql_warehouses":
		return host + "/sql/warehouses/" + id
	case "quality_monitors":
		return host + "/explore/data/" + id
	case "secret_scopes":
		return host + "/secrets/scopes/" + id
	default:
		return host
	}
}
