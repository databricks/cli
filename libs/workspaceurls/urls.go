package workspaceurls

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

const (
	ResourceAlerts                = "alerts"
	ResourceApps                  = "apps"
	ResourceClusters              = "clusters"
	ResourceDashboards            = "dashboards"
	ResourceExperiments           = "experiments"
	ResourceJobs                  = "jobs"
	ResourceModels                = "models"
	ResourceModelServingEndpoints = "model_serving_endpoints"
	ResourceNotebooks             = "notebooks"
	ResourcePipelines             = "pipelines"
	ResourceQueries               = "queries"
	ResourceRegisteredModels      = "registered_models"
	ResourceWarehouses            = "warehouses"
)

const (
	AlertPattern                = "sql/alerts-v2/%s"
	AppPattern                  = "apps/%s"
	ClusterPattern              = "compute/clusters/%s"
	DashboardPattern            = "dashboardsv3/%s/published"
	ExperimentPattern           = "ml/experiments/%s"
	JobPattern                  = "jobs/%s"
	ModelPattern                = "ml/models/%s"
	ModelServingEndpointPattern = "ml/endpoints/%s"
	NotebookPattern             = "#notebook/%s"
	PipelinePattern             = "pipelines/%s"
	QueryPattern                = "sql/editor/%s"
	RegisteredModelPattern      = "explore/data/models/%s"
	WarehousePattern            = "sql/warehouses/%s"
)

var resourceURLPatterns = map[string]string{
	ResourceAlerts:                AlertPattern,
	ResourceApps:                  AppPattern,
	ResourceClusters:              ClusterPattern,
	ResourceDashboards:            DashboardPattern,
	ResourceExperiments:           ExperimentPattern,
	ResourceJobs:                  JobPattern,
	ResourceModels:                ModelPattern,
	ResourceModelServingEndpoints: ModelServingEndpointPattern,
	ResourceNotebooks:             NotebookPattern,
	ResourcePipelines:             PipelinePattern,
	ResourceQueries:               QueryPattern,
	ResourceRegisteredModels:      RegisteredModelPattern,
	ResourceWarehouses:            WarehousePattern,
}

// LookupPattern returns the workspace URL pattern for a resource type.
func LookupPattern(resourceType string) (string, bool) {
	pattern, ok := resourceURLPatterns[resourceType]
	return pattern, ok
}

// SortResourceTypes returns a sorted copy of resource types.
func SortResourceTypes(resourceTypes []string) []string {
	names := append([]string(nil), resourceTypes...)
	slices.Sort(names)
	return names
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

// BuildResourceURL constructs a full workspace URL for a resource path pattern.
func BuildResourceURL(host, pattern, id string, workspaceID int64) (string, error) {
	baseURL, err := WorkspaceBaseURL(host, workspaceID)
	if err != nil {
		return "", err
	}

	return ResourceURL(*baseURL, pattern, id), nil
}

// ResourceURL formats a workspace URL for a resource path pattern.
func ResourceURL(baseURL url.URL, pattern, id string) string {
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
