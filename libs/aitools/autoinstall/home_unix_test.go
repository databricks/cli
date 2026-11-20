//go:build unix

package autoinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
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

func TestRootInUserHomeOwnedByRoot(t *testing.T) {
	var st unix.Stat_t
	require.NoError(t, unix.Stat("/", &st))
	if st.Uid != 0 {
		t.Skip("the root directory is not owned by root, as in a user namespace that does not map root")
	}
	ctx := env.WithUserHomeDir(t.Context(), "/")
	assert.False(t, rootInUserHome(ctx, 0))
}

func TestRootInUserHomeOwnedByAnotherUser(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a home the test creates is owned by root")
	}
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	assert.True(t, rootInUserHome(ctx, 0))
}
