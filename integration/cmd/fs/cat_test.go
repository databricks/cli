package fs_test

import (
	"context"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
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

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)

			err := f.Write(context.Background(), "hello.txt", strings.NewReader("abcd"), filer.CreateParentDirectories)
			require.NoError(t, err)

			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "cat", path.Join(tmpDir, "hello.txt"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "abcd", stdout.String())
		})
	}
}

func TestFsCatOnADir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)

			err := f.Mkdir(context.Background(), "dir1")
			require.NoError(t, err)

			_, _, err = testcli.RequireErrorRun(t, ctx, "fs", "cat", path.Join(tmpDir, "dir1"))
			assert.ErrorAs(t, err, &filer.NotAFile{})
		})
	}
}

func TestFsCatOnNonExistentFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			_, tmpDir := testCase.setupFiler(t)

			_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "cat", path.Join(tmpDir, "non-existent-file"))
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

func TestFsCatForDbfsInvalidScheme(t *testing.T) {
	ctx := context.Background()
	_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "cat", "dab:/non-existent-file")
	assert.ErrorContains(t, err, "invalid scheme: dab")
}

func TestFsCatDoesNotSupportOutputModeJson(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	tmpDir := acc.TemporaryDbfsDir(wt, "fs-cat-")
	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	err = f.Write(ctx, "hello.txt", strings.NewReader("abc"))
	require.NoError(t, err)

	_, _, err = testcli.RequireErrorRun(t, ctx, "fs", "cat", "dbfs:"+path.Join(tmpDir, "hello.txt"), "--output=json")
	assert.ErrorContains(t, err, "json output not supported")
}
