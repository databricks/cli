package phases

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/stretchr/testify/assert"
)

func TestStatePathScopeSignals(t *testing.T) {
	cases := []struct {
		name                    string
		rootPath                string
		statePath               string
		permissions             []resources.Permission
		isShared                bool
		outsideRoot             bool
		scopeExceedsPermissions bool
	}{
		{
			name:      "state under root, not shared",
			rootPath:  "/Workspace/Users/me@example.test/bundle",
			statePath: "/Workspace/Users/me@example.test/bundle/state",
		},
		{
			name:        "state outside root, not shared",
			rootPath:    "/Workspace/Users/me@example.test/bundle",
			statePath:   "/Workspace/Users/me@example.test/other-state",
			outsideRoot: true,
		},
		{
			name:                    "state shared without users manage",
			rootPath:                "/Workspace/Users/me@example.test/bundle",
			statePath:               "/Workspace/Shared/state",
			isShared:                true,
			outsideRoot:             true,
			scopeExceedsPermissions: true,
		},
		{
			name:        "state shared with users manage",
			rootPath:    "/Workspace/Users/me@example.test/bundle",
			statePath:   "/Workspace/Shared/state",
			permissions: []resources.Permission{{Level: permissions.CAN_MANAGE, GroupName: "users"}},
			isShared:    true,
			outsideRoot: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{Config: config.Root{
				Workspace: config.Workspace{
					RootPath:  tc.rootPath,
					StatePath: tc.statePath,
				},
				Permissions: tc.permissions,
			}}
			isShared, outsideRoot, scopeExceeds := statePathScopeSignals(b)
			assert.Equal(t, tc.isShared, isShared, "isShared")
			assert.Equal(t, tc.outsideRoot, outsideRoot, "outsideRoot")
			assert.Equal(t, tc.scopeExceedsPermissions, scopeExceeds, "scopeExceedsPermissions")
		})
	}
}
