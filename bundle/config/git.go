package config

type Git struct {
	Branch    string `json:"branch,omitempty" bundle:"metadata"`
	OriginURL string `json:"origin_url,omitempty" bundle:"metadata"`
	Commit    string `json:"commit,omitempty" bundle:"readonly,metadata"`

	// Inferred is set to true if the Git details were inferred and weren't set explicitly
	Inferred bool `json:"-" bundle:"readonly"`

	// The actual branch according to Git (may be different from the configured branch)
	ActualBranch string `json:"-" bundle:"readonly"`
}
