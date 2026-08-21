package aicode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bundleForValidate(t *testing.T, codeSourcePath string, gitSource *jobs.GitSource) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"train": {
						JobSettings: jobs.JobSettings{
							GitSource: gitSource,
							Tasks: []jobs.Task{
								{
									TaskKey:       "train",
									AiRuntimeTask: &jobs.AiRuntimeTask{CodeSourcePath: codeSourcePath},
								},
							},
						},
					},
				},
			},
		},
	}
	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "databricks.yml")}})
	return b
}

// mkCodeDir creates a code_source directory (with one file) under the bundle's
// sync root, so the path resolves to an existing directory this mutator packages.
func mkCodeDir(t *testing.T, b *bundle.Bundle, rel string) {
	t.Helper()
	dir := filepath.Join(b.SyncRootPath, rel)
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "train.py"), []byte("print()\n"), 0o600))
}

// A local path that is not an existing directory is left alone: it flows through
// the standard artifact-upload path (e.g. a pre-built tarball built by an
// `artifacts` block, which does not exist yet at validate time).
func TestValidateNonDirectoryCodeSourceIsSkipped(t *testing.T) {
	// Missing path (nothing on disk yet).
	b := bundleForValidate(t, "does-not-exist", nil)
	assert.Empty(t, Validate().Apply(t.Context(), b))

	// Existing local file (a pre-built tarball), not a directory.
	b = bundleForValidate(t, "code.tgz", nil)
	require.NoError(t, os.WriteFile(filepath.Join(b.SyncRootPath, "code.tgz"), []byte("x"), 0o600))
	assert.Empty(t, Validate().Apply(t.Context(), b))
}

func TestValidateGitSourceConflict(t *testing.T) {
	b := bundleForValidate(t, "src", &jobs.GitSource{GitUrl: "https://example.invalid/repo"})
	mkCodeDir(t, b, "src")
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "cannot be combined with git_source")
}

func TestValidateRemoteCodeSourceIsSkipped(t *testing.T) {
	b := bundleForValidate(t, "/Volumes/main/default/code/x.tar.gz", nil)
	diags := Validate().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

// A code_source_path escaping the bundle sync root is rejected with a clear
// message (it can't be synced), rather than failing later as an opaque io/fs error.
func TestValidateCodeSourceOutsideBundleRoot(t *testing.T) {
	b := bundleForValidate(t, "../shared", nil)
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "outside the bundle root")
}

// Source-linked deployment doesn't copy files to the workspace file path, so the
// packaged snapshot would never be uploaded; the combination is rejected.
func TestValidateSourceLinkedConflict(t *testing.T) {
	b := bundleForValidate(t, "src", nil)
	mkCodeDir(t, b, "src")
	enabled := true
	b.Config.Presets.SourceLinkedDeployment = &enabled
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "source-linked deployment")
}

// A local code_source_path nested under a for_each_task is not packaged by the
// mutator, so it is rejected rather than silently skipped.
func TestValidateForEachTaskCodeSourceRejected(t *testing.T) {
	dir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"train": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "fanout",
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{
											AiRuntimeTask: &jobs.AiRuntimeTask{CodeSourcePath: "src"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "databricks.yml")}})
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "for_each_task")
}

// A sync.exclude pattern matching the reserved snapshot dir would drop the
// generated archive from the upload (exclude wins over include), so it is rejected.
func TestValidateSyncExcludeMatchingSnapshotDir(t *testing.T) {
	for _, pattern := range []string{".air_snapshots/*", "**/*.tar.gz", ".air_snapshots"} {
		b := bundleForValidate(t, "src", nil)
		mkCodeDir(t, b, "src")
		b.Config.Sync.Exclude = []string{pattern}
		diags := Validate().Apply(t.Context(), b)
		require.Len(t, diags, 1, "pattern %q should be rejected", pattern)
		assert.Contains(t, diags[0].Summary, "sync.exclude")
	}
}

// An unrelated sync.exclude is fine — the guard only fires for patterns that would
// filter the snapshot dir.
func TestValidateSyncExcludeUnrelatedIsAllowed(t *testing.T) {
	b := bundleForValidate(t, "src", nil)
	mkCodeDir(t, b, "src")
	b.Config.Sync.Exclude = []string{"*.log", "build/**"}
	assert.Empty(t, Validate().Apply(t.Context(), b))
}

// A real file/dir at the reserved snapshot path collides with the overlay, so it is
// rejected.
func TestValidateReservedSnapshotDirCollision(t *testing.T) {
	b := bundleForValidate(t, "src", nil)
	mkCodeDir(t, b, "src")
	require.NoError(t, os.MkdirAll(filepath.Join(b.SyncRootPath, ".air_snapshots"), 0o700))
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "reserved")
}

