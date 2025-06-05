package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/fileset"
	"golang.org/x/sync/errgroup"
)

func ValidateSyncPatterns() bundle.ReadOnlyMutator {
	return &validateSyncPatterns{}
}

type validateSyncPatterns struct{ bundle.RO }

func (v *validateSyncPatterns) Name() string {
	return "validate:validate_sync_patterns"
}

func (v *validateSyncPatterns) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	s := b.Config.Sync
	if len(s.Exclude) == 0 && len(s.Include) == 0 {
		return nil
	}

	diags, err := checkPatterns(s.Exclude, "sync.exclude", b)
	if err != nil {
		return diag.FromErr(err)
	}

	includeDiags, err := checkPatterns(s.Include, "sync.include", b)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags.Extend(includeDiags)
}

func checkPatterns(patterns []string, path string, b *bundle.Bundle) (diag.Diagnostics, error) {
	var errs errgroup.Group
	var diags diag.SafeDiagnostics

	for index, pattern := range patterns {
		// If the pattern is negated, strip the negation prefix
		// and check if the pattern matches any files.
		// Negation in gitignore syntax means "don't look at this path'
		// So if p matches nothing it's useless negation, but if there are matches,
		// it means: do not include these files into result set
		p := strings.TrimPrefix(pattern, "!")
		errs.Go(func() error {
			fs, err := fileset.NewGlobSet(b.BundleRoot, []string{p})
			if err != nil {
				return err
			}

			all, err := fs.Files()
			if err != nil {
				return err
			}

			if len(all) == 0 {
				path := fmt.Sprintf("%s[%d]", path, index)
				diags.Append(diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   fmt.Sprintf("Pattern %s does not match any files", pattern),
					Locations: b.Config.GetLocations(path),
					Paths:     []dyn.Path{dyn.MustPathFromString(path)},
				})
			}
			return nil
		})
	}

	return diags.Diags, errs.Wait()
}
