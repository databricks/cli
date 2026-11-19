//go:build unix

package autoinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestRootInUserHome(t *testing.T) {
	userHome := t.TempDir()
	tests := []struct {
		name string
		euid int
		home string
	}{
		{name: "the user in their home", euid: os.Geteuid(), home: userHome},
		{name: "another user in the home", euid: os.Geteuid() + 1, home: userHome},
		// The root directory is owned by root on every Unix system.
		{name: "root in a root-owned home", euid: 0, home: "/"},
		{name: "root with a home that does not exist", euid: 0, home: filepath.Join(userHome, "missing")},
		{name: "root without a home", euid: 0, home: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := env.WithUserHomeDir(t.Context(), tc.home)
			assert.False(t, rootInUserHome(ctx, tc.euid))
		})
	}
}

func TestRootInUserHomeOwnedByAnotherUser(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a home the test creates is owned by root")
	}
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	assert.True(t, rootInUserHome(ctx, 0))
}
