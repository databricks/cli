package config

type Git struct {
	Branch    string `json:"branch,omitempty"`
	RemoteURL string `json:"remote_url,omitempty"`
	Commit    string `json:"commit,omitempty" bundle:"readonly"`
}
