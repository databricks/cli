package lakebox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSSHConfigBlockShape(t *testing.T) {
	block := buildSSHConfigBlock("/home/u/.ssh/lakebox_ed25519", "uw2.dbrx.dev", "2222")

	// Header marker so users know not to hand-edit, and the alias name
	// that the UI's "First time setup?" disclosure documents.
	assert.Contains(t, block, "# Managed by")
	assert.Contains(t, block, "Host "+sshConfigAlias+"\n")
	assert.Contains(t, block, "HostName uw2.dbrx.dev")
	assert.Contains(t, block, "Port 2222")
	assert.Contains(t, block, "IdentityFile /home/u/.ssh/lakebox_ed25519")

	// Mirrors buildSSHArgs in ssh.go so connections through this alias
	// behave identically to `databricks lakebox ssh`. If those change,
	// this list should change too.
	for _, expect := range []string{
		"IdentitiesOnly yes",
		"StrictHostKeyChecking no",
		"UserKnownHostsFile /dev/null",
		"LogLevel ERROR",
	} {
		assert.Contains(t, block, expect, "missing required SSH option %q", expect)
	}

	// No User directive — the sandbox id travels in the destination
	// (`ssh <id>@lakebox-gw`), so one alias serves every sandbox.
	assert.NotContains(t, block, "\n    User ", "User directive must not be set")
}

func TestWriteManagedConfigCreatesWithRightPerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, sshIncludeFileName)
	require.NoError(t, writeManagedConfig(path, "Host foo\n    HostName bar\n"))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "SSH config files must be 0600")
}

func TestWriteManagedConfigIdempotentDoesNotRewriteFile(t *testing.T) {
	// A no-op re-write must not bump the file's mtime — otherwise
	// `lakebox register` looks like it's churning files on every call.
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
