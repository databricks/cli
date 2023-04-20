package config

import "github.com/databricks/bricks/libs/git"

type GitConfig struct {
	Branch    string `json:"branch"`
	RemoteUrl string `json:"remote_url"`
	Commit    string `json:"commit"`
}

// TODO: test for when git is not present
func LoadGitConfig() (*GitConfig, error) {
	originUrl, err := git.HttpsOrigin()
	if err != nil {
		return nil, err
	}
	branch := 
}
