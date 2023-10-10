package config

type Git struct {
	Branch    string `json:"branch,omitempty"`
	OriginURL string `json:"origin_url,omitempty"`
	Commit    string `json:"commit,omitempty" bundle:"readonly"`

	// Path to bundle root (ie directory containing databricks.yml) relative to
	// the git repository root.
	BundleRoot string `json:"bundle_root,omitempty" bundle:"readonly"`

	// Inferred is set to true if the Git details were inferred and weren't set explicitly
	Inferred bool `json:"-" bundle:"readonly"`

	// The actual branch according to Git (may be different from the configured branch)
	ActualBranch string `json:"-" bundle:"readonly"`
}
