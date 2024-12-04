package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/vfs"
)

type loadGitDetails struct{}

func LoadGitDetails() *loadGitDetails {
	return &loadGitDetails{}
}

func (m *loadGitDetails) Name() string {
	return "LoadGitDetails"
}

func (m *loadGitDetails) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	info, err := git.FetchRepositoryInfo(ctx, b.BundleRoot.Native(), b.WorkspaceClient())
	if err != nil {
		diags = append(diags, diag.WarningFromErr(err)...)
	}

	if info.WorktreeRoot == "" {
		b.WorktreeRoot = b.BundleRoot
	} else {
		b.WorktreeRoot = vfs.MustNew(info.WorktreeRoot)
	}

	config := &b.Config.Bundle.Git

	config.ActualBranch = info.CurrentBranch
	if config.Branch == "" && info.CurrentBranch != "" {
		config.Inferred = true
	}

	// The value from config takes precedence; however, we always warn if configValue and fetchedValue disagree.
	// In case of branch and commit and absence of --force we raise severity to Error
	diags = append(diags, checkMatch("Git branch", info.CurrentBranch, &config.Branch, b.Config.Bundle.Force)...)
	diags = append(diags, checkMatch("Git commit", info.LatestCommit, &config.Commit, b.Config.Bundle.Force)...)
	diags = append(diags, checkMatch("Git originURL", info.OriginURL, &config.OriginURL, true)...)

	relBundlePath, err := filepath.Rel(b.WorktreeRoot.Native(), b.BundleRoot.Native())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	} else {
		b.Config.Bundle.Git.BundleRootPath = filepath.ToSlash(relBundlePath)
	}

	config.BundleRootPath = filepath.ToSlash(relBundlePath)
	return diags
}

func checkMatch(field, fetchedValue string, configValue *string, allowedToMismatch bool) []diag.Diagnostic {
	if fetchedValue == "" {
		return nil
	}

	if *configValue == "" {
		*configValue = fetchedValue
		return nil
	}

	if *configValue != fetchedValue {
		tmpl := "not on the right %s:\n  expected according to configuration: %s\n  actual: %s%s"
		extra := ""
		var severity diag.Severity
		if allowedToMismatch {
			severity = diag.Warning
		} else {
			severity = diag.Error
			extra = "\nuse --force to override"
		}

		return []diag.Diagnostic{{
			Severity: severity,
			Summary:  fmt.Sprintf(tmpl, field, *configValue, fetchedValue, extra),
		}}
	}

	return nil
}
