package mutator

import (
	"context"
	"fmt"
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
	var diags diag.Diagnostics
	info, err := git.FetchRepositoryInfo(ctx, b.BundleRoot.Native(), b.WorkspaceClient())
	if err != nil {
		diags = append(diags, diag.WarningFromErr(err)...)
	}

	b.WorktreeRoot = info.GuessedWorktreeRoot
	config := &b.Config.Bundle.Git

	config.ActualBranch = info.CurrentBranch
	if config.Branch == "" && info.CurrentBranch != "" {
		config.Inferred = true
	}

	diags = checkMatch(diags, "Git branch", info.CurrentBranch, &config.Branch, b.Config.Bundle.Force)
	diags = checkMatch(diags, "Git commit", info.LatestCommit, &config.Commit, b.Config.Bundle.Force)
	diags = checkMatch(diags, "Git originURL", info.OriginURL, &config.OriginURL, true)

	relBundlePath, err := filepath.Rel(b.WorktreeRoot.Native(), b.BundleRoot.Native())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	config.BundleRootPath = filepath.ToSlash(relBundlePath)
	return diags
}

func checkMatch(diags []diag.Diagnostic, field, fetchedValue string, configValue *string, allowedToMismatch bool) []diag.Diagnostic {
	if fetchedValue == "" {
		return diags
	}

	// The value from config takes precedence; however, we always warn if configValue and fetchedValue disagree.
	// In case of branch and commit and absence of --force we raise severity to Error

	if *configValue == "" {
		*configValue = fetchedValue
		return diags
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

		return append(diags, diag.Diagnostic{
			Severity: severity,
			Summary:  fmt.Sprintf(tmpl, field, *configValue, fetchedValue, extra),
		})
	}

	return diags
}
