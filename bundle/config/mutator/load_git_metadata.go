package mutator

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/folders"
	"github.com/databricks/bricks/libs/git"
)

type loadGitMetadata struct {
	// relative path to a dir containing a bundle config file
	configPath string
}

func LoadGitMetadata(path string) *loadGitMetadata {
	return &loadGitMetadata{
		configPath: path,
	}
}

func (m *loadGitMetadata) Name() string {
	return "LoadGitMetadata"
}

func (m *loadGitMetadata) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// travel up the directory tree to find the git repository root
	repoRoot, err := folders.FindDirWithLeaf(filepath.Join(b.Config.Path, m.configPath), ".git")
	if os.IsNotExist(err) {
		// The current project is not a repository since no .git directory is found
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	gitDirPath := filepath.Join(repoRoot, ".git")
	configLoader, err := git.NewConfigLoader(gitDirPath)
	if err != nil {
		return nil, err
	}
	// load branch name if undefined
	if b.Config.Bundle.Git.Branch == "" {
		branch, err := configLoader.Branch()
		if err != nil {
			return nil, err
		}
		b.Config.Bundle.Git.Branch = branch
	}
	// load commit hash if undefined
	if b.Config.Bundle.Git.Commit == "" {
		commit, err := configLoader.Commit()
		if err != nil {
			return nil, err
		}
		b.Config.Bundle.Git.Commit = commit
	}
	// load origin url if undefined
	if b.Config.Bundle.Git.RemoteUrl == "" {
		remoteUrl, err := configLoader.HttpsOrigin()
		if !git.IsErrOriginUrlNotDefined(err) && err != nil {
			return nil, err
		}
		if !git.IsErrOriginUrlNotDefined(err) {
			b.Config.Bundle.Git.RemoteUrl = remoteUrl
		}
	}
	return nil, nil
}
