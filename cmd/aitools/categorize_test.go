package aitools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyInstallError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want protos.AitoolsErrorCategory
	}{
		{
			name: "nil is success",
			err:  nil,
			want: protos.AitoolsErrorCategoryUnspecified,
		},
		{
			name: "blocked cli not on path",
			err:  &installer.BlockedError{Agent: "claude-code", Reason: installer.ReasonCLINotOnPath},
			want: protos.AitoolsErrorCategoryCLINotOnPath,
		},
		{
			name: "blocked install failed",
			err:  &installer.BlockedError{Agent: "codex", Reason: installer.ReasonInstallFailed},
			want: protos.AitoolsErrorCategoryPluginInstallFailed,
		},
		{
			name: "blocked error with unknown reason is uncategorized",
			err:  &installer.BlockedError{Agent: "codex", Reason: "some-future-reason"},
			want: protos.AitoolsErrorCategoryUncategorized,
		},
		{
			name: "wrapped skill not found",
			err:  fmt.Errorf("resolve failed: %w", &installer.SkillError{Skill: "databricks", Reason: installer.ReasonSkillNotFound, Detail: "not found"}),
			want: protos.AitoolsErrorCategorySkillNotFound,
		},
		{
			name: "version incompatible",
			err:  &installer.SkillError{Skill: "databricks", Reason: installer.ReasonVersionIncompatible, Detail: "requires CLI version 0.5 (running 0.4)"},
			want: protos.AitoolsErrorCategoryVersionIncompatible,
		},
		{
			name: "experimental skill",
			err:  &installer.SkillError{Skill: "test-exp", Reason: installer.ReasonExperimentalSkill, Detail: "is experimental; use --experimental to install"},
			want: protos.AitoolsErrorCategoryExperimentalSkill,
		},
		{
			name: "skill error with unknown reason is uncategorized",
			err:  &installer.SkillError{Skill: "databricks", Reason: "some-future-reason"},
			want: protos.AitoolsErrorCategoryUncategorized,
		},
		{
			name: "blocked error joined with another error is still classified",
			err:  errors.Join(&installer.BlockedError{Agent: "codex", Reason: installer.ReasonInstallFailed}, errors.New("other")),
			want: protos.AitoolsErrorCategoryPluginInstallFailed,
		},
		{
			name: "unrecognized error is uncategorized",
			err:  errors.New("boom"),
			want: protos.AitoolsErrorCategoryUncategorized,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, classifyInstallError(tc.err))
		})
	}
}

func TestClassifyInstallErrorGenericCauses(t *testing.T) {
	urlErr := &url.Error{Op: "Get", URL: "https://example.test", Err: errors.New("connection refused")}
	cases := []struct {
		name string
		err  error
		want protos.AitoolsErrorCategory
	}{
		{"network", fmt.Errorf("failed to fetch manifest: %w", urlErr), protos.AitoolsErrorCategoryNetwork},
		// The shape net/http returns for a refused connection: the Errno sits under the network types.
		{"network errno", fmt.Errorf("failed to fetch manifest: %w", &url.Error{Op: "Get", URL: "https://example.test", Err: &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}}), protos.AitoolsErrorCategoryNetwork},
		// A sandbox that blocks outbound sockets: a network failure, even though the Errno is EACCES.
		{"network permission denied", fmt.Errorf("failed to fetch manifest: %w", &url.Error{Op: "Get", URL: "https://example.test", Err: &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.EACCES)}}), protos.AitoolsErrorCategoryNetwork},
		{"network timeout", fmt.Errorf("failed to fetch manifest: %w", &url.Error{Op: "Get", URL: "https://example.test", Err: os.ErrDeadlineExceeded}), protos.AitoolsErrorCategoryTimeout},
		{"deadline", fmt.Errorf("fetch: %w", context.DeadlineExceeded), protos.AitoolsErrorCategoryTimeout},
		{"cancelled", fmt.Errorf("fetch: %w", context.Canceled), protos.AitoolsErrorCategoryCommandInterrupted},
		{"cancelled request", &url.Error{Op: "Get", URL: "https://example.test", Err: context.Canceled}, protos.AitoolsErrorCategoryCommandInterrupted},
		{"manifest missing", fmt.Errorf("skills manifest at x@y: %w", clicompat.ErrNotFound), protos.AitoolsErrorCategoryManifestNotFound},
		// The manifest, release payload and install state all decode JSON; only typed errors can tell them apart.
		{"json syntax error stays uncategorized", fmt.Errorf("failed to parse manifest: %w", json.Unmarshal([]byte("<html>"), &struct{}{})), protos.AitoolsErrorCategoryUncategorized},
		{"json type error stays uncategorized", fmt.Errorf("failed to parse manifest: %w", json.Unmarshal([]byte(`{"a":1}`), &struct{ A string }{})), protos.AitoolsErrorCategoryUncategorized},
		{"permission", fmt.Errorf("write: %w", &fs.PathError{Op: "open", Path: "/x", Err: fs.ErrPermission}), protos.AitoolsErrorCategoryPermissionDenied},
		// syscall.Errno implements net.Error, so OS errors must not reach the network check.
		{"permission errno", fmt.Errorf("write: %w", &fs.PathError{Op: "open", Path: "/x", Err: syscall.EACCES}), protos.AitoolsErrorCategoryPermissionDenied},
		{"permission on rename", fmt.Errorf("save state: %w", &os.LinkError{Op: "rename", Old: "/a", New: "/b", Err: syscall.EACCES}), protos.AitoolsErrorCategoryPermissionDenied},
		{"filesystem", fmt.Errorf("write: %w", &fs.PathError{Op: "mkdir", Path: "/x", Err: errors.New("read-only file system")}), protos.AitoolsErrorCategoryFilesystem},
		{"filesystem from os", fmt.Errorf("read state: %w", readMissingFile(t)), protos.AitoolsErrorCategoryFilesystem},
		{"filesystem on rename", fmt.Errorf("replace skill: %w", &os.LinkError{Op: "rename", Old: "/a", New: "/b", Err: syscall.EBUSY}), protos.AitoolsErrorCategoryFilesystem},
		{"filesystem syscall", fmt.Errorf("resolve cwd: %w", os.NewSyscallError("getwd", syscall.ENOENT)), protos.AitoolsErrorCategoryFilesystem},
		{"user abort", huh.ErrUserAborted, protos.AitoolsErrorCategoryUserAborted},
		{"plain text stays uncategorized", errors.New("something"), protos.AitoolsErrorCategoryUncategorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, classifyInstallError(tc.err))
		})
	}
}

// A corrupt install state must not be blamed on the remote manifest. It stays
// UNCATEGORIZED until the installer returns a typed state error, which should
// change this to STATE_INVALID.
func TestClassifyInstallErrorCorruptStateIsUncategorized(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".state.json"), []byte("{not json"), 0o600))
	_, err := installer.LoadState(dir)
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, classifyInstallError(fmt.Errorf("failed to load install state: %w", err)))
}

// readMissingFile returns the error the OS gives for a missing file, so the
// test covers the platform's own error value (syscall.ENOENT on Unix,
// ERROR_FILE_NOT_FOUND on Windows).
func readMissingFile(t *testing.T) error {
	_, err := os.ReadFile(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
	return err
}
