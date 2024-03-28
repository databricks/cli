package mutator

import (
	"context"

	semver "github.com/Masterminds/semver/v3"
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/diag"
)

func VerifyCliVersion() bundle.Mutator {
	return &verifyCliVersion{}
}

type verifyCliVersion struct {
}

func (v *verifyCliVersion) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// No constraints specified, skip the check.
	if b.Config.Bundle.DatabricksCliVersion == "" {
		return nil
	}

	constraint := b.Config.Bundle.DatabricksCliVersion
	currentVersion := build.GetInfo().Version
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return diag.Errorf("invalid version constraint %q specified. Please specify the version constraint in the format 0.0.0", constraint)
	}

	version, err := semver.NewVersion(currentVersion)
	if err != nil {
		return diag.Errorf("parsing CLI version %q failed", currentVersion)
	}

	if !c.Check(version) {
		return diag.Errorf("Databricks CLI version constraint not satisfied. Required: %s, current: %s", constraint, currentVersion)
	}

	return nil
}

func (v *verifyCliVersion) Name() string {
	return "VerifyCliVersion"
}
