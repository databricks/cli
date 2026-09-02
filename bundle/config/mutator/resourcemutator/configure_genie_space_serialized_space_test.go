package resourcemutator_test

import (
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

func TestConfigureGenieSpaceSerializedSpace(t *testing.T) {
	const fileName = "space.geniespace.json"

	tests := []struct {
		name string
		// filePath is set on the resource as-is (already sync-root-relative).
		filePath string
		// writeFile creates filePath with fileContents before the mutator runs.
		writeFile       bool
		fileContents    string
		setSerialized   bool
		serializedSpace any
		// wantSerialized is the expected serialized_space after a successful run.
		wantSerialized any
		// wantErr, when non-empty, is a substring expected in the diagnostics.
		wantErr string
	}{
		{
			// The file is read verbatim, so formatting and the trailing newline
			// are preserved (unlike the inline path, which re-marshals).
			name:           "file_path reads file contents verbatim",
			filePath:       fileName,
			writeFile:      true,
			fileContents:   `{"version": 1}` + "\n",
			wantSerialized: `{"version": 1}` + "\n",
		},
		{
			// Inline maps are marshaled to a compact JSON string with sorted keys
			// so config and state hold an identical string and don't drift.
			name:            "inline map is marshaled to a JSON string",
			setSerialized:   true,
			serializedSpace: map[string]any{"version": 1},
			wantSerialized:  `{"version":1}`,
		},
		{
			name:            "inline string is left unchanged",
			setSerialized:   true,
			serializedSpace: `{"version":1}`,
			wantSerialized:  `{"version":1}`,
		},
		{
			// Neither field set: the absent field must pass through, not error.
			name:           "neither file_path nor serialized_space passes through",
			wantSerialized: nil,
		},
		{
			name:            "both file_path and serialized_space is rejected",
			filePath:        fileName,
			setSerialized:   true,
			serializedSpace: map[string]any{"version": 1},
			wantErr:         "both file_path and serialized_space are set; specify only one",
		},
		{
			name:            "non-structured serialized_space is rejected",
			setSerialized:   true,
			serializedSpace: true,
			wantErr:         "serialized_space must be a string, map, or sequence, got bool",
		},
		{
			name:     "unreadable file_path is an error",
			filePath: "does_not_exist.json",
			wantErr:  "failed to read serialized genie space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.writeFile {
				require.NoError(t, os.WriteFile(filepath.Join(dir, tt.filePath), []byte(tt.fileContents), 0o600))
			}

			gs := &resources.GenieSpace{
				GenieSpaceConfig: resources.GenieSpaceConfig{Title: "My Genie Space"},
				FilePath:         tt.filePath,
			}
			if tt.setSerialized {
				gs.SerializedSpace = tt.serializedSpace
			}

			b := &bundle.Bundle{
				SyncRootPath:   dir,
				BundleRootPath: dir,
				SyncRoot:       vfs.MustNew(dir),
				Config: config.Root{
					Resources: config.Resources{
						GenieSpaces: map[string]*resources.GenieSpace{"my_space": gs},
					},
				},
			}

			diags := bundle.ApplySeq(t.Context(), b, resourcemutator.ConfigureGenieSpaceSerializedSpace())

			if tt.wantErr != "" {
				require.Error(t, diags.Error())
				assert.ErrorContains(t, diags.Error(), tt.wantErr)
				return
			}

			require.NoError(t, diags.Error())
			assert.Equal(t, tt.wantSerialized, b.Config.Resources.GenieSpaces["my_space"].SerializedSpace)
		})
	}
}
