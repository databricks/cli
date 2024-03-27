package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/diag"
	"golang.org/x/mod/semver"
)

func VerifyCliVersion() bundle.Mutator {
	return &verifyCliVersion{}
}

type verifyCliVersion struct {
}

func (v *verifyCliVersion) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	currentVersion := "v" + build.GetInfo().Version
	if b.Config.Bundle.MinDatabricksCliVersion != "" {
		v := "v" + b.Config.Bundle.MinDatabricksCliVersion
		if err := validateVersion(v); err != nil {
			return diag.FromErr(err)
		}

		if isLowerVersion(currentVersion, v) {
			return diag.Errorf("minimum Databricks CLI version required: %s, current version: %s", v, currentVersion)
		}
	}

	if b.Config.Bundle.MaxDatabricksCliVersion != "" {
		v := "v" + b.Config.Bundle.MaxDatabricksCliVersion
		if err := validateVersion(v); err != nil {
			return diag.FromErr(err)
		}

		if isLowerVersion(v, currentVersion) {
			return diag.Errorf("maximum Databricks CLI version required: %s, current version: %s", v, currentVersion)
		}
	}
	return nil
}

func (v *verifyCliVersion) Name() string {
	return "VerifyCliVersion"
}

func isLowerVersion(v1, v2 string) bool {
	return semver.Compare(v1, v2) < 0
}

func validateVersion(v string) error {
	if !semver.IsValid(v) {
		return fmt.Errorf("invalid version %q specified. Please specify the version in the format 0.0.0", v)
	}
	return nil
}
