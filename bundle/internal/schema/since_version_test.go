package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeSinceVersionsStoredWins(t *testing.T) {
	// A stored version is authoritative and must not be overwritten by a freshly
	// computed (possibly drifted) value. This is what keeps versions stable when
	// a type is refactored and history re-keys the field to a newer version.
	computed := map[string]string{
		"pkg.Type.field":     "v0.300.0", // drifted forward by a refactor
		"pkg.Type.new_field": "v0.310.0", // genuinely new, not yet stored
	}
	stored := map[string]string{
		"pkg.Type.field": "v0.229.0",
	}

	merged := mergeSinceVersions(computed, stored, nil)

	assert.Equal(t, "v0.229.0", merged["pkg.Type.field"], "stored version must win")
	assert.Equal(t, "v0.310.0", merged["pkg.Type.new_field"], "new field keeps computed version")
}

func TestMergeSinceVersionsAliasInheritsOldVersion(t *testing.T) {
	// A renamed/retyped field whose new key is not yet stored inherits the old
	// key's version instead of being treated as brand new.
	computed := map[string]string{
		"pkg.AppPermission.user_name": "v0.247.0", // when the typed struct appeared
		"pkg.Permission.user_name":    "v0.229.0",
	}
	stored := map[string]string{
		"pkg.Permission.user_name": "v0.229.0",
	}
	aliases := map[string]string{
		"pkg.AppPermission.user_name": "pkg.Permission.user_name",
	}

	merged := mergeSinceVersions(computed, stored, aliases)

	assert.Equal(t, "v0.229.0", merged["pkg.AppPermission.user_name"],
		"renamed field must inherit the original key's version")
}

func TestMergeSinceVersionsAliasSkippedWhenAlreadyFrozen(t *testing.T) {
	// Once the new key is stored, the alias is a no-op: the stored value stands.
	computed := map[string]string{}
	stored := map[string]string{
		"pkg.AppPermission.user_name": "v0.247.0",
		"pkg.Permission.user_name":    "v0.229.0",
	}
	aliases := map[string]string{
		"pkg.AppPermission.user_name": "pkg.Permission.user_name",
	}

	merged := mergeSinceVersions(computed, stored, aliases)

	assert.Equal(t, "v0.247.0", merged["pkg.AppPermission.user_name"],
		"a frozen key must not be rewritten by an alias")
}

func TestStoredSinceVersionsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "since_versions.json")

	want := map[string]string{"a.b": "v0.1.0", "c.d": "v0.2.0"}
	require.NoError(t, saveStoredSinceVersions(path, want))

	got, err := loadStoredSinceVersions(path)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLoadStoredSinceVersionsMissingFile(t *testing.T) {
	// A fresh checkout (or docgen branch without the file yet) must not error —
	// the first run seeds it.
	got, err := loadStoredSinceVersions(filepath.Join(t.TempDir(), "does-not-exist.json"))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestStoredSinceVersionsWriteIsCanonical(t *testing.T) {
	// saveStoredSinceVersions must write sorted keys with a trailing newline so
	// the file committed to docgen stays diff-stable across runs.
	path := filepath.Join(t.TempDir(), "since_versions.json")
	require.NoError(t, saveStoredSinceVersions(path, map[string]string{"b": "v0.2.0", "a": "v0.1.0"}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "{\n  \"a\": \"v0.1.0\",\n  \"b\": \"v0.2.0\"\n}\n", string(got))
}
