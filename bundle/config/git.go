package config

type Git struct {
	Branch    string `json:"branch,omitempty"`
	OriginURL string `json:"origin_url,omitempty"`
	Commit    string `json:"commit,omitempty" bundle:"readonly"`

	// Path to bundle root relative to the git repository root.
	BundleRootPath string `json:"bundle_root_path,omitempty" bundle:"readonly"`

	// Inferred is set to true if the Git details were inferred and weren't set explicitly
	Inferred bool `json:"inferred" bundle:"readonly"`

	// The actual branch according to Git (may be different from the configured branch)
	ActualBranch string `json:"actual_branch" bundle:"readonly"`
}
