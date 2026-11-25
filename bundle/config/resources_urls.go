package config

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/workspaceurls"
)

// ResourceURL returns the workspace URL of r and whether r's type has a page in
// the workspace UI at all. A resource of a type with a page that is not deployed
// yet returns ("", true); a resource of a type without a page returns ("", false).
func ResourceURL(r ConfigResource) (string, bool) {
	u, ok := r.(resources.URLInitializer)
	if !ok {
		return "", false
	}
	return u.GetURL(), true
}

// WorkspaceURLPatterns returns, by resource key, the URL pattern of every
// resource type whose page in the workspace UI is addressed by a single ID.
func WorkspaceURLPatterns() map[string]workspaceurls.Pattern {
	patterns := map[string]workspaceurls.Pattern{}
	for key, d := range ResourceTypesImplementing[resources.WorkspaceURLDeclarer]() {
		if p, ok := d.WorkspaceURL(); ok && p.Path != "" {
			patterns[key] = p
		}
	}
	return patterns
}
