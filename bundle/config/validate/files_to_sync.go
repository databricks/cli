package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/libs/diag"
)

func FilesToSync() bundle.Mutator {
	return &filesToSync{}
}

type filesToSync struct {
}

func (v *filesToSync) Name() string {
	return "validate:files_to_sync"
}

func (v *filesToSync) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	sync, err := files.GetSync(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := diag.Diagnostics{}
	fl, err := sync.GetFileList(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(fl) == 0 {
		if len(b.Config.Sync.Exclude) == 0 {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "There are no files to sync, please check your .gitignore",
			})
		} else {
			loc := location{path: "sync.exclude", b: b}
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "There are no files to sync, please check your .gitignore and sync.exclude configuration",
				Location: loc.Location(),
				Path:     loc.Path(),
			})
		}
	}

	return diags
}
