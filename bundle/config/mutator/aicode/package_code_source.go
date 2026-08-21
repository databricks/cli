// Package aicode packages a local directory referenced by an AI Runtime task's
// code_source_path into a content-addressed tarball inside the bundle, and rewrites
// code_source_path to the workspace path that tarball occupies once synced. Remote
// values are left untouched.
//
// The archive is overlaid on the sync tree and uploaded by normal bundle file sync
// in the deploy phase; the mutator performs no workspace writes, so it is safe in
// the build phase (which runs before `bundle plan`). Living in the bundle means
// `bundle destroy` cleans it, and the content-addressed name lets incremental sync
// skip re-uploading unchanged code.
//
// Not done via mutator.TranslatePaths (which handles command_path): that runs in
// initialize, which also runs on `bundle validate`, so the archive would be
// materialized during validate. Build phase is deploy-only.
package aicode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/log"
	libsync "github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/vfs"
)

// codeSourcePattern is the config location of an AI Runtime task's
// code_source_path. It matches a direct task only — the same scope aicode.Validate
// operates on. ai_runtime_task nested under a for_each_task is not a supported
// combination yet (Validate rejects it); when it is, both should gain it together.
var codeSourcePattern = dyn.NewPattern(
	dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(),
	dyn.Key("tasks"), dyn.AnyIndex(),
	dyn.Key("ai_runtime_task"), dyn.Key("code_source_path"),
)

// codeSource is a single local code_source_path occurrence to package.
type codeSource struct {
	configPath dyn.Path
	location   dyn.Location
	// value is the raw code_source_path string as written in config.
	value string
}

func PackageCodeSource() bundle.Mutator {
	return &packageCodeSource{}
}

type packageCodeSource struct{}

func (m *packageCodeSource) Name() string {
	return "aicode.PackageCodeSource"
}

func (m *packageCodeSource) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Package two source kinds into a single list of archives to overlay:
	//   - a local-directory code_source_path string (collectLocalCodeSources), and
	//   - a DABs-native code_source block (collectCodeSourceBlocks), which additionally
	//     supports include_paths narrowing and git-ref pinning.
	// Both lower to the same shape: a config path (…ai_runtime_task.code_source_path) to
	// set to the uploaded workspace path, plus the archive bytes to overlay.
	var packaged []packagedSource

	strSources, diags := collectLocalCodeSources(b)
	if diags.HasError() {
		return diags
	}
	for _, cs := range strSources {
		relArchive, archive, err := packageOne(ctx, b, cs)
		if err != nil {
			return diags.Extend(diag.FromErr(err))
		}
		packaged = append(packaged, packagedSource{
			configPath: cs.configPath,
			location:   cs.location,
			relArchive: relArchive,
			archive:    archive,
		})
	}

	blockSources, blockDiags := collectCodeSourceBlocks(b)
	diags = diags.Extend(blockDiags)
	if diags.HasError() {
		return diags
	}
	for _, cs := range blockSources {
		relArchive, archive, err := packageBlock(ctx, b, cs)
		if err != nil {
			return diags.Extend(diag.FromErr(err))
		}
		packaged = append(packaged, packagedSource{
			configPath: cs.configPath,
			location:   cs.location,
			relArchive: relArchive,
			archive:    archive,
		})
	}

	if len(packaged) == 0 {
		return diags
	}

	// overlayFiles maps each archive's sync-relative path to its bytes; remotePaths maps
	// each config location to the synced workspace path its archive will occupy. Both
	// are built before any config mutation so packaging failures are reported first. The
	// archives are added to the sync root as in-memory overlay files (see below) rather
	// than written to disk, so the user's working tree is not dirtied by deploy.
	overlayFiles := make(map[string][]byte, len(packaged))
	remotePaths := make(map[string]string, len(packaged))
	for _, p := range packaged {
		overlayFiles[p.relArchive] = p.archive
		// The workspace path the archive occupies once file sync uploads it. Matches
		// how command_path is translated (workspace.file_path + sync-relative path).
		remotePaths[p.configPath.String()] = path.Join(b.Config.Workspace.FilePath, p.relArchive)
	}

	// Overlay the archives onto the sync root: bundle file sync walks and uploads
	// them like real files, but they never touch the user's working tree.
	syncRoot, err := vfs.Overlay(b.SyncRoot, overlayFiles)
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}
	b.SyncRoot = syncRoot

	// Signal GetSyncIncludePatterns to force-sync the snapshot dir for this bundle,
	// so a user ignore rule can't filter the archives out of the upload set.
	b.HasAiRuntimeCodeSnapshot = true

	err = b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for _, p := range packaged {
			remote := remotePaths[p.configPath.String()]
			var err error
			root, err = dyn.SetByPath(root, p.configPath, dyn.NewValue(remote, []dyn.Location{p.location}))
			if err != nil {
				return root, fmt.Errorf("failed to update code_source_path to %q: %w", remote, err)
			}
		}
		return root, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return diags
}

