package aicode

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingFiler records every Write and reports files as absent, so an upload
// always happens and its target name is captured.
type recordingFiler struct {
	filer.Filer
	written map[string][]byte
	exists  map[string]bool
}

func newRecordingFiler() *recordingFiler {
	return &recordingFiler{written: map[string][]byte{}, exists: map[string]bool{}}
}

func (f *recordingFiler) Write(ctx context.Context, p string, r io.Reader, _ ...filer.WriteMode) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.written[p] = b
	return nil
}

func (f *recordingFiler) Stat(ctx context.Context, p string) (fs.FileInfo, error) {
	if f.exists[p] {
		return fakeFileInfo{}, nil
	}
	return nil, fs.ErrNotExist
}

type fakeFileInfo struct{ fs.FileInfo }

func bundleWithCodeSource(t *testing.T, codeSourcePath string) (*bundle.Bundle, string) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "train.py"), []byte("print('x')"), 0o644))

	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Workspace: config.Workspace{
				ArtifactPath: "/Workspace/Users/me/.bundle/artifacts",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"train": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "train",
									AiRuntimeTask: &jobs.AiRuntimeTask{
										Experiment:     "exp",
										CodeSourcePath: codeSourcePath,
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
	return b, dir
}

func TestPackageAndUploadRewritesLocalCodeSource(t *testing.T) {
	b, _ := bundleWithCodeSource(t, "src")
	f := newRecordingFiler()

	diags := bundle.Apply(t.Context(), b, &packageAndUpload{client: f})
	require.Empty(t, diags)

	rewritten := b.Config.Resources.Jobs["train"].Tasks[0].AiRuntimeTask.CodeSourcePath
	assert.Regexp(t, `^/Workspace/Users/me/\.bundle/artifacts/\.internal/src_[0-9a-f]{16}\.tar\.gz$`, rewritten)

	require.Len(t, f.written, 1)
	for name := range f.written {
		assert.Regexp(t, `^src_[0-9a-f]{16}\.tar\.gz$`, name)
	}
}

func TestPackageAndUploadSkipsRemoteCodeSource(t *testing.T) {
	b, _ := bundleWithCodeSource(t, "/Volumes/main/default/code/existing.tar.gz")
	f := newRecordingFiler()

	diags := bundle.Apply(t.Context(), b, &packageAndUpload{client: f})
	require.Empty(t, diags)

	assert.Equal(t, "/Volumes/main/default/code/existing.tar.gz",
		b.Config.Resources.Jobs["train"].Tasks[0].AiRuntimeTask.CodeSourcePath)
	assert.Empty(t, f.written, "remote code_source_path must not trigger an upload")
}

func TestPackageAndUploadSkipsUploadWhenArchiveExists(t *testing.T) {
	b, _ := bundleWithCodeSource(t, "src")

	// Determine the archive name the mutator will compute, then mark it present.
	var buf bytes.Buffer
	sha, err := buildTarball(t.Context(), vfs.MustNew(filepath.Join(b.SyncRootPath, "src")), "src", &buf)
	require.NoError(t, err)
	archiveName := "src_" + sha[:16] + ".tar.gz"

	f := newRecordingFiler()
	f.exists[archiveName] = true

	diags := bundle.Apply(t.Context(), b, &packageAndUpload{client: f})
	require.Empty(t, diags)

	assert.Empty(t, f.written, "existing archive must not be re-uploaded")
	assert.Equal(t, "/Workspace/Users/me/.bundle/artifacts/.internal/"+archiveName,
		b.Config.Resources.Jobs["train"].Tasks[0].AiRuntimeTask.CodeSourcePath)
}
