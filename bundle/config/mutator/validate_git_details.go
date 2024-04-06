package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validateGitDetails struct{}

func ValidateGitDetails() *validateGitDetails {
	return &validateGitDetails{}
}

func (m *validateGitDetails) Name() string {
	return "ValidateGitDetails"
}

func (m *validateGitDetails) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Bundle.Git.Branch == "" || b.Config.Bundle.Git.ActualBranch == "" {
		return nil
	}

	if b.Config.Bundle.Git.Branch != b.Config.Bundle.Git.ActualBranch && !b.Config.Bundle.Force {
		return diag.Errorf("not on the right Git branch:\n  expected according to configuration: %s\n  actual: %s\nuse --force to override", b.Config.Bundle.Git.Branch, b.Config.Bundle.Git.ActualBranch)
	}
	return nil
}
