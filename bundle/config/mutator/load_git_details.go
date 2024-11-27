package mutator

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/git"
)

type loadGitDetails struct{}

func LoadGitDetails() *loadGitDetails {
	return &loadGitDetails{}
}

func (m *loadGitDetails) Name() string {
	return "LoadGitDetails"
}

func (m *loadGitDetails) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	info, err := git.FetchRepositoryInfo(ctx, b.BundleRoot, b.WorkspaceClient())
	if err != nil {
		return diag.FromErr(err)
	}

	b.WorktreeRoot = info.WorktreeRoot

	b.Config.Bundle.Git.ActualBranch = info.CurrentBranch
	if b.Config.Bundle.Git.Branch == "" {
		// Only load branch if there's no user defined value
		b.Config.Bundle.Git.Inferred = true
		b.Config.Bundle.Git.Branch = info.CurrentBranch
	}

	// load commit hash if undefined
	if b.Config.Bundle.Git.Commit == "" {
		b.Config.Bundle.Git.Commit = info.LatestCommit
	}

	// load origin url if undefined
	if b.Config.Bundle.Git.OriginURL == "" {
		b.Config.Bundle.Git.OriginURL = info.OriginURL
	}

	relBundlePath, err := filepath.Rel(info.WorktreeRoot.Native(), b.BundleRoot.Native())
	if err != nil {
		return diag.FromErr(err)
	}

	b.Config.Bundle.Git.BundleRootPath = filepath.ToSlash(relBundlePath)
	return nil
}
