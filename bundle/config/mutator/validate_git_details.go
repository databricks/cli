package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
)

type validateGitDetails struct{}

func ValidateGitDetails() *validateGitDetails {
	return &validateGitDetails{}
}

func (m *validateGitDetails) Name() string {
	return "ValidateGitDetails"
}

func (m *validateGitDetails) Apply(ctx context.Context, b *bundle.Bundle) error {
	if b.Config.Bundle.Git.Branch == "" || b.Config.Bundle.Git.ActualBranch == "" {
		return nil
	}

	if b.Config.Bundle.Git.Branch != b.Config.Bundle.Git.ActualBranch && !b.Config.Bundle.Force {
		return fmt.Errorf("not on the right Git branch:\n  expected according to configuration: %s\n  actual: %s\nuse --force to override", b.Config.Bundle.Git.Branch, b.Config.Bundle.Git.ActualBranch)
	}
	return nil
}
