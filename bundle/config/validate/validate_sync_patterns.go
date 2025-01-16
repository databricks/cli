package validate

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/fileset"
	"golang.org/x/sync/errgroup"
)

func ValidateSyncPatterns() bundle.ReadOnlyMutator {
	return &validateSyncPatterns{}
}

type validateSyncPatterns struct{}

func (v *validateSyncPatterns) Name() string {
	return "validate:validate_sync_patterns"
}

func (v *validateSyncPatterns) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	s := rb.Config().Sync
	if len(s.Exclude) == 0 && len(s.Include) == 0 {
		return nil
	}

	diags, err := checkPatterns(s.Exclude, "sync.exclude", rb)
	if err != nil {
		return diag.FromErr(err)
	}

	includeDiags, err := checkPatterns(s.Include, "sync.include", rb)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags.Extend(includeDiags)
}

func checkPatterns(patterns []string, path string, rb bundle.ReadOnlyBundle) (diag.Diagnostics, error) {
	var mu sync.Mutex
	var errs errgroup.Group
	var diags diag.Diagnostics

	for index, pattern := range patterns {
		// If the pattern is negated, strip the negation prefix
		// and check if the pattern matches any files.
		// Negation in gitignore syntax means "don't look at this path'
		// So if p matches nothing it's useless negation, but if there are matches,
		// it means: do not include these files into result set
		p := strings.TrimPrefix(pattern, "!")
		errs.Go(func() error {
			fs, err := fileset.NewGlobSet(rb.BundleRoot(), []string{p})
			if err != nil {
				return err
			}

			all, err := fs.Files()
			if err != nil {
				return err
			}

			if len(all) == 0 {
				loc := location{path: fmt.Sprintf("%s[%d]", path, index), rb: rb}
				mu.Lock()
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   fmt.Sprintf("Pattern %s does not match any files", pattern),
					Locations: []dyn.Location{loc.Location()},
					Paths:     []dyn.Path{loc.Path()},
				})
				mu.Unlock()
			}
			return nil
		})
	}

	return diags, errs.Wait()
}
