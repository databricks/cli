package postgrescmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemp(t *testing.T, name, contents string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(contents), 0o644))
	return p
}

func TestCollectInputs_PositionalOnly(t *testing.T) {
	units, err := collectInputs(t.Context(), strings.NewReader(""), []string{"SELECT 1"}, nil)
	require.NoError(t, err)
	require.Len(t, units, 1)
	assert.Equal(t, "SELECT 1", units[0].SQL)
	assert.Equal(t, "argv[1]", units[0].Source)
}

func TestCollectInputs_MultiplePositionals(t *testing.T) {
	units, err := collectInputs(t.Context(), strings.NewReader(""), []string{"SELECT 1", "SELECT 2"}, nil)
	require.NoError(t, err)
	require.Len(t, units, 2)
	assert.Equal(t, "SELECT 1", units[0].SQL)
	assert.Equal(t, "SELECT 2", units[1].SQL)
}

func TestCollectInputs_FileOnly(t *testing.T) {
	p := writeTemp(t, "q.sql", "SELECT * FROM t")
	units, err := collectInputs(t.Context(), strings.NewReader(""), nil, []string{p})
	require.NoError(t, err)
	require.Len(t, units, 1)
	assert.Equal(t, "SELECT * FROM t", units[0].SQL)
	assert.Contains(t, units[0].Source, "--file")
}

func TestCollectInputs_FilesFirstThenPositionals(t *testing.T) {
	p1 := writeTemp(t, "a.sql", "SELECT 1")
	p2 := writeTemp(t, "b.sql", "SELECT 2")
	units, err := collectInputs(t.Context(), strings.NewReader(""), []string{"SELECT 3"}, []string{p1, p2})
	require.NoError(t, err)
	require.Len(t, units, 3)
	assert.Equal(t, "SELECT 1", units[0].SQL)
	assert.Equal(t, "SELECT 2", units[1].SQL)
	assert.Equal(t, "SELECT 3", units[2].SQL)
}

func TestCollectInputs_DotSQLAutoDetect(t *testing.T) {
	p := writeTemp(t, "data.sql", "SELECT 42")
	units, err := collectInputs(t.Context(), strings.NewReader(""), []string{p}, nil)
	require.NoError(t, err)
	require.Len(t, units, 1)
	assert.Equal(t, "SELECT 42", units[0].SQL)
}

func TestCollectInputs_DotSQLNotExistingFallsThroughToLiteral(t *testing.T) {
	units, err := collectInputs(t.Context(), strings.NewReader(""), []string{"/nonexistent/path.sql"}, nil)
	require.NoError(t, err)
	require.Len(t, units, 1)
	assert.Equal(t, "/nonexistent/path.sql", units[0].SQL)
}

func TestCollectInputs_StdinOnly(t *testing.T) {
	units, err := collectInputs(t.Context(), strings.NewReader("SELECT 1\n"), nil, nil)
	require.NoError(t, err)
	require.Len(t, units, 1)
	assert.Equal(t, "SELECT 1", units[0].SQL)
	assert.Equal(t, "stdin", units[0].Source)
}

func TestCollectInputs_StdinIgnoredWhenPositionalsPresent(t *testing.T) {
	units, err := collectInputs(t.Context(), strings.NewReader("FROM STDIN"), []string{"SELECT 1"}, nil)
	require.NoError(t, err)
	require.Len(t, units, 1)
	assert.Equal(t, "SELECT 1", units[0].SQL)
}

func TestCollectInputs_EmptyStdinErrors(t *testing.T) {
	_, err := collectInputs(t.Context(), strings.NewReader(""), nil, nil)
	assert.ErrorContains(t, err, "no SQL provided")
}

func TestCollectInputs_EmptyFileErrors(t *testing.T) {
	p := writeTemp(t, "empty.sql", "")
	_, err := collectInputs(t.Context(), strings.NewReader(""), nil, []string{p})
	assert.ErrorContains(t, err, "is empty")
}

func TestCollectInputs_EmptyPositional(t *testing.T) {
	_, err := collectInputs(t.Context(), strings.NewReader(""), []string{"  "}, nil)
	assert.ErrorContains(t, err, "is empty")
}