// packagedSource is one code source packaged into an archive, ready to overlay on the
// sync root and to rewrite into the task's code_source_path.
type packagedSource struct {
	// configPath is the …ai_runtime_task.code_source_path to set to the uploaded path.
	configPath dyn.Path
	location   dyn.Location
	// relArchive is the archive's sync-relative path; archive is its bytes.
	relArchive string
	archive    []byte
}

// snapshotSubdir is the sync-relative dir the archives are placed under (dedicated
// so a snapshot is never nested in the dir it snapshots). See bundle.AiCodeSnapshotDir.
const snapshotSubdir = bundle.AiCodeSnapshotDir

// packageOne packages the local directory for a single code source into a
// reproducible, content-addressed tarball and returns its sync-relative path plus
// the archive bytes. It performs no disk or workspace write: the caller overlays the
// bytes onto the sync root and the deploy-phase file sync uploads them.
func packageOne(ctx context.Context, b *bundle.Bundle, cs codeSource) (string, []byte, error) {
	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(cs.value))
	dirName := filepath.Base(localDir)

	// relBase is the code directory relative to the sync root, used both to scope the
	// sync file list to this directory and to re-base archive entry names under it.
	relBase, err := filepath.Rel(b.SyncRootPath, localDir)
	if err != nil {
		return "", nil, fmt.Errorf("code_source_path %q: %w", cs.value, err)
	}
	relBase = filepath.ToSlash(relBase)

	files, err := codeSourceFiles(ctx, b, relBase)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list files for code_source_path %q: %w", cs.value, err)
	}
	// An empty file list means every file under the directory was filtered out
	// (gitignore / sync.exclude) or the directory is empty. Packaging it would deploy
	// a job with no code, so fail with an actionable message instead.
	if len(files) == 0 {
		return "", nil, fmt.Errorf("code_source_path %q has no files to package (all excluded by .gitignore or sync.exclude, or the directory is empty)", cs.value)
	}

	// Build the archive in memory so its content hash can name the file; the hash is
	// computed while gzipping, so this adds no extra pass over the files.
	var buf bytes.Buffer
	sha, err := buildCodeSnapshot(b.SyncRoot, relBase, files, dirName, &buf)
	if err != nil {
		return "", nil, fmt.Errorf("failed to package code_source_path %q: %w", cs.value, err)
	}
	// Content-addressed name + incremental file sync means an unchanged archive keeps
	// the same synced path and is not re-uploaded.
	relArchive := path.Join(snapshotSubdir, fmt.Sprintf("%s_%s.tar.gz", dirName, sha[:16]))
	log.Debugf(ctx, "packaged code snapshot %s for code_source_path %q", relArchive, cs.value)
	return relArchive, buf.Bytes(), nil
}

