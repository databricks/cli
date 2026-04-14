package workspaceurls

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

var resourceURLPatterns = map[string]string{
	"alerts":                  "sql/alerts-v2/%s",
	"apps":                    "apps/%s",
	"clusters":                "compute/clusters/%s",
	"dashboards":              "dashboardsv3/%s/published",
	"experiments":             "ml/experiments/%s",
	"jobs":                    "jobs/%s",
	"models":                  "ml/models/%s",
	"model_serving_endpoints": "ml/endpoints/%s",
	"notebooks":               "#notebook/%s",
	"pipelines":               "pipelines/%s",
	"queries":                 "sql/editor/%s",
	"registered_models":       "explore/data/models/%s",
	"warehouses":              "sql/warehouses/%s",
}

// dotSeparatedResources lists resource types where the identifier is commonly
// provided as a dot-separated name (e.g. "catalog.schema.model") but the URL
// requires slash-separated segments.
var dotSeparatedResources = map[string]bool{
	"registered_models": true,
}

// ResourceTypes returns a sorted list of all supported resource type names.
func ResourceTypes() []string {
	names := make([]string, 0, len(resourceURLPatterns))
	for k := range resourceURLPatterns {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

// ResourceURL constructs a workspace URL for a named resource type and ID.
func ResourceURL(baseURL url.URL, resourceType, id string) string {
	pattern, ok := resourceURLPatterns[resourceType]
	if !ok {
		return ""
	}
	id = normalizeDotSeparatedID(resourceType, id)
	return formatResourceURL(baseURL, pattern, id)
}

// BuildResourceURL constructs a full workspace URL from a host string, resource
// type name, ID, and workspace ID. It parses the host, appends ?o=<workspace-id>
// when needed, and formats the resource path.
func BuildResourceURL(host, resourceType, id string, workspaceID int64) (string, error) {
	baseURL, err := WorkspaceBaseURL(host, workspaceID)
	if err != nil {
		return "", err
	}

	result := ResourceURL(*baseURL, resourceType, id)
	if result == "" {
		return "", fmt.Errorf("unknown resource type %q, must be one of: %s", resourceType, strings.Join(ResourceTypes(), ", "))
	}
	return result, nil
}

// WorkspaceBaseURL parses a workspace host and appends ?o=<workspace-id> when needed.
func WorkspaceBaseURL(host string, workspaceID int64) (*url.URL, error) {
	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace host %q: %w", host, err)
	}

	if workspaceID == 0 {
		return baseURL, nil
	}

	orgID := strconv.FormatInt(workspaceID, 10)
	if hasWorkspaceIDInHostname(baseURL.Hostname(), orgID) {
		return baseURL, nil
	}

	values := baseURL.Query()
	values.Add("o", orgID)
	baseURL.RawQuery = values.Encode()

	return baseURL, nil
}

func normalizeDotSeparatedID(resourceType, id string) string {
	if dotSeparatedResources[resourceType] {
		return strings.ReplaceAll(id, ".", "/")
	}
	return id
}

func formatResourceURL(baseURL url.URL, pattern, id string) string {
	resourcePath := fmt.Sprintf(pattern, id)
	if strings.HasPrefix(resourcePath, "#") {
		baseURL.Path = "/"
		baseURL.Fragment = resourcePath[1:]
	} else {
		baseURL.Path = resourcePath
	}

	return baseURL.String()
}

func hasWorkspaceIDInHostname(hostname, workspaceID string) bool {
	remainder, ok := strings.CutPrefix(strings.ToLower(hostname), "adb-"+workspaceID)
	return ok && (remainder == "" || strings.HasPrefix(remainder, "."))
}
