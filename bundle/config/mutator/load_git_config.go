package mutator

import (
	"context"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/git"
)

type loadGitConfig struct{}

func LoadGitConfig() *loadGitConfig {
	return &loadGitConfig{}
}

func (m *loadGitConfig) Name() string {
	return "LoadGitConfig"
}

func (m *loadGitConfig) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// Load relevant git repository
	repo, err := git.NewRepository(b.Config.Path)
	if err != nil {
		return nil, err
	}
	// load branch name if undefined
	if b.Config.Bundle.Git.Branch == "" {
		branch, err := repo.CurrentBranch()
		if err != nil {
			return nil, err
		}
		b.Config.Bundle.Git.Branch = branch
	}
	// load commit hash if undefined
	if b.Config.Bundle.Git.Commit == "" {
		commit, err := repo.LatestCommit()
		if err != nil {
			return nil, err
		}
		b.Config.Bundle.Git.Commit = commit
	}
	// load origin url if undefined
	if b.Config.Bundle.Git.RemoteURL == "" {
		remoteUrl, err := repo.OriginUrl()
		if err != nil {
			return nil, err
		}
		b.Config.Bundle.Git.RemoteURL = remoteUrl
	}
	return nil, nil
}
