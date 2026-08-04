package aicode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	ignore "github.com/sabhiram/go-gitignore"
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

	// packagesCode records whether any task actually has a code_source_path this
	// mutator will package. The bundle-level guards below (which reject configs that
	// would drop the generated snapshot from sync) only matter in that case, so
	// they're gated on it to avoid spurious errors on unrelated bundles.
	packagesCode := false

	for name, job := range b.Config.Resources.Jobs {
		jobPath := jobsPath.Append(dyn.Key(name))

		for i, task := range job.Tasks {
			taskPath := jobPath.Append(dyn.Key("tasks"), dyn.Index(i))
			if task.AiRuntimeTask != nil && v.packagesLocalDir(b, task.AiRuntimeTask.CodeSourcePath) {
				packagesCode = true
			}

			// A local code_source_path under a for_each_task is not packaged by this
			// mutator (aicode collects only direct tasks). Reject it rather than let a
			// nested ai_runtime_task deploy an un-packaged local path.
			if task.ForEachTask != nil && task.ForEachTask.Task.AiRuntimeTask != nil {
				nestedCode := task.ForEachTask.Task.AiRuntimeTask.CodeSourcePath
				if nestedCode != "" && libraries.IsLocalPath(nestedCode) {
					p := taskPath.Append(dyn.Key("for_each_task"), dyn.Key("task"),
						dyn.Key("ai_runtime_task"), dyn.Key("code_source_path"))
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "ai_runtime_task with a local code_source_path is not supported inside a for_each_task",
						Detail:    "Set code_source_path to a workspace or volume path",
						Locations: b.Config.GetLocations(p.String()),
						Paths:     []dyn.Path{p},
					})
				}
			}

			if task.AiRuntimeTask == nil {
				continue
			}
			codePath := taskPath.Append(dyn.Key("ai_runtime_task"), dyn.Key("code_source_path"))
			diags = diags.Extend(v.validateTask(b, job.GitSource, task.AiRuntimeTask.CodeSourcePath, codePath))
		}
	}

	if packagesCode {
		diags = diags.Extend(validateSnapshotDir(b))
	}

	return diags
}

// packagesLocalDir reports whether codeSourcePath is one this mutator packages: a
// local path that is an existing directory. A local *file* (a pre-built tarball from
// an `artifacts` block) is uploaded by the artifact path instead, so it must not
// trigger the snapshot-directory guards — gating those on IsLocalPath alone rejects
// valid pre-built-tarball bundles that exclude "*.tar.gz" from sync.
func (v *validate) packagesLocalDir(b *bundle.Bundle, codeSourcePath string) bool {
	if codeSourcePath == "" || !libraries.IsLocalPath(codeSourcePath) {
		return false
	}
	// A stat error is reported by validateTask; treat it as "not a directory" here.
	isDir, err := isExistingDir(filepath.Join(b.SyncRootPath, filepath.FromSlash(codeSourcePath)))
	return err == nil && isDir
}

// validateSnapshotDir rejects two configs that would silently drop the generated
// code archive from sync (leaving the job pointing at an un-uploaded path):
//
//   - A real file/dir at bundle.AiCodeSnapshotDir collides with the overlay (sync
//     carries the user's entry, not the archive).
//   - A sync.exclude matching that path removes the archive (exclude is applied
//     after include, so it beats the force-include).
func validateSnapshotDir(b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	syncExcludePath := dyn.NewPath(dyn.Key("sync"), dyn.Key("exclude"))

	// A user-owned file or directory at the reserved path collides with the overlay.
	local := filepath.Join(b.SyncRootPath, bundle.AiCodeSnapshotDir)
	if _, err := os.Stat(local); err == nil {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("%q is reserved for AI Runtime code snapshots and must not exist in the bundle", bundle.AiCodeSnapshotDir),
			Detail:   "Remove it; the deploy generates code archives under this path.",
		})
	}

	// A sync.exclude matching the reserved path would drop the generated archive from
	// the upload (exclude wins over the force-include).
	if matchesSnapshotDir(b.Config.Sync.Exclude) {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("sync.exclude must not match %q, which holds the AI Runtime code snapshot", bundle.AiCodeSnapshotDir),
			Detail:    "Remove the pattern that excludes it; otherwise the deployed job's code_source_path would not be uploaded.",
			Locations: b.Config.GetLocations(syncExcludePath.String()),
			Paths:     []dyn.Path{syncExcludePath},
		})
	}

	return diags
}

