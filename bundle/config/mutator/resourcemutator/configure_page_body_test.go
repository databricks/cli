package resourcemutator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/knowledge"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurePageBody(t *testing.T) {
	const fileName = "chargeback.md"
	// Markdown is whitespace sensitive, so the file is used as is: no trimming, and no
	// newline added or removed at the end.
	const fileContents = "# Chargeback\n\n    indented code\n\nNo newline at the end"

	tests := []struct {
		name      string
		filePath  string
		body      string
		writeFile bool
		wantBody  string
		wantErr   string
	}{
		{
			name:      "file_path only loads the file",
			filePath:  fileName,
			writeFile: true,
			wantBody:  fileContents,
		},
		{
			name:     "body only is left as is",
			body:     "inline",
			wantBody: "inline",
		},
		{
			name:     "neither is left as is",
			wantBody: "",
		},
		{
			name:      "both is an error",
			filePath:  fileName,
			body:      "inline",
			writeFile: true,
			wantErr:   "both file_path and body are set; specify only one",
		},
		{
			name:     "missing file is an error",
			filePath: fileName,
			wantErr:  "failed to read page body from file_path chargeback.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.writeFile {
				require.NoError(t, os.WriteFile(filepath.Join(dir, fileName), []byte(fileContents), 0o600))
			}

			b := &bundle.Bundle{
				SyncRootPath:   dir,
				BundleRootPath: dir,
				SyncRoot:       vfs.MustNew(dir),
				Config: config.Root{
					Resources: config.Resources{
						Pages: map[string]*resources.Page{
							"my_page": {
								Page: knowledge.Page{
									Domain:      "domains/1234",
									DisplayName: "Chargeback",
									Body:        tt.body,
								},
								FilePath: tt.filePath,
							},
						},
					},
				},
			}

			diags := bundle.ApplySeq(t.Context(), b, resourcemutator.ConfigurePageBody())

			if tt.wantErr != "" {
				require.Error(t, diags.Error())
				assert.ErrorContains(t, diags.Error(), tt.wantErr)
				return
			}

			require.NoError(t, diags.Error())
			page := b.Config.Resources.Pages["my_page"]
			assert.Equal(t, tt.wantBody, page.Body)
			// file_path is kept: it records where the body came from.
			assert.Equal(t, tt.filePath, page.FilePath)
		})
	}
}
