package mutator

import (
	"context"
	"fmt"
	"regexp"

	semver "github.com/Masterminds/semver/v3"
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/diag"
)

func VerifyCliVersion() bundle.Mutator {
	return &verifyCliVersion{}
}

type verifyCliVersion struct{}

func (v *verifyCliVersion) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// No constraints specified, skip the check.
	if b.Config.Bundle.DatabricksCliVersion == "" {
		return nil
	}

	constraint := b.Config.Bundle.DatabricksCliVersion
	if err := validateConstraintSyntax(constraint); err != nil {
		return diag.FromErr(err)
	}
	currentVersion := build.GetInfo().Version
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return diag.FromErr(err)
	}

	version, err := semver.NewVersion(currentVersion)
	if err != nil {
		return diag.Errorf("parsing CLI version %q failed", currentVersion)
	}

	if !c.Check(version) {
		if version.Prerelease() == "dev" && version.Major() == 0 {
			return diag.Warningf("Ignoring Databricks CLI version constraint for development build. Required: %s, current: %s", constraint, currentVersion)
		}

		return diag.Errorf("Databricks CLI version constraint not satisfied. Required: %s, current: %s", constraint, currentVersion)
	}

	return nil
}

func (v *verifyCliVersion) Name() string {
	return "VerifyCliVersion"
}

// validateConstraintSyntax validates the syntax of the version constraint.
func validateConstraintSyntax(constraint string) error {
	r := generateConstraintSyntaxRegexp()
	if !r.MatchString(constraint) {
		return fmt.Errorf("invalid version constraint %q specified. Please specify the version constraint in the format (>=) 0.0.0(, <= 1.0.0)", constraint)
	}

	return nil
}

// Generate regexp which matches the supported version constraint syntax.
func generateConstraintSyntaxRegexp() *regexp.Regexp {
	// We intentionally only support the format supported by requirements.txt:
	// 1. 0.0.0
	// 2. >= 0.0.0
	// 3. <= 0.0.0
	// 4. > 0.0.0
	// 5. < 0.0.0
	// 6. != 0.0.0
	// 7. 0.0.*
	// 8. 0.*
	// 9. >= 0.0.0, <= 1.0.0
	// 10. 0.0.0-0
	// 11. 0.0.0-beta
	// 12. >= 0.0.0-0, <= 1.0.0-0

	matchVersion := `(\d+\.\d+\.\d+(\-\w+)?|\d+\.\d+.\*|\d+\.\*)`
	matchOperators := `(>=|<=|>|<|!=)?`
	return regexp.MustCompile(fmt.Sprintf(`^%s ?%s(, %s %s)?$`, matchOperators, matchVersion, matchOperators, matchVersion))
}