// matchesSnapshotDir reports whether any sync.exclude pattern would remove a file
// under the reserved snapshot directory, using the same gitignore-style matcher the
// sync engine applies to exclude patterns (see libs/fileset).
func matchesSnapshotDir(exclude []string) bool {
	if len(exclude) == 0 {
		return false
	}
	matcher := ignore.CompileIgnoreLines(exclude...)
	// A representative archive path; the mutator names archives
	// <AiCodeSnapshotDir>/<dir>_<sha>.tar.gz.
	return matcher.MatchesPath(bundle.AiCodeSnapshotDir + "/probe.tar.gz")
}

func (v *validate) validateTask(b *bundle.Bundle, gitSource *jobs.GitSource, codeSourcePath string, codePath dyn.Path) diag.Diagnostics {
	// Only local code_source_path values are packaged at deploy; remote values
	// are used as-is and need no validation here.
	if codeSourcePath == "" || !libraries.IsLocalPath(codeSourcePath) {
		return nil
	}

	locations := b.Config.GetLocations(codePath.String())
	reject := func(summary, detail string) diag.Diagnostics {
		return diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   summary,
			Detail:    detail,
			Locations: locations,
			Paths:     []dyn.Path{codePath},
		}}
	}

	// The packaged directory must live inside the bundle sync root: it is uploaded as
	// part of the bundle, so a path escaping the root (e.g. "../shared") can't be
	// synced. Reject it here with a clear message rather than letting it fail later as
	// an opaque io/fs "invalid argument" when the file list is built.
	if rel, err := filepath.Rel(b.SyncRootPath, filepath.Join(b.SyncRootPath, filepath.FromSlash(codeSourcePath))); err != nil || !filepath.IsLocal(rel) {
		return reject(
			fmt.Sprintf("code_source_path %q is outside the bundle root", codeSourcePath),
			"code_source_path must point at a directory inside the bundle, or at a workspace or volume path",
		)
	}

	// This mutator packages a local *directory*. A local path that is not an existing
	// directory is left alone so it flows through the standard artifact-upload path:
	// a pre-built tarball delivered via an `artifacts` block is produced during the
	// build phase (so it does not exist yet at validate time) and is uploaded as a
	// file, not packaged here. Only when the path is an existing directory do the
	// packaging-specific constraints below apply. A stat failure other than not-exist
	// (e.g. unreadable parent) is surfaced rather than silently skipped.
	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(codeSourcePath))
	isDir, err := isExistingDir(localDir)
	if err != nil {
		return reject(fmt.Sprintf("failed to inspect code_source_path %q: %v", codeSourcePath, err), "")
	}
	if !isDir {
		return nil
	}

	// The deploy engine retrieves task files from git when git_source is set, so
	// packaging a local directory would be silently ignored. Reject the combination.
	if gitSource != nil {
		return reject(
			"ai_runtime_task with a local code_source_path cannot be combined with git_source",
			"Remove git_source or set code_source_path to a workspace or volume path",
		)
	}

	// Immutable-folder deployments upload a single content-addressed snapshot and
	// do not support the per-task code packaging this mutator performs.
	if b.IsImmutableFolder() {
		return reject("ai_runtime_task with a local code_source_path is not supported with experimental.immutable_folder", "")
	}

	// Source-linked deployment runs jobs against the source files in place and does
	// not copy them to the workspace file path (files.Upload is a no-op). This
	// mutator relies on file sync uploading the packaged snapshot, so the two are
	// incompatible: reject rather than deploy a job pointing at an un-uploaded path.
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		return reject(
			"ai_runtime_task with a local code_source_path is not supported with source-linked deployment",
			"Disable source-linked deployment, or set code_source_path to a workspace or volume path",
		)
	}

	return nil
}
