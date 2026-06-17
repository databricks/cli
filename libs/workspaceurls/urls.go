package workspaceurls

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

var resourceURLPatterns = map[string]string{
	"alerts":                  "sql/alerts-v2/%s",
	"apps":                    "apps/%s",
	"catalogs":                "explore/data/%s",
	"clusters":                "compute/clusters/%s",
	"dashboards":              "dashboardsv3/%s/published",
	"database_catalogs":       "explore/data/%s",
	"database_instances":      "compute/database-instances/%s",
	"experiments":             "ml/experiments/%s",
	"genie_spaces":            "genie/rooms/%s",
	"jobs":                    "jobs/%s",
	"models":                  "ml/models/%s",
	"model_serving_endpoints": "ml/endpoints/%s",
	"notebooks":               "#notebook/%s",
	"pipelines":               "pipelines/%s",
	"postgres_catalogs":       "explore/data/%s",
	"postgres_synced_tables":  "explore/data/%s",
	"quality_monitors":        "explore/data/%s",
	"queries":                 "sql/editor/%s",
	"registered_models":       "explore/data/models/%s",
	"schemas":                 "explore/data/%s",
	"synced_database_tables":  "explore/data/%s",
	"vector_search_endpoints": "compute/vector-search/%s",
	"vector_search_indexes":   "explore/data/%s",
	"volumes":                 "explore/data/volumes/%s",
	"warehouses":              "sql/warehouses/%s",
}

// resourceAliases lets callers use bundle-config plural names as synonyms for
// canonical URL keys. Canonical names match SDK service groups (e.g. the
// `databricks warehouses` CLI group), bundle plural names match the
// resources.<Type>.ResourceDescription().PluralName values (e.g. "sql_warehouses").
// Aliases do not appear in ResourceTypes() so the completion list stays unambiguous.
var resourceAliases = map[string]string{
	"sql_warehouses": "warehouses",
}

// dotSeparatedResources lists resource types where the identifier is commonly
// provided as a dot-separated name (e.g. "catalog.schema.model") but the URL
// requires slash-separated segments.
var dotSeparatedResources = map[string]bool{
	"catalogs":               true,
	"postgres_synced_tables": true,
	"quality_monitors":       true,
	"registered_models":      true,
	"schemas":                true,
	"vector_search_indexes":  true,
	"volumes":                true,
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

// JobRunURL converts a legacy job-run URL of the form
//
//	https://<host>/?o=<id>#job/<jobID>/run/<runID>
//
// into the modern path form
//
//	https://<host>/jobs/<jobID>/runs/<runID>?o=<id>
//
// The legacy fragment form relies on client-side redirection that only
// resolves for workspace admins; non-admin users with CAN_VIEW are sent to the
// workspace homepage instead. The modern path form resolves for any user
// permitted to view the run. The workspace selector query param (o) is
// preserved as-is from the original URL.
// See https://github.com/databricks/cli/issues/5142.
func JobRunURL(legacy string) (string, error) {
	u, err := url.Parse(legacy)
	if err != nil {
		return "", fmt.Errorf("parsing run URL %q: %w", legacy, err)
	}

	jobID, runID, ok := parseLegacyRunFragment(u.Fragment)
	if !ok {
		return "", fmt.Errorf("unexpected run URL fragment %q", u.Fragment)
	}

	u.Fragment = ""
	u.Path = fmt.Sprintf("/jobs/%s/runs/%s", jobID, runID)
	return u.String(), nil
}

// parseLegacyRunFragment extracts the job and run IDs from a legacy run URL
// fragment of the form "job/<jobID>/run/<runID>".
func parseLegacyRunFragment(fragment string) (jobID, runID string, ok bool) {
	parts := strings.Split(fragment, "/")
	if len(parts) != 4 || parts[0] != "job" || parts[2] != "run" || parts[1] == "" || parts[3] == "" {
		return "", "", false
	}
	return parts[1], parts[3], true
}

// ResourceURL constructs a workspace URL for a named resource type and ID.
func ResourceURL(baseURL url.URL, resourceType, id string) string {
	resourceType = resolveAlias(resourceType)
	pattern, ok := resourceURLPatterns[resourceType]
	if !ok {
		return ""
	}
	id = normalizeDotSeparatedID(resourceType, id)
	return formatResourceURL(baseURL, pattern, id)
}

// BuildResourceURL constructs a full workspace URL from a host string, resource
// type name, ID, and workspace ID. It parses the host, appends ?w=<workspace-id>
// when needed, and formats the resource path. An empty workspaceID skips the
// query parameter (use when the workspace ID is unknown or already encoded in
// the hostname).
func BuildResourceURL(host, resourceType, id, workspaceID string) (string, error) {
	baseURL, err := workspaceBaseURL(host, workspaceID)
	if err != nil {
		return "", err
	}

	result := ResourceURL(*baseURL, resourceType, id)
	if result == "" {
		return "", fmt.Errorf("unknown resource type %q, must be one of: %s", resourceType, strings.Join(ResourceTypes(), ", "))
	}
	return result, nil
}

func resolveAlias(resourceType string) string {
	if canonical, ok := resourceAliases[resourceType]; ok {
		return canonical
	}
	return resourceType
}

func workspaceBaseURL(host, workspaceID string) (*url.URL, error) {
	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace host %q: %w", host, err)
	}

	if workspaceID == "" {
		return baseURL, nil
	}

	if hasWorkspaceIDInHostname(baseURL.Hostname(), workspaceID) {
		return baseURL, nil
	}

	values := baseURL.Query()
	values.Add("w", workspaceID)
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
