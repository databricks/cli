package resourcemutator_test

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pngHeader  = []byte("\x89PNG\r\n\x1a\n")
	jpegHeader = []byte("\xff\xd8\xff\xe0")
)

func padTo(header []byte, size int) []byte {
	return append(bytes.Clone(header), make([]byte, size-len(header))...)
}

func TestConfigureAppThumbnail(t *testing.T) {
	tests := []struct {
		name string
		// contents is written to thumbnail.png when non-nil.
		contents      []byte
		thumbnailPath string
		wantErr       string
	}{
		{name: "png", contents: padTo(pngHeader, 100), thumbnailPath: "thumbnail.png"},
		{name: "jpeg", contents: padTo(jpegHeader, 100), thumbnailPath: "thumbnail.png"},
		{name: "png at the size limit", contents: padTo(pngHeader, 250_000), thumbnailPath: "thumbnail.png"},
		{name: "no thumbnail_path", thumbnailPath: ""},
		{
			name:          "png over the size limit",
			contents:      padTo(pngHeader, 250_001),
			thumbnailPath: "thumbnail.png",
			wantErr:       "app thumbnail thumbnail.png is 250001 bytes, larger than the limit of 250000 bytes",
		},
		{
			name:          "gif",
			contents:      padTo([]byte("GIF89a"), 100),
			thumbnailPath: "thumbnail.png",
			wantErr:       "app thumbnail thumbnail.png must be a PNG or JPEG image, got image/gif",
		},
		{
			name:          "empty file",
			contents:      []byte{},
			thumbnailPath: "thumbnail.png",
			wantErr:       "must be a PNG or JPEG image",
		},
		{
			name:          "missing file",
			thumbnailPath: "missing.png",
			wantErr:       "failed to read app thumbnail missing.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.contents != nil {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "thumbnail.png"), tt.contents, 0o600))
			}

			app := &resources.App{ThumbnailPath: tt.thumbnailPath}
			b := &bundle.Bundle{
				SyncRootPath:   dir,
				BundleRootPath: dir,
				SyncRoot:       vfs.MustNew(dir),
				Config: config.Root{
					Resources: config.Resources{
						Apps: map[string]*resources.App{"my_app": app},
					},
				},
			}

			diags := bundle.ApplySeq(t.Context(), b, resourcemutator.ConfigureAppThumbnail())

			if tt.wantErr != "" {
				assert.ErrorContains(t, diags.Error(), tt.wantErr)
				return
			}
			require.NoError(t, diags.Error())
			want := ""
			if tt.contents != nil {
				want = base64.StdEncoding.EncodeToString(tt.contents)
			}
			assert.Equal(t, want, b.Config.Resources.Apps["my_app"].ThumbnailContent)
		})
	}
}
