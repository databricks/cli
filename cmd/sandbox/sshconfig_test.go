package sandbox

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSSHConfigBlockShape(t *testing.T) {
	block := buildSSHConfigBlock("/home/u/.ssh/sandbox_ed25519", []string{"uw2.dbrx.dev"}, "2222")

	// The Host key is the literal gateway hostname (so UI editor deep
	// links like `ssh-remote+<id>@<gateway>` resolve directly), and we
	// only set the two directives that meaningfully differ from SSH
	// defaults: Port (the gateway is on 2222) and IdentityFile +
	// IdentitiesOnly (pin our key, suppress the offer cascade).
	assert.Contains(t, block, "# Managed by")
	assert.Contains(t, block, "Host uw2.dbrx.dev\n")
	assert.Contains(t, block, "Port 2222")
	assert.Contains(t, block, "IdentityFile /home/u/.ssh/sandbox_ed25519")
	assert.Contains(t, block, "IdentitiesOnly yes")

	// No HostName — Host already IS the real host, so HostName would
	// be a redundant self-reference.
	assert.NotContains(t, block, "HostName ", "HostName must not be set when Host already equals the hostname")

	// No User directive — the sandbox id travels in the destination
	// (`ssh <id>@<gateway>`).
	assert.NotContains(t, block, "\n    User ", "User directive must not be set")

	// Per the design (mirrors the workspace UI's "First time setup?"
	// snippet), we deliberately do NOT set StrictHostKeyChecking,
	// UserKnownHostsFile, or LogLevel — IDE Remote-SSH should behave
	// like a normal `ssh <host>` invocation (TOFU host keys, default
	// log level).
	for _, forbidden := range []string{
		"StrictHostKeyChecking",
		"UserKnownHostsFile",
		"LogLevel",
	} {
		assert.NotContains(t, block, forbidden, "must not set %q — SSH defaults are correct", forbidden)
	}
}

// Two registered workspaces in different regions must each get their
// own Host stanza so editor Remote-SSH for one doesn't silently break
// when the user registers against the other.
func TestBuildSSHConfigBlockMultipleGateways(t *testing.T) {
	block := buildSSHConfigBlock(
		"/home/u/.ssh/sandbox_ed25519",
		[]string{"uw2.dbrx.dev", "ue1.dbrx.dev"},
		"2222",
	)
	assert.Contains(t, block, "Host uw2.dbrx.dev\n")
	assert.Contains(t, block, "Host ue1.dbrx.dev\n")
	// Each Host should be followed by Port + IdentityFile, so a naive
	// "Port 2222" count is a quick sanity check on per-gateway repetition.
	assert.Equal(t, 2, strings.Count(block, "Port 2222"))
	assert.Equal(t, 2, strings.Count(block, "IdentityFile /home/u/.ssh/sandbox_ed25519"))
}

func TestWriteManagedConfigCreatesWithRightPerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, sshIncludeFileName)
	require.NoError(t, writeManagedConfig(path, "Host foo\n    HostName bar\n"))

	info, err := os.Stat(path)
	require.NoError(t, err)

	// Windows does not honor Unix permission bits; os.Stat reports 0o666
	// regardless of what was passed to os.WriteFile (matches the carve-out
	// in state_test.go).
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "SSH config files must be 0600")
	}
}

func TestWriteManagedConfigIdempotentDoesNotRewriteFile(t *testing.T) {
	// A no-op re-write must not bump the file's mtime — otherwise
	// `sandbox register` looks like it's churning files on every call.
	dir := t.TempDir()
	path := filepath.Join(dir, sshIncludeFileName)
	content := "Host foo\n    HostName bar\n"
	require.NoError(t, writeManagedConfig(path, content))
	before, err := os.Stat(path)
	require.NoError(t, err)

	require.NoError(t, writeManagedConfig(path, content))
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, before.ModTime(), after.ModTime(), "matching content must short-circuit before rename")
}

func TestWriteManagedConfigReplacesDifferentContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, sshIncludeFileName)
	require.NoError(t, writeManagedConfig(path, "v1\n"))
	require.NoError(t, writeManagedConfig(path, "v2\n"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "v2\n", string(data))
}

func TestEnsureMainIncludesManagedCreatesFileFromScratch(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config")
	managedPath := filepath.Join(dir, sshIncludeFileName)

	require.NoError(t, ensureMainIncludesManaged(mainPath, managedPath))

	data, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	text := string(data)
	assert.Contains(t, text, sshConfigBeginMarker)
	assert.Contains(t, text, "Include "+managedPath)
	assert.Contains(t, text, sshConfigEndMarker)
}

func TestEnsureMainIncludesManagedIsIdempotent(t *testing.T) {
	// Second call must be a no-op — the existing block is detected by
	// marker presence and we leave the file alone.
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config")
	managedPath := filepath.Join(dir, sshIncludeFileName)
	require.NoError(t, ensureMainIncludesManaged(mainPath, managedPath))
	first, err := os.ReadFile(mainPath)
	require.NoError(t, err)

	require.NoError(t, ensureMainIncludesManaged(mainPath, managedPath))
	second, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
}

func TestEnsureMainIncludesManagedPreservesUserConfigBelow(t *testing.T) {
	// Existing user config must survive verbatim, and our block must
	// land *above* it so SSH's first-match-wins behaviour picks our
	// values for the alias.
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config")
	managedPath := filepath.Join(dir, sshIncludeFileName)
	userContent := "Host myserver\n    HostName example.test\n    User alice\n"
	require.NoError(t, os.WriteFile(mainPath, []byte(userContent), 0o600))

	require.NoError(t, ensureMainIncludesManaged(mainPath, managedPath))
	data, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	text := string(data)
	assert.Contains(t, text, userContent, "user's existing config must be preserved")

	beginIdx := strings.Index(text, sshConfigBeginMarker)
	userIdx := strings.Index(text, "Host myserver")
	require.GreaterOrEqual(t, beginIdx, 0)
	require.GreaterOrEqual(t, userIdx, 0)
	assert.Less(t, beginIdx, userIdx, "managed block must precede the user's existing config")
}

// When state has no cached gateway hosts (no register has populated
// the cache, or sandbox.json is missing), maybeWriteSSHConfig must
// short-circuit before any disk I/O so it can't accidentally write a
// malformed `Host \n` stanza. Also pins the no-disk-touch guarantee
// against a future refactor that might `os.MkdirAll` before the check.
func TestMaybeWriteSSHConfigSkipsWhenNoGatewaysCached(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText,
		io.NopCloser(strings.NewReader("")), io.Discard, io.Discard, "", ""))

	require.NoError(t, maybeWriteSSHConfig(ctx, filepath.Join(home, ".ssh", "sandbox_ed25519")))

	// No managed file should have been created.
	_, err := os.Stat(filepath.Join(home, ".ssh", sshIncludeFileName))
	assert.ErrorIs(t, err, fs.ErrNotExist, "managed include file must not be created when no gateways are cached")

	// And ~/.ssh/config must not have been touched either.
	_, err = os.Stat(filepath.Join(home, ".ssh", "config"))
	assert.ErrorIs(t, err, fs.ErrNotExist, "main ssh config must not be created when no gateways are cached")
}

func TestHasOurMarkedBlock(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", false},
		{"only user content", "Host foo\n    HostName bar\n", false},
		{"with our marker", sshConfigBeginMarker + "\nInclude /tmp/x\n" + sshConfigEndMarker + "\n", true},
		{"marker with surrounding whitespace", "   " + sshConfigBeginMarker + "   \n", true},
		// A user could conceivably write a comment containing the
		// literal marker text — accepted; it's their own footgun.
		{"marker mid-line (rejected)", "## " + sshConfigBeginMarker + " inline\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, hasOurMarkedBlock(tc.input))
		})
	}
}
