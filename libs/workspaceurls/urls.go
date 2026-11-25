package workspaceurls

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// Pattern is the URL of a resource's page in the workspace UI, relative to the
// workspace root.
type Pattern struct {
	// Path holds a single %s in place of the resource ID. A leading "#" places
	// the path in the URL fragment.
	Path string

	// DotSeparatedID is set when the ID is commonly given as a dot-separated name
	// (e.g. "catalog.schema.model") but the URL requires slash-separated segments.
	DotSeparatedID bool
}

// URL returns the URL of the page of the resource with the given ID.
func (p Pattern) URL(baseURL url.URL, id string) string {
	if p.DotSeparatedID {
		id = strings.ReplaceAll(id, ".", "/")
	}
	return formatResourceURL(baseURL, p.Path, id)
}

// ownPatterns holds the URL patterns of the resource types that callers do not
// pass in, because they are not bundle resources.
var ownPatterns = map[string]Pattern{
	"notebooks":  {Path: "#notebook/%s"},
	"queries":    {Path: "sql/editor/%s"},
	"warehouses": {Path: "sql/warehouses/%s"},
}

// resourceAliases lets callers use bundle-config plural names as synonyms for
// canonical URL keys. Canonical names match SDK service groups (e.g. the
// `databricks warehouses` CLI group), bundle plural names match the
// resources.<Type>.ResourceDescription().PluralName values (e.g. "sql_warehouses").
// Aliases do not appear in ResourceTypes() so the completion list stays unambiguous.
var resourceAliases = map[string]string{
	"sql_warehouses": "warehouses",
}

// ResourceTypes returns a sorted list of all supported resource type names:
// the keys of patterns and the types this package knows itself, without aliases.
func ResourceTypes(patterns map[string]Pattern) []string {
	names := make([]string, 0, len(patterns)+len(ownPatterns))
	for k := range patterns {
		if _, ok := resourceAliases[k]; !ok {
			names = append(names, k)
		}
	}
	for k := range ownPatterns {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

// DeploymentURL returns the workspace URL for a bundle deployment:
// <host>/deployments/<deploymentID>?version=<version>. Version pins the page to the deploy that produced it.
func DeploymentURL(baseURL url.URL, deploymentID string, version int) string {
	if deploymentID == "" {
		return ""
	}

	baseURL.Path = "deployments/" + deploymentID
	if version > 0 {
		values := baseURL.Query()
		values.Set("version", strconv.Itoa(version))
		baseURL.RawQuery = values.Encode()
	}
	return baseURL.String()
}

// JobRunPath returns the modern workspace path for a job run, of the form
//
//	jobs/<jobID>/runs/<runID>
//
// Callers join this onto a workspace base URL. It exists because the Jobs API
// returns a legacy fragment URL (#job/<jobID>/run/<runID>) that relies on
// client-side redirection that only resolves for workspace admins; non-admin
// users with CAN_VIEW are sent to the workspace homepage instead. The modern
// path form resolves for any user permitted to view the run.
// See https://github.com/databricks/cli/issues/5142.
func JobRunPath(jobID, runID string) string {
	return fmt.Sprintf("jobs/%s/runs/%s", jobID, runID)
}

// JobRunURL constructs a workspace URL for a job run. It takes two IDs (parent
// job and run), so it can't be expressed as a [Pattern].
func JobRunURL(baseURL url.URL, jobID, runID string) string {
	baseURL.Path = JobRunPath(jobID, runID)
	return baseURL.String()
}

// ModernizeJobRunPageURL converts the legacy run URL returned by the Jobs API
//
//	https://<host>/?o=<id>#job/<jobID>/run/<runID>
//
// into the modern path form
//
//	https://<host>/jobs/<jobID>/runs/<runID>?o=<id>
//
// so that non-admin users permitted to view the run are not redirected to the
// workspace homepage. See https://github.com/databricks/cli/issues/5142. The
// workspace selector query param (o) is preserved as-is. The conversion is
// cosmetic, so the original URL is returned on the rare chance the format is
// unexpected.
func ModernizeJobRunPageURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	jobID, runID, ok := parseLegacyRunFragment(u.Fragment)
	if !ok {
		return raw
	}

	u.Fragment = ""
	u.Path = "/" + JobRunPath(jobID, runID)
	return u.String()
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

// lookupPattern returns the pattern of the named resource type, looking it up
// in patterns and in the types this package knows itself.
func lookupPattern(patterns map[string]Pattern, resourceType string) (Pattern, bool) {
	if canonical, ok := resourceAliases[resourceType]; ok {
		resourceType = canonical
	}
	if p, ok := ownPatterns[resourceType]; ok {
		return p, true
	}
	p, ok := patterns[resourceType]
	return p, ok
}

// BuildResourceURL constructs a full workspace URL from a host string, resource
// type name, ID, and workspace ID. The type is looked up in patterns and in the
// types this package knows itself. It parses the host, appends ?w=<workspace-id>
// when needed, and formats the resource path. An empty workspaceID skips the
// query parameter (use when the workspace ID is unknown or already encoded in
// the hostname).
func BuildResourceURL(patterns map[string]Pattern, host, resourceType, id, workspaceID string) (string, error) {
	baseURL, err := workspaceBaseURL(host, workspaceID)
	if err != nil {
		return "", err
	}

	pattern, ok := lookupPattern(patterns, resourceType)
	if !ok {
		return "", fmt.Errorf("unknown resource type %q, must be one of: %s", resourceType, strings.Join(ResourceTypes(patterns), ", "))
	}
	return pattern.URL(*baseURL, id), nil
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