// codeSourceFiles returns the files under the code directory (relBase, relative to
// the sync root) that should go into the snapshot. It reuses the bundle's sync
// options so the file list is filtered exactly like bundle file sync: .gitignore
// aware, plus the top-level sync.include/exclude globs. Scoping Paths to relBase
// restricts the walk (and the returned relative paths) to the code directory.
func codeSourceFiles(ctx context.Context, b *bundle.Bundle, relBase string) ([]fileset.File, error) {
	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return nil, err
	}
	// Scope the file list to the code directory (relBase) while keeping the
	// bundle's include/exclude globs, so filtering matches bundle file sync.
	fl, err := libsync.NewFileList(ctx, opts.WorktreeRoot, opts.LocalRoot, []string{relBase}, opts.Include, opts.Exclude)
	if err != nil {
		return nil, err
	}
	all, err := fl.Files(ctx)
	if err != nil {
		return nil, err
	}

	// sync.include is force-added regardless of the scoped walk, so the list can
	// contain files outside the code directory. Keep only what is under relBase, or
	// those strays make an all-filtered directory look non-empty and an empty archive
	// ships.
	if relBase == "." {
		return all, nil
	}
	prefix := relBase + "/"
	return slices.DeleteFunc(all, func(f fileset.File) bool {
		return !strings.HasPrefix(f.Relative, prefix)
	}), nil
}

// collectLocalCodeSources returns every AI Runtime task code_source_path that
// points at a local directory. Already-remote values are skipped.
func collectLocalCodeSources(b *bundle.Bundle) ([]codeSource, diag.Diagnostics) {
	var sources []codeSource
	var diags diag.Diagnostics

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(root, codeSourcePattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			value, ok := v.AsString()
			if !ok {
				return v, fmt.Errorf("expected string, got %s", v.Kind())
			}
			if !libraries.IsLocalPath(value) {
				return v, nil
			}
			// Only package a local *directory*. A local file (e.g. a pre-built
			// tarball delivered via an `artifacts` block) is left alone so it flows
			// through the standard artifact-upload path as a file. aicode.Validate
			// applies the same directory check, so the two stay in agreement.
			localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(value))
			isDir, err := isExistingDir(localDir)
			if err != nil {
				return v, fmt.Errorf("code_source_path %q: %w", value, err)
			}
			if !isDir {
				return v, nil
			}
			sources = append(sources, codeSource{
				configPath: p,
				location:   v.Location(),
				value:      value,
			})
			return v, nil
		})
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return sources, diags
}

// taskPattern matches every job task, used to locate a task by task_key when lowering
// a code_source block (keyed by task_key in Root.AiRuntimeExtras).
var taskPattern = dyn.NewPattern(
	dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(),
	dyn.Key("tasks"), dyn.AnyIndex(),
)

// blockSource is a DABs-native code_source block to package: the config path of the
// task's code_source_path to write, a location for diagnostics, and the block options.
type blockSource struct {
	configPath dyn.Path
	location   dyn.Location
	opts       config.CodeSourceOptions
}

