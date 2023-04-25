package config

type Git struct {
	Branch    string `json:"branch"`
	RemoteURL string `json:"remote_url"`
	Commit    string `json:"commit"`
}
