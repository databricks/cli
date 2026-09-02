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

func TestConfigureDashboardSerializedDashboard(t *testing.T) {
	const fileName = "dashboard.lvdash.json"

	tests := []struct {
		name string
		// filePath is set on the resource as-is (already sync-root-relative).
		filePath string
		// writeFile creates filePath with fileContents before the mutator runs.
		writeFile           bool
		fileContents        string
		setSerialized       bool
		serializedDashboard any
		// wantSerialized is the expected serialized_dashboard after a successful run.
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
			fileContents:   `{"pages": 1}` + "\n",
			wantSerialized: `{"pages": 1}` + "\n",
		},
		{
			// Inline maps are marshaled to a compact JSON string with sorted keys
			// so config and state hold an identical string and don't drift.
			name:                "inline map is marshaled to a JSON string",
			setSerialized:       true,
			serializedDashboard: map[string]any{"pages": 1},
			wantSerialized:      `{"pages":1}`,
		},
		{
			name:                "inline string is left unchanged",
			setSerialized:       true,
			serializedDashboard: `{"pages":1}`,
			wantSerialized:      `{"pages":1}`,
		},
		{
			// Neither field set: the absent field must pass through, not error.
			name:           "neither file_path nor serialized_dashboard passes through",
			wantSerialized: nil,
		},
		{
			name:                "both file_path and serialized_dashboard is rejected",
			filePath:            fileName,
			setSerialized:       true,
			serializedDashboard: map[string]any{"pages": 1},
			wantErr:             "both file_path and serialized_dashboard are set; specify only one",
		},
		{
			name:                "non-structured serialized_dashboard is rejected",
			setSerialized:       true,
			serializedDashboard: true,
			wantErr:             "serialized_dashboard must be a string or map, got bool",
		},
		{
			name: "inline sequence is rejected",
			setSerialized:   true,
			serializedDashboard: []any{map[string]any{"version": 1}},
			wantErr:         "serialized_dashboard must be a string or map, got sequence",
		}
		{
			name:     "unreadable file_path is an error",
			filePath: "does_not_exist.json",
			wantErr:  "failed to read serialized dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.writeFile {
				require.NoError(t, os.WriteFile(filepath.Join(dir, tt.filePath), []byte(tt.fileContents), 0o600))
			}

			dash := &resources.Dashboard{
				DashboardConfig: resources.DashboardConfig{DisplayName: "My Dashboard"},
				FilePath:        tt.filePath,
			}
			if tt.setSerialized {
				dash.SerializedDashboard = tt.serializedDashboard
			}

			b := &bundle.Bundle{
				SyncRootPath:   dir,
				BundleRootPath: dir,
				SyncRoot:       vfs.MustNew(dir),
				Config: config.Root{
					Resources: config.Resources{
						Dashboards: map[string]*resources.Dashboard{"my_dashboard": dash},
					},
				},
			}

			diags := bundle.ApplySeq(t.Context(), b, resourcemutator.ConfigureDashboardSerializedDashboard())

			if tt.wantErr != "" {
				require.Error(t, diags.Error())
				assert.ErrorContains(t, diags.Error(), tt.wantErr)
				return
			}

			require.NoError(t, diags.Error())
			assert.Equal(t, tt.wantSerialized, b.Config.Resources.Dashboards["my_dashboard"].SerializedDashboard)
		})
	}
}
