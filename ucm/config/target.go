package config

import "github.com/databricks/cli/ucm/config/variable"

// Target defines overrides for a single deployment target (dev/staging/prod
// or any user-chosen name). Merged into Root when SelectTarget runs.
type Target struct {
	// Default marks this target as the one to select when the user does not
	// pass --target.
	Default bool `json:"default,omitempty"`

	Workspace *Workspace `json:"workspace,omitempty"`
	Account   *Account   `json:"account,omitempty"`
	Resources *Resources `json:"resources,omitempty"`

	// Variables per-target override. Values replace (not deep-merge) the
	// matching root-level variable's default/lookup.
	Variables map[string]*variable.TargetVariable `json:"variables,omitempty"`
}
