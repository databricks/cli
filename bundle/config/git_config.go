package config

import "github.com/databricks/bricks/libs/git"

type GitConfig struct {
	Branch    string `json:"branch"`
	RemoteUrl string `json:"remote_url"`
	Commit    string `json:"commit"`
}

func LoadGitConfig(path string) (*GitConfig, error) {
	l, err := git.NewConfigLoader(path)

	// return early if the project is not a repository
	if err == git.ErrNotARepository {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// read the remote "origin" url. The user might not have origin set and instead
	// use a different name for their remote repository, in which case we load
	// an empty string for the origin url
	originUrl, err := l.HttpsOrigin()
	if err != nil && !git.IsErrOriginUrlNotDefined(err) {
		return nil, err
	}
	// read current selected branch in repo
	branch, err := l.Branch()
	if err != nil {
		return nil, err
	}
	// read the current commit's hash
	// TODO: see what happens when no commits in repo
	commit, err := l.Commit()
	if err != nil {
		return nil, err
	}

	return &GitConfig{
		Branch:    branch,
		RemoteUrl: originUrl,
		Commit:    commit,
	}, nil
}
