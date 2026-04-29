package deployment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrate_HelpTextIsUcmFlavored guards against accidental "bundle" leaks
// in the migrate verb's user-facing prose. The full bundle source mentions
// "bundle" in its help text; the ucm fork must reword every occurrence so
// `databricks ucm deployment migrate --help` reads cleanly.
func TestMigrate_HelpTextIsUcmFlavored(t *testing.T) {
	cmd := newMigrateCommand()

	short := cmd.Short
	long := cmd.Long

	// Short summary should describe the engine migration without the
	// "bundle" framing.
	assert.NotEmpty(t, short)
	assert.NotContains(t, strings.ToLower(short), "bundle")

	// Long help should not advertise `bundle deploy` / `bundle plan`.
	require.NotEmpty(t, long)
	assert.NotContains(t, long, "bundle deploy")
	assert.NotContains(t, long, "bundle plan")
	// Sanity: it should mention the ucm flow.
	assert.Contains(t, long, "ucm deploy")
}

// TestMigrate_NoPlanCheckFlag locks in the --noplancheck flag so a future
// refactor doesn't silently drop it. The flag is the user's escape hatch
// when the spawned `ucm plan` invocation can't be reproduced cleanly
// (e.g. CI that already vetted the deploy).
func TestMigrate_NoPlanCheckFlag(t *testing.T) {
	cmd := newMigrateCommand()
	flag := cmd.Flags().Lookup("noplancheck")
	require.NotNil(t, flag, "migrate must expose --noplancheck")
	assert.Equal(t, "false", flag.DefValue)
}

// TestReadTerraformStateHeader_MissingFile asserts we surface the OS error
// (rather than synthesizing a noop) when there is no local terraform state
// to migrate from. The migrate verb relies on this distinction to print the
// "no existing local state" guidance instead of a generic parse error.
func TestReadTerraformStateHeader_MissingFile(t *testing.T) {
	_, err := readTerraformStateHeader(filepath.Join(t.TempDir(), "missing.tfstate"))
	require.Error(t, err)
}

// TestReadTerraformStateHeader_Parses verifies the lineage + serial
// extraction matches the on-disk tfstate JSON shape — the same shape
// terraform itself emits and bundle's StateDesc reads.
func TestReadTerraformStateHeader_Parses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "terraform.tfstate")
	body := `{"version":4,"lineage":"abc-123","serial":7,"resources":[]}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))

	hdr, err := readTerraformStateHeader(path)
	require.NoError(t, err)
	assert.Equal(t, "abc-123", hdr.Lineage)
	assert.Equal(t, 7, hdr.Serial)
}

// TestReadTerraformStateHeader_RejectsMalformed surfaces a clear parse
// error rather than letting a corrupt tfstate slip through and produce a
// blank-lineage migrated database (which would mint a fresh lineage and
// silently break terraform-side rollback).
func TestReadTerraformStateHeader_RejectsMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "terraform.tfstate")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	_, err := readTerraformStateHeader(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}
