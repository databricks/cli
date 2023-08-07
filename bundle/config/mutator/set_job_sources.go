package mutator

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type setJobSources struct{}

func SetJobSources() bundle.Mutator {
	return &setJobSources{}
}

func (m *setJobSources) Name() string {
	return "SetJobSources"
}

func (m *setJobSources) Apply(ctx context.Context, b *bundle.Bundle) error {
	repo, err := git.NewRepository(b.Config.Path)
	if err != nil {
		return err
	}
	if !repo.IsRealRepo() {
		return nil
	}

	for _, job := range b.Config.Resources.Jobs {
		branch := ""
		commit := ""

		if b.Config.Bundle.Git.Branch != "" {
			// Set branch, If current checkout is a branch.
			branch = b.Config.Bundle.Git.Branch
		} else {
			// Set the commit SHA if current checkout is not a branch.
			commit = b.Config.Bundle.Git.Commit
		}

		relPath, err := filepath.Rel(repo.Root(), job.ConfigFilePath)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		job.GitSource = &jobs.GitSource{
			GitBranch: branch,
			GitCommit: commit,
			GitUrl:    b.Config.Bundle.Git.OriginURL,
			JobSource: &jobs.JobSource{
				ImportFromGitBranch: branch,

				// Set job source config path, i.e the path to yml file containing
				// the job definition.
				JobConfigPath: relPath,
			},
		}
	}

	return nil
}
