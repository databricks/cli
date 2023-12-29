package mutator

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
)

type loadGitDetails struct{}

func LoadGitDetails() *loadGitDetails {
	return &loadGitDetails{}
}

func (m *loadGitDetails) Name() string {
	return "LoadGitDetails"
}

func (m *loadGitDetails) Apply(ctx context.Context, b *bundle.Bundle) error {
	// Load relevant git repository
	repo, err := git.NewRepository(b.Config.Path)
	if err != nil {
		return err
	}

	// Read branch name of current checkout
	branch, err := repo.CurrentBranch()
	if err == nil {
		b.Config.Bundle.Git.ActualBranch = branch
		if b.Config.Bundle.Git.Branch == "" {
			// Only load branch if there's no user defined value
			b.Config.Bundle.Git.Inferred = true
			b.Config.Bundle.Git.Branch = branch
		}
	} else {
		log.Warnf(ctx, "failed to load current branch: %s", err)
	}

	// load commit hash if undefined
	if b.Config.Bundle.Git.Commit == "" {
		commit, err := repo.LatestCommit()
		if err != nil {
			log.Warnf(ctx, "failed to load latest commit: %s", err)
		} else {
			b.Config.Bundle.Git.Commit = commit
		}
	}
	// load origin url if undefined
	if b.Config.Bundle.Git.OriginURL == "" {
		remoteUrl := repo.OriginUrl()
		b.Config.Bundle.Git.OriginURL = remoteUrl
	}

	// Compute relative path of the bundle root from the Git repo root.
	absBundlePath, err := filepath.Abs(b.Config.Path)
	if err != nil {
		return err
	}
	// repo.Root() returns the absolute path of the repo
	relBundlePath, err := filepath.Rel(repo.Root(), absBundlePath)
	if err != nil {
		return err
	}
	b.Config.Bundle.Git.BundleRootPath = filepath.ToSlash(relBundlePath)
	return nil
}