// The snapshot-dir guards only fire when a task actually packages a local
// code_source — an unrelated bundle with a stray .air_snapshots is not this
// mutator's concern.
func TestValidateSnapshotGuardsSkippedWithoutLocalCodeSource(t *testing.T) {
	b := bundleForValidate(t, "/Volumes/main/default/code/x.tar.gz", nil)
	require.NoError(t, os.MkdirAll(filepath.Join(b.SyncRootPath, ".air_snapshots"), 0o700))
	b.Config.Sync.Exclude = []string{".air_snapshots/*"}
	assert.Empty(t, Validate().Apply(t.Context(), b))
}

// setCodeSourceBlock attaches a DABs-native code_source block to the "train" task,
// as rewriteAiRuntimeCodeSource would have done at load.
func setCodeSourceBlock(b *bundle.Bundle, block config.CodeSourceOptions) {
	b.Config.AiRuntimeExtras = map[string]map[string]config.AiRuntimeTaskExtras{
		"train": {"train": {CodeSource: &block}},
	}
}

func TestValidateCodeSourceBlock(t *testing.T) {
	tests := []struct {
		name  string
		block config.CodeSourceOptions
		// codeSourcePath, when set, is also placed on the task to test the conflict.
		codeSourcePath string
		wantErr        string // "" means no error expected
	}{
		{
			name:  "valid working tree",
			block: config.CodeSourceOptions{RootPath: "src"},
		},
		{
			name:  "valid include_paths",
			block: config.CodeSourceOptions{RootPath: "src", IncludePaths: []string{"a", "b/c"}},
		},
		{
			name:  "valid git commit short",
			block: config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Commit: "abc1234"}},
		},
		{
			name:  "valid git commit full sha",
			block: config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Commit: "0123456789abcdef0123456789abcdef01234567"}},
		},
		{
			name:    "git commit too short",
			block:   config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Commit: "abc123"}},
			wantErr: "invalid code_source.git.commit",
		},
		{
			name:    "git commit is a moving ref",
			block:   config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Commit: "HEAD"}},
			wantErr: "invalid code_source.git.commit",
		},
		{
			name:    "git commit non-hex (branch name)",
			block:   config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Commit: "main"}},
			wantErr: "invalid code_source.git.commit",
		},
		{
			name:    "git commit injection shape",
			block:   config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Commit: "--output=/tmp/x"}},
			wantErr: "invalid code_source.git.commit",
		},
		{
			name:           "conflict with code_source_path",
			block:          config.CodeSourceOptions{RootPath: "src"},
			codeSourcePath: "src",
			wantErr:        "both code_source and code_source_path",
		},
		{
			name:    "empty root_path",
			block:   config.CodeSourceOptions{RootPath: ""},
			wantErr: "root_path is required",
		},
		{
			name:    "root_path outside bundle",
			block:   config.CodeSourceOptions{RootPath: "../escape"},
			wantErr: "outside the bundle root",
		},
		{
			name:    "include_paths absolute",
			block:   config.CodeSourceOptions{RootPath: "src", IncludePaths: []string{"/abs"}},
			wantErr: "must be relative paths",
		},
		{
			name:    "include_paths traversal",
			block:   config.CodeSourceOptions{RootPath: "src", IncludePaths: []string{"a/../b"}},
			wantErr: "'..' traversal",
		},
		{
			name:    "remote_volume not supported",
			block:   config.CodeSourceOptions{RootPath: "src", RemoteVolume: "/Volumes/main/x"},
			wantErr: "remote_volume is not yet supported",
		},
		{
			name:    "git branch and commit both set",
			block:   config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Branch: "main", Commit: "abc"}},
			wantErr: "mutually exclusive",
		},
		{
			name:    "git neither branch nor commit",
			block:   config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{}},
			wantErr: "requires either 'branch' or 'commit'",
		},
		{
			name:    "git invalid branch",
			block:   config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Branch: "bad;rm -rf"}},
			wantErr: "invalid code_source.git.branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bundleForValidate(t, tt.codeSourcePath, nil)
			mkCodeDir(t, b, "src")
			setCodeSourceBlock(b, tt.block)

			diags := Validate().Apply(t.Context(), b)
			if tt.wantErr == "" {
				assert.Empty(t, diags)
				return
			}
			require.True(t, diags.HasError(), "expected an error")
			assert.Contains(t, diags[0].Summary+" "+diags[0].Detail, tt.wantErr)
		})
	}
}

// A code_source.git ref combined with the job's git_source is rejected: the deploy
// engine would fetch task files from git_source and ignore the packaged snapshot.
func TestValidateCodeSourceBlockGitSourceConflict(t *testing.T) {
	b := bundleForValidate(t, "", &jobs.GitSource{GitUrl: "https://example.invalid/repo"})
	mkCodeDir(t, b, "src")
	setCodeSourceBlock(b, config.CodeSourceOptions{RootPath: "src", Git: &config.CodeSourceGit{Commit: "abc1234"}})
	diags := Validate().Apply(t.Context(), b)
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "cannot be combined with the job's git_source")
}
