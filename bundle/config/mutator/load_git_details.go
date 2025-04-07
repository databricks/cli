package mutator

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/vfs"
)

type loadGitDetails struct{}

func LoadGitDetails() bundle.Mutator {
	return &loadGitDetails{}
}

func (m *loadGitDetails) Name() string {
	return "LoadGitDetails"
}

func (m *loadGitDetails) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	info, err := git.FetchRepositoryInfo(ctx, b.BundleRoot.Native(), b.WorkspaceClient())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			diags = append(diags, diag.WarningFromErr(err)...)
		}
	}

	if info.WorktreeRoot == "" {
		b.WorktreeRoot = b.SyncRoot
	} else {
		b.WorktreeRoot = vfs.MustNew(info.WorktreeRoot)
	}

	b.Config.Bundle.Git.ActualBranch = info.CurrentBranch
	if b.Config.Bundle.Git.Branch == "" {
		// Only load branch if there's no user defined value
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

	relBundlePath, err := filepath.Rel(b.WorktreeRoot.Native(), b.BundleRoot.Native())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	} else {
		b.Config.Bundle.Git.BundleRootPath = filepath.ToSlash(relBundlePath)
	}
	return diags
}
