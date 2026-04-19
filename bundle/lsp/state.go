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

// LoadResourceState reads the direct engine deployment state to get resource IDs.
// Returns a map from resource path (e.g., "resources.jobs.my_job") to ResourceInfo.
func LoadResourceState(bundleRoot, target string) map[string]ResourceInfo {
	directPath := filepath.Join(bundleRoot, ".databricks", "bundle", target, "resources.json")
	if info := loadDirectState(directPath); info != nil {
		return info
	}
	return make(map[string]ResourceInfo)
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
