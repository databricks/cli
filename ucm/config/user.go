package config

import "github.com/databricks/databricks-sdk-go/service/iam"

// User is the populated-at-runtime current workspace user.
//
// Set by PopulateCurrentUser mutator; not declared in ucm.yml. Mirrors
// bundle/config.User minus the dyn-marshal plumbing because ucm excludes
// CurrentUser from the dyn tree (json:"-" on the Workspace field) — it's
// runtime-only state consumed by render helpers.
type User struct {
	// Short user name derived from UserName.
	ShortName string `json:"short_name,omitempty"`

	// Short user name stripped of non-alphanumeric characters. Usable as a
	// prefix for URL-bearing resources.
	DomainFriendlyName string `json:"domain_friendly_name,omitempty"`

	*iam.User
}
