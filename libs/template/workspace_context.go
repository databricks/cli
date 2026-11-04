package template

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/jsonschema"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
)

// workspaceVariable is the root template variable that holds the workspace context.
const workspaceVariable = "workspace"

// Query parameters that select a workspace on a host shared by several workspaces.
// Matches what the SDK reads from a host URL (workspaceIDFromQuery in databricks-sdk-go/config).
var workspaceSelectorParams = []string{"w", "o", "workspace_id"}

// Sources of a workspace ID that the user set: a profile, an environment variable, or the
// bundle in the working directory (bundle/config/workspace.go labels its attributes "bundle").
// The SDK records no source for an ID that it resolves from host metadata or from the host
// query, so such an ID reports the dynamic configuration source.
var userSetWorkspaceIDSources = []sdkconfig.SourceType{
	sdkconfig.SourceFile,
	sdkconfig.SourceEnv,
	sdkconfig.SourceType("bundle"),
}

// workspaceContext is exposed to templates as .workspace. Unlike the workspace_host helper,
// it carries the workspace ID, which is required to address a workspace on a host shared by
// several workspaces.
//
// The fields are strings so that templates which print the root data get stable output.
type workspaceContext struct {
	// Hostname is the workspace host without scheme, port, path, or query.
	Hostname string

	host        string
	workspaceID string

	// urlWorkspaceID is the workspace ID that URL adds as ?w=. It is empty unless the user set the
	// ID. Strict profile matching (DATABRICKS_BUNDLE_STRICT_PROFILE_MATCH) compares the ID in the
	// host with the profile's workspace_id, and a profile without workspace_id matches only a host
	// without an ID.
	urlWorkspaceID string
}

// loadWorkspaceContext returns the workspace context of the selected profile, or nil if the
// schema declares a workspace input. A declared input owns the name even before it has a value.
// Input values the schema doesn't declare don't claim the name, because bundle init ignores them.
func loadWorkspaceContext(ctx context.Context, schema *jsonschema.Schema) (*workspaceContext, error) {
	if _, ok := schema.Properties[workspaceVariable]; ok {
		return nil, nil
	}
	cfg := cmdctx.WorkspaceClient(ctx).Config
	host, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workspace host %q: %w", cfg.Host, err)
	}
	workspaceID := cfg.WorkspaceID
	if workspaceID == profile.WorkspaceIDNone {
		workspaceID = ""
	}
	var urlWorkspaceID string
	if attr, ok := cfg.GetAuthDetails().Configuration["workspace_id"]; ok && slices.Contains(userSetWorkspaceIDSources, attr.Source.Type) {
		urlWorkspaceID = workspaceID
	}
	return &workspaceContext{
		Hostname:       host.Hostname(),
		host:           cfg.Host,
		workspaceID:    workspaceID,
		urlWorkspaceID: urlWorkspaceID,
	}, nil
}

// templateData returns the root data for rendering a template. The workspace context is added
// to a copy of the input values, so it never takes part in input defaults, prompts, or validation.
// It replaces an input value the schema doesn't declare, which only schema rendering keeps.
func templateData(values map[string]any, workspace *workspaceContext) map[string]any {
	if workspace == nil {
		return values
	}
	data := make(map[string]any, len(values)+1)
	maps.Copy(data, values)
	data[workspaceVariable] = *workspace
	return data
}

// ID returns the workspace ID from the resolved configuration. It fails when the ID is unknown,
// so that templates don't render a configuration that cannot be routed to a workspace on a host
// shared by several workspaces. It doesn't make a request to discover the ID.
func (c workspaceContext) ID() (string, error) {
	if c.workspaceID == "" {
		return "", errors.New("cannot determine workspace ID, please set workspace_id in the configuration profile or the DATABRICKS_WORKSPACE_ID environment variable")
	}
	return c.workspaceID, nil
}

// URL returns the workspace URL with an optional path relative to the workspace host,
// for example URL "/jobs". It adds ?w=<id> when the profile, an environment variable, or the
// bundle sets the workspace ID.
func (c workspaceContext) URL(path ...string) (string, error) {
	if len(path) > 1 {
		return "", fmt.Errorf("URL accepts at most one path, got %d", len(path))
	}
	base, err := url.Parse(c.host)
	if err != nil {
		return "", fmt.Errorf("failed to parse workspace host %q: %w", c.host, err)
	}
	// Unlike workspaceurls.WorkspaceBaseURL, this adds ?w= also when the hostname has the ID
	// (adb-<id>), because strict profile matching reads the ID only from the query.
	if c.urlWorkspaceID != "" {
		query := base.Query()
		query.Set("w", c.urlWorkspaceID)
		base.RawQuery = query.Encode()
	}
	if len(path) == 0 {
		return base.String(), nil
	}
	ref, err := url.Parse(path[0])
	if err != nil {
		return "", fmt.Errorf("failed to parse workspace URL path %q: %w", path[0], err)
	}
	if ref.Scheme != "" || ref.Host != "" {
		return "", fmt.Errorf("workspace URL path %q must be relative to the workspace host", path[0])
	}
	// url.URL.Query drops pairs it cannot parse, which would hide a workspace selector.
	query, err := url.ParseQuery(ref.RawQuery)
	if err != nil {
		return "", fmt.Errorf("failed to parse query of workspace URL path %q: %w", path[0], err)
	}
	for _, param := range workspaceSelectorParams {
		if query.Has(param) {
			return "", fmt.Errorf("workspace URL path %q must not set the %q query parameter, the workspace is selected by the profile", path[0], param)
		}
	}
	u := base.ResolveReference(ref)
	// The path's query replaces the base URL query, which holds only the workspace selector.
	// Append the selector after the path's parameters to keep their order.
	u.RawQuery = ref.RawQuery
	if base.RawQuery != "" {
		if u.RawQuery != "" {
			u.RawQuery += "&"
		}
		u.RawQuery += base.RawQuery
	}
	return u.String(), nil
}
