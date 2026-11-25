package resources

import (
	"net/url"

	"github.com/databricks/cli/libs/workspaceurls"
)

// WorkspaceURLDeclarer is implemented by every resource type, in the type's own file.
type WorkspaceURLDeclarer interface {
	// WorkspaceURL returns the URL pattern of the type's page in the workspace UI
	// and true, or false if the type has no page. The pattern is empty if the page
	// is not addressed by a single ID; InitializeURL then builds the URL itself.
	WorkspaceURL() (workspaceurls.Pattern, bool)
}

// URLInitializer is implemented by exactly the resource types that declare a
// page in the workspace UI through [WorkspaceURLDeclarer].
type URLInitializer interface {
	// InitializeURL sets the URL of the resource's page. It leaves the URL empty
	// if the resource lacks the ID the URL needs, for example before its first deploy.
	InitializeURL(baseURL url.URL)

	// GetURL returns the URL set by InitializeURL.
	GetURL() string
}

// workspaceURL returns the URL of the page of the resource with the given ID,
// using the pattern that d declares.
func workspaceURL(d WorkspaceURLDeclarer, baseURL url.URL, id string) string {
	p, _ := d.WorkspaceURL()
	return p.URL(baseURL, id)
}
