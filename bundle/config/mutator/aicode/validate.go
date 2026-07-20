package aicode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// Validate surfaces local code_source_path misconfigurations at `bundle validate`
// time rather than mid-deploy. It performs no uploads.
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
	// Only local values are packaged; remote ones are used as-is.
	if codeSourcePath == "" || !libraries.IsLocalPath(codeSourcePath) {
		return nil
	}

	locations := b.Config.GetLocations(codePath.String())

	// git_source retrieves task files from git, so a local directory would be ignored.
	if jobHasGitSource {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "ai_runtime_task with a local code_source_path cannot be combined with git_source",
			Detail:    "Remove git_source or set code_source_path to a workspace or volume path",
			Locations: locations,
			Paths:     []dyn.Path{codePath},
		}}
	}

	// Immutable-folder uploads one snapshot and doesn't support per-task packaging.
	if b.IsImmutableFolder() {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   "ai_runtime_task with a local code_source_path is not supported with experimental.immutable_folder",
			Locations: locations,
			Paths:     []dyn.Path{codePath},
		}}
	}

	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(codeSourcePath))
	info, err := os.Stat(localDir)
	if err != nil {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("code_source_path %q not found", codeSourcePath),
			Locations: locations,
			Paths:     []dyn.Path{codePath},
		}}
	}
	if !info.IsDir() {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("code_source_path %q must be a directory", codeSourcePath),
			Locations: locations,
			Paths:     []dyn.Path{codePath},
		}}
	}

	return nil
}
