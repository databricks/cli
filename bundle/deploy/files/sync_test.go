package files

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dbr"
	"github.com/stretchr/testify/assert"
)

func TestWritesOntoSource(t *testing.T) {
	for _, tc := range []struct {
		name         string
		isDbr        bool
		syncRootPath string
		filePath     string
		expect       bool
	}{
		{
			name:         "file_path is the workspace sync root",
			isDbr:        true,
			syncRootPath: "/Workspace/Users/me/my_bundle",
			filePath:     "/Workspace/Users/me/my_bundle/",
			expect:       true,
		},
		{
			name:         "file_path from a variable without the /Workspace prefix",
			isDbr:        true,
			syncRootPath: "/Workspace/Users/me/my_bundle",
			filePath:     "/Users/me/my_bundle",
			expect:       true,
		},
		{
			name:         "default file_path from the workspace",
			isDbr:        true,
			syncRootPath: "/Workspace/Users/me/my_bundle",
			filePath:     "/Workspace/Users/me/.bundle/my_bundle/default/files",
			expect:       false,
		},
		{
			name:         "same path outside the runtime",
			isDbr:        false,
			syncRootPath: "/Workspace/Users/me/my_bundle",
			filePath:     "/Workspace/Users/me/my_bundle",
			expect:       false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := dbr.MockRuntime(t.Context(), dbr.Environment{IsDbr: tc.isDbr})
			b := &bundle.Bundle{
				SyncRootPath: tc.syncRootPath,
				Config: config.Root{
					Workspace: config.Workspace{FilePath: tc.filePath},
				},
			}
			assert.Equal(t, tc.expect, writesOntoSource(ctx, b))
		})
	}

	t.Run("no runtime detection", func(t *testing.T) {
		assert.False(t, writesOntoSource(t.Context(), &bundle.Bundle{}))
	})
}
