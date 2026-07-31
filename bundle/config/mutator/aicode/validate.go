package aicode

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// Validate checks AI Runtime tasks that reference a local code_source_path so
// that misconfigurations surface at `bundle validate` time with an actionable
// message, rather than as an obscure failure mid-deploy. It performs no uploads.
func Validate() bundle.ReadOnlyMutator {
	return &validate{}
}

type validate struct{ bundle.RO }

func (v *validate) Name() string {
	return "aicode.Validate"
}

func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	jobsPath := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"))

	for name, job := range b.Config.Resources.Jobs {
		jobPath := jobsPath.Append(dyn.Key(name))

		for i, task := range job.Tasks {
			if task.AiRuntimeTask == nil {
				continue
			}
			codePath := jobPath.Append(dyn.Key("tasks"), dyn.Index(i),
				dyn.Key("ai_runtime_task"), dyn.Key("code_source_path"))
			diags = diags.Extend(v.validateTask(b, job.GitSource != nil, task.AiRuntimeTask.CodeSourcePath, codePath))
		}
	}

	return diags
}

func (v *validate) validateTask(b *bundle.Bundle, jobHasGitSource bool, codeSourcePath string, codePath dyn.Path) diag.Diagnostics {
	// Only local code_source_path values are packaged at deploy; remote values
	// are used as-is and need no validation here.
	if codeSourcePath == "" || !libraries.IsLocalPath(codeSourcePath) {
		return nil
	}

	// This mutator packages a local *directory*. A local path that is not an existing
	// directory is left alone so it flows through the standard artifact-upload path:
	// a pre-built tarball delivered via an `artifacts` block is produced during the
	// build phase (so it does not exist yet at validate time) and is uploaded as a
	// file, not packaged here. Only when the path is an existing directory do the
	// packaging-specific constraints below apply.
	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(codeSourcePath))
	info, statErr := os.Stat(localDir)
	if statErr != nil || !info.IsDir() {
		return nil
	}

	locations := b.Config.GetLocations(codePath.String())

	// The deploy engine retrieves task files from git when git_source is set, so
	// packaging a local directory would be silently ignored. Reject the combination.
	if jobHasGitSource {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "ai_runtime_task with a local code_source_path cannot be combined with git_source",
			Detail:    "Remove git_source or set code_source_path to a workspace or volume path",
			Locations: locations,
			Paths:     []dyn.Path{codePath},
		}}
	}

	// Immutable-folder deployments upload a single content-addressed snapshot and
	// do not support the per-task code packaging this mutator performs.
	if b.IsImmutableFolder() {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "ai_runtime_task with a local code_source_path is not supported with experimental.immutable_folder",
			Locations: locations,
			Paths:     []dyn.Path{codePath},
		}}
	}

	return nil
}
