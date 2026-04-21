package config

// Target defines overrides for a single deployment target (dev/staging/prod
// or any user-chosen name). Merged into Root when SelectTarget runs.
type Target struct {
	// Default marks this target as the one to select when the user does not
	// pass --target.
	Default bool `json:"default,omitempty"`

	Workspace *Workspace `json:"workspace,omitempty"`
	Account   *Account   `json:"account,omitempty"`
	Resources *Resources `json:"resources,omitempty"`
}
