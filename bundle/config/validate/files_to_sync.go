package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/sync"
)

func FilesToSync() bundle.ReadOnlyMutator {
	return &filesToSync{}
}

type filesToSync struct{ bundle.RO }

func (v *filesToSync) Name() string {
	return "validate:files_to_sync"
}

func (v *filesToSync) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// The user may be intentional about not synchronizing any files.
	// In this case, we should not show any warnings.
	if len(b.Config.Sync.Paths) == 0 {
		return nil
	}

	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Validation must not mutate the workspace; a dry-run sync skips creating
	// a missing remote file path.
	opts.DryRun = true

	s, err := sync.New(ctx, *opts)
	if err != nil {
		return diag.FromErr(err)
	}

	fl, err := s.GetFileList(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// If there are files to sync, we don't need to show any warnings.
	if len(fl) != 0 {
		return nil
	}

	diags := diag.Diagnostics{}
	if len(b.Config.Sync.Exclude) == 0 {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "There are no files to sync, please check your .gitignore",
		})
	} else {
		path := "sync.exclude"
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "There are no files to sync, please check your .gitignore and sync.exclude configuration",
			// Show all locations where sync.exclude is defined, since merging
			// sync.exclude is additive.
			Locations: b.Config.GetLocations(path),
			Paths:     []dyn.Path{dyn.MustPathFromString(path)},
		})
	}

	return diags
}
