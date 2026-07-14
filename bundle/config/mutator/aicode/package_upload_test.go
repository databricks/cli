package aicode

import (
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
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// codeSnapshotDir is the workspace location code snapshots are uploaded to in
// tests, derived from the fixture user below.
const codeSnapshotDir = "/Workspace/Users/me@databricks.com/.air/repo_snapshots/src"

// recordingFiler records every Write, so a test can capture the uploaded archive.
type recordingFiler struct {
	filer.Filer
	written map[string][]byte
}

func newRecordingFiler() *recordingFiler {
	return &recordingFiler{written: map[string][]byte{}}
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
	return nil, fs.ErrNotExist
}

// bundleWithCodeSource builds a bundle rooted at dir whose single AI Runtime task
// points at codeSourcePath. The caller populates dir before applying the mutator.
func bundleWithCodeSource(t *testing.T, dir, codeSourcePath string) *bundle.Bundle {
	t.Helper()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Workspace: config.Workspace{
				CurrentUser: &config.User{User: &iam.User{UserName: "me@databricks.com"}},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"train": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:       "train",
									AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp", CodeSourcePath: codeSourcePath},
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

// bundleWithSrc creates a src/ directory with one file and returns the bundle.
func bundleWithSrc(t *testing.T, codeSourcePath string) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "train.py"), []byte("print('x')"), 0o644))
	return bundleWithCodeSource(t, dir, codeSourcePath)
}

func codeSourcePath(b *bundle.Bundle) string {
	return b.Config.Resources.Jobs["train"].Tasks[0].AiRuntimeTask.CodeSourcePath
}

func TestPackageAndUploadRewritesLocalCodeSource(t *testing.T) {
	b := bundleWithSrc(t, "src")
	f := newRecordingFiler()

	diags := bundle.Apply(t.Context(), b, &packageAndUpload{client: f})
	require.Empty(t, diags)

	assert.Regexp(t, `^`+codeSnapshotDir+`/src_\d{8}_\d{6}\.tar\.gz$`, codeSourcePath(b))

	require.Len(t, f.written, 1)
	for name := range f.written {
		assert.Regexp(t, `^src_\d{8}_\d{6}\.tar\.gz$`, name)
	}
}

func TestPackageAndUploadSkipsRemoteCodeSource(t *testing.T) {
	b := bundleWithSrc(t, "/Volumes/main/default/code/existing.tar.gz")
	f := newRecordingFiler()

	diags := bundle.Apply(t.Context(), b, &packageAndUpload{client: f})
	require.Empty(t, diags)

	assert.Equal(t, "/Volumes/main/default/code/existing.tar.gz", codeSourcePath(b))
	assert.Empty(t, f.written, "remote code_source_path must not trigger an upload")
}
