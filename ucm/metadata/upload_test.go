package metadata_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/deploy"
	ucmfiler "github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStateFiler captures the last Write call so tests can assert on the path,
// mode and payload without touching a real filesystem.
type fakeStateFiler struct {
	writePath string
	writeMode ucmfiler.WriteMode
	writeData []byte
	writeErr  error
}

func (f *fakeStateFiler) Read(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeStateFiler) Write(_ context.Context, path string, r io.Reader, mode ucmfiler.WriteMode) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.writePath = path
	f.writeMode = mode
	f.writeData = data
	return f.writeErr
}

func (f *fakeStateFiler) Delete(_ context.Context, _ string) error { return nil }

func (f *fakeStateFiler) Stat(_ context.Context, _ string) (ucmfiler.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeStateFiler) ReadDir(_ context.Context, _ string) ([]ucmfiler.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func sampleMetadata() metadata.Metadata {
	return metadata.Metadata{
		Version:      metadata.Version,
		CliVersion:   "v-test",
		Ucm:          metadata.UcmMeta{Name: "demo", Target: "dev"},
		DeploymentID: "00000000-0000-0000-0000-000000000001",
		Timestamp:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func TestUploadWritesToFiler(t *testing.T) {
	u := &ucm.Ucm{RootPath: t.TempDir()}
	u.Config.Ucm = config.Ucm{Name: "demo", Target: "dev"}

	f := &fakeStateFiler{}
	backend := deploy.Backend{StateFiler: f}

	err := metadata.Upload(t.Context(), u, backend, sampleMetadata())
	require.NoError(t, err)

	assert.Equal(t, metadata.MetadataFileName, f.writePath)
	assert.True(t, f.writeMode.Has(ucmfiler.WriteModeOverwrite))
	assert.True(t, f.writeMode.Has(ucmfiler.WriteModeCreateParents))

	var got metadata.Metadata
	require.NoError(t, json.Unmarshal(f.writeData, &got))
	assert.Equal(t, sampleMetadata(), got)
}

func TestUploadFilenameDoesNotCollideWithStateFiles(t *testing.T) {
	assert.NotEqual(t, deploy.UcmStateFileName, metadata.MetadataFileName)
	assert.NotEqual(t, deploy.TfStateFileName, metadata.MetadataFileName)
}

func TestUploadReturnsErrorOnFilerFailure(t *testing.T) {
	u := &ucm.Ucm{RootPath: t.TempDir()}
	u.Config.Ucm = config.Ucm{Name: "demo", Target: "dev"}

	boom := errors.New("boom")
	f := &fakeStateFiler{writeErr: boom}
	backend := deploy.Backend{StateFiler: f}

	err := metadata.Upload(t.Context(), u, backend, sampleMetadata())
	require.ErrorIs(t, err, boom)
}

func TestUploadRejectsNilUcm(t *testing.T) {
	backend := deploy.Backend{StateFiler: &fakeStateFiler{}}
	err := metadata.Upload(t.Context(), nil, backend, sampleMetadata())
	require.Error(t, err)
}

func TestUploadRejectsMissingStateFiler(t *testing.T) {
	u := &ucm.Ucm{RootPath: t.TempDir()}
	u.Config.Ucm = config.Ucm{Name: "demo", Target: "dev"}

	err := metadata.Upload(t.Context(), u, deploy.Backend{}, sampleMetadata())
	require.Error(t, err)
}

// TestUploadEndToEndWithLocalFiler exercises Upload against the same
// LocalClient-backed StateFiler used by the phases fixture so we catch
// regressions where the metadata blob disagrees with what tests observe on
// disk via the filer layer.
func TestUploadEndToEndWithLocalFiler(t *testing.T) {
	u := &ucm.Ucm{RootPath: t.TempDir()}
	u.Config.Ucm = config.Ucm{Name: "demo", Target: "dev"}

	remoteDir := t.TempDir()
	remote, err := libsfiler.NewLocalClient(remoteDir)
	require.NoError(t, err)
	backend := deploy.Backend{StateFiler: ucmfiler.NewStateFilerFromFiler(remote)}

	require.NoError(t, metadata.Upload(t.Context(), u, backend, sampleMetadata()))

	rc, err := backend.StateFiler.Read(t.Context(), metadata.MetadataFileName)
	require.NoError(t, err)
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)

	var got metadata.Metadata
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, sampleMetadata(), got)
}
