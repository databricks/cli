package fs_test

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsCat(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			f, tmpDir := testCase.setupFiler(t)

			err := f.Write(t.Context(), "hello.txt", strings.NewReader("abcd"), filer.CreateParentDirectories)
			require.NoError(t, err)

			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "cat", path.Join(tmpDir, "hello.txt"))
			assert.Empty(t, stderr.String())
			assert.Equal(t, "abcd", stdout.String())
		})
	}
}

func TestFsCatOnADir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			f, tmpDir := testCase.setupFiler(t)

			err := f.Mkdir(t.Context(), "dir1")
			require.NoError(t, err)

			_, _, err = testcli.RequireErrorRun(t, ctx, "fs", "cat", path.Join(tmpDir, "dir1"))
			assert.ErrorIs(t, err, fs.ErrInvalid)
		})
	}
}

func TestFsCatOnNonExistentFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			_, tmpDir := testCase.setupFiler(t)

			_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "cat", path.Join(tmpDir, "non-existent-file"))
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}
