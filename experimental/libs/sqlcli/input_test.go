package sqlcli

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

func TestCollect_PositionalOnly(t *testing.T) {
	got, err := Collect(t.Context(), strings.NewReader(""), []string{"SELECT 1"}, nil, CollectOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "SELECT 1", got[0].SQL)
	assert.Equal(t, "argv[1]", got[0].Source)
}

func TestCollect_MultiplePositionals(t *testing.T) {
	got, err := Collect(t.Context(), strings.NewReader(""), []string{"SELECT 1", "SELECT 2"}, nil, CollectOptions{})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "SELECT 1", got[0].SQL)
	assert.Equal(t, "SELECT 2", got[1].SQL)
}

func TestCollect_FileOnly(t *testing.T) {
	p := writeTemp(t, "q.sql", "SELECT * FROM t")
	got, err := Collect(t.Context(), strings.NewReader(""), nil, []string{p}, CollectOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "SELECT * FROM t", got[0].SQL)
	assert.Contains(t, got[0].Source, "--file")
}

func TestCollect_FilesFirstThenPositionals(t *testing.T) {
	p1 := writeTemp(t, "a.sql", "SELECT 1")
	p2 := writeTemp(t, "b.sql", "SELECT 2")
	got, err := Collect(t.Context(), strings.NewReader(""), []string{"SELECT 3"}, []string{p1, p2}, CollectOptions{})
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "SELECT 1", got[0].SQL)
	assert.Equal(t, "SELECT 2", got[1].SQL)
	assert.Equal(t, "SELECT 3", got[2].SQL)
}

func TestCollect_DotSQLAutoDetect(t *testing.T) {
	p := writeTemp(t, "data.sql", "SELECT 42")
	got, err := Collect(t.Context(), strings.NewReader(""), []string{p}, nil, CollectOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "SELECT 42", got[0].SQL)
}

func TestCollect_DotSQLNotExistingFallsThroughToLiteral(t *testing.T) {
	got, err := Collect(t.Context(), strings.NewReader(""), []string{"/nonexistent/path.sql"}, nil, CollectOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "/nonexistent/path.sql", got[0].SQL)
}

func TestCollect_StdinOnly(t *testing.T) {
	got, err := Collect(t.Context(), strings.NewReader("SELECT 1\n"), nil, nil, CollectOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "SELECT 1", got[0].SQL)
	assert.Equal(t, "stdin", got[0].Source)
}

func TestCollect_StdinIgnoredWhenPositionalsPresent(t *testing.T) {
	got, err := Collect(t.Context(), strings.NewReader("FROM STDIN"), []string{"SELECT 1"}, nil, CollectOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "SELECT 1", got[0].SQL)
}

func TestCollect_EmptyStdinErrors(t *testing.T) {
	_, err := Collect(t.Context(), strings.NewReader(""), nil, nil, CollectOptions{})
	assert.ErrorContains(t, err, "no SQL provided")
}

func TestCollect_EmptyFileErrors(t *testing.T) {
	p := writeTemp(t, "empty.sql", "")
	_, err := Collect(t.Context(), strings.NewReader(""), nil, []string{p}, CollectOptions{})
	assert.ErrorContains(t, err, "is empty")
}

func TestCollect_EmptyPositional(t *testing.T) {
	_, err := Collect(t.Context(), strings.NewReader(""), []string{"  "}, nil, CollectOptions{})
	assert.ErrorContains(t, err, "is empty")
}

func TestCollect_CustomCleanerStripsComments(t *testing.T) {
	cleaner := func(s string) string {
		// Naive comment stripper: drop lines starting with --
		var lines []string
		for line := range strings.SplitSeq(s, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "--") {
				lines = append(lines, line)
			}
		}
		return strings.Join(lines, "\n")
	}
	got, err := Collect(
		t.Context(), strings.NewReader(""),
		[]string{"-- ignored\nSELECT 1\n-- also ignored"},
		nil,
		CollectOptions{Cleaner: cleaner},
	)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "SELECT 1", got[0].SQL)
}
