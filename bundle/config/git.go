package config

type Git struct {
	Branch    string `json:"branch"`
	RemoteUrl string `json:"remote_url"`
	Commit    string `json:"commit"`
}
