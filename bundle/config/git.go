package config

type Git struct {
	Branch    string `json:"branch,omitempty"`
	OriginURL string `json:"origin_url,omitempty"`
	Commit    string `json:"commit,omitempty" bundle:"readonly"`
}
