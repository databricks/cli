package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/fileset"
)

func ValidateSyncPatterns() bundle.ReadOnlyMutator {
	return &validateSyncPatterns{}
}

type validateSyncPatterns struct {
}

func (v *validateSyncPatterns) Name() string {
	return "validate:validate_sync_patterns"
}

func (v *validateSyncPatterns) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	sync := rb.Config().Sync()
	if len(sync.Exclude) == 0 && len(sync.Include) == 0 {
		return nil
	}

	diags := diag.Diagnostics{}
	if len(sync.Exclude) != 0 {
		for i, exclude := range sync.Exclude {
			fs, err := fileset.NewGlobSet(rb.RootPath(), []string{exclude})
			if err != nil {
				return diag.FromErr(err)
			}

			all, err := fs.All()
			if err != nil {
				return diag.FromErr(err)
			}

			if len(all) == 0 {
				loc := location{path: fmt.Sprintf("sync.exclude[%d]", i), rb: rb}
				diags = diags.Append(diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("Exclude pattern %s does not match any files", exclude),
					Location: loc.Location(),
					Path:     loc.Path(),
				})
			}
		}
	}

	if len(sync.Include) != 0 {
		for i, include := range sync.Include {
			fs, err := fileset.NewGlobSet(rb.RootPath(), []string{include})
			if err != nil {
				return diag.FromErr(err)
			}

			all, err := fs.All()
			if err != nil {
				return diag.FromErr(err)
			}

			if len(all) == 0 {
				loc := location{path: fmt.Sprintf("sync.include[%d]", i), rb: rb}
				diags = diags.Append(diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("Include pattern %s does not match any files", include),
					Location: loc.Location(),
					Path:     loc.Path(),
				})
			}
		}
	}

	return diags
}