// collectCodeSourceBlocks returns one blockSource per task that carries a code_source
// block (extracted into Root.AiRuntimeExtras before normalization). It matches each
// stashed (job, task_key) to the task's current position in the config tree — so it
// tracks the task after MergeJobTasks reorders tasks by task_key — and yields the
// …ai_runtime_task.code_source_path config path to set to the uploaded workspace path.
func collectCodeSourceBlocks(b *bundle.Bundle) ([]blockSource, diag.Diagnostics) {
	var sources []blockSource
	var diags diag.Diagnostics
	if len(b.Config.AiRuntimeExtras) == 0 {
		return nil, diags
	}

	// Read-only pattern walk over the current tree — the callback returns each task
	// unchanged and only records matches, so it does not go through Config.Mutate.
	_, err := dyn.MapByPattern(b.Config.Value(), taskPattern, func(p dyn.Path, task dyn.Value) (dyn.Value, error) {
		jobName := p[2].Key()
		taskKey, ok := task.Get("task_key").AsString()
		if !ok {
			return task, nil
		}
		extras, ok := b.Config.AiRuntimeExtras[jobName][taskKey]
		if !ok || extras.CodeSource == nil {
			return task, nil
		}
		sources = append(sources, blockSource{
			configPath: p.Append(dyn.Key("ai_runtime_task"), dyn.Key("code_source_path")),
			location:   task.Get("ai_runtime_task").Location(),
			opts:       *extras.CodeSource,
		})
		return task, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return sources, diags
}

// packageBlock packages a code_source block into a content-addressed tarball and
// returns its sync-relative path plus the archive bytes. Like packageOne it performs
// no disk or workspace write; the caller overlays the bytes and file sync uploads them.
//
// A git ref archives the pinned commit via `git archive` (uncommitted changes are not
// included); otherwise the working tree is packaged via the same file-list path as a
// bare code_source_path, narrowed by include_paths.
func packageBlock(ctx context.Context, b *bundle.Bundle, cs blockSource) (string, []byte, error) {
	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(cs.opts.RootPath))
	dirName := filepath.Base(localDir)

	if cs.opts.Git != nil {
		commit, err := resolveGitCommit(ctx, localDir, cs.opts.Git, cs.opts.IncludePaths)
		if err != nil {
			return "", nil, err
		}
		archive, sha, err := buildGitArchive(ctx, localDir, commit, dirName, cs.opts.IncludePaths)
		if err != nil {
			return "", nil, err
		}
		relArchive := path.Join(snapshotSubdir, fmt.Sprintf("%s_%s.tar.gz", dirName, sha[:16]))
		log.Debugf(ctx, "packaged git code snapshot %s for code_source.root_path %q at %s", relArchive, cs.opts.RootPath, shortSHA(commit))
		return relArchive, archive, nil
	}

	relBase, err := filepath.Rel(b.SyncRootPath, localDir)
	if err != nil {
		return "", nil, fmt.Errorf("code_source.root_path %q: %w", cs.opts.RootPath, err)
	}
	relBase = filepath.ToSlash(relBase)

	files, err := codeSourceFiles(ctx, b, relBase)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list files for code_source.root_path %q: %w", cs.opts.RootPath, err)
	}
	if len(cs.opts.IncludePaths) > 0 {
		files = filterIncludePaths(files, relBase, cs.opts.IncludePaths)
	}
	// An empty file list means everything was filtered out (gitignore / sync.exclude /
	// include_paths) or the directory is empty. Packaging it would deploy a job with no
	// code, so fail with an actionable message instead.
	if len(files) == 0 {
		return "", nil, fmt.Errorf("code_source.root_path %q has no files to package (all excluded by .gitignore, sync.exclude, or include_paths, or the directory is empty)", cs.opts.RootPath)
	}

	var buf bytes.Buffer
	sha, err := buildCodeSnapshot(b.SyncRoot, relBase, files, dirName, &buf)
	if err != nil {
		return "", nil, fmt.Errorf("failed to package code_source.root_path %q: %w", cs.opts.RootPath, err)
	}
	relArchive := path.Join(snapshotSubdir, fmt.Sprintf("%s_%s.tar.gz", dirName, sha[:16]))
	log.Debugf(ctx, "packaged code snapshot %s for code_source.root_path %q", relArchive, cs.opts.RootPath)
	return relArchive, buf.Bytes(), nil
}

// filterIncludePaths keeps only files under one of the include paths. Each include is
// relative to the code directory (relBase); f.Relative is relative to the sync root, so
// the includes are re-based onto relBase before matching.
func filterIncludePaths(files []fileset.File, relBase string, includePaths []string) []fileset.File {
	prefixes := make([]string, 0, len(includePaths))
	for _, inc := range includePaths {
		inc = strings.Trim(strings.TrimSpace(inc), "/")
		if relBase != "." {
			inc = relBase + "/" + inc
		}
		prefixes = append(prefixes, inc)
	}
	return slices.DeleteFunc(files, func(f fileset.File) bool {
		for _, pre := range prefixes {
			// Match an exact file entry or anything under a directory entry.
			if f.Relative == pre || strings.HasPrefix(f.Relative, pre+"/") {
				return false
			}
		}
		return true
	})
}

// isExistingDir reports whether path is an existing directory. A not-exist error
// is not an error here (the path is simply not a directory this mutator packages),
// but any other stat failure — notably a permission error on the parent — is
// surfaced so it is not silently swallowed into "skip".
func isExistingDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}
