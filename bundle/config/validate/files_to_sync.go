package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

func FilesToSync() bundle.ReadOnlyMutator {
	return &filesToSync{}
}

type filesToSync struct{}

func (v *filesToSync) Name() string {
	return "validate:files_to_sync"
}

func (v *filesToSync) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	// The user may be intentional about not synchronizing any files.
	// In this case, we should not show any warnings.
	if len(rb.Config().Sync.Paths) == 0 {
		return nil
	}

	sync, err := files.GetSync(ctx, rb)
	if err != nil {
		return diag.FromErr(err)
	}

	fl, err := sync.GetFileList(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// If there are files to sync, we don't need to show any warnings.
	if len(fl) != 0 {
		return nil
	}

	diags := diag.Diagnostics{}
	if len(rb.Config().Sync.Exclude) == 0 {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "There are no files to sync, please check your .gitignore",
		})
	} else {
		loc := location{path: "sync.exclude", rb: rb}
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "There are no files to sync, please check your .gitignore and sync.exclude configuration",
			// Show all locations where sync.exclude is defined, since merging
			// sync.exclude is additive.
			Locations: loc.Locations(),
			Paths:     []dyn.Path{loc.Path()},
		})
	}

	return diags
}
