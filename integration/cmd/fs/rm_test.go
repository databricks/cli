package fs_test

import (
	"context"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsRmFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Create a file
			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)
			err := f.Write(context.Background(), "hello.txt", strings.NewReader("abcd"), filer.CreateParentDirectories)
			require.NoError(t, err)

			// Check file was created
			_, err = f.Stat(context.Background(), "hello.txt")
			assert.NoError(t, err)

			// Run rm command
			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "rm", path.Join(tmpDir, "hello.txt"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// Assert file was deleted
			_, err = f.Stat(context.Background(), "hello.txt")
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

func TestFsRmEmptyDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Create a directory
			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)
			err := f.Mkdir(context.Background(), "a")
			require.NoError(t, err)

			// Check directory was created
			_, err = f.Stat(context.Background(), "a")
			assert.NoError(t, err)

			// Run rm command
			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "rm", path.Join(tmpDir, "a"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// Assert directory was deleted
			_, err = f.Stat(context.Background(), "a")
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

func TestFsRmNonEmptyDirectory(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Create a directory
			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)
			err := f.Mkdir(context.Background(), "a")
			require.NoError(t, err)

			// Create a file in the directory
			err = f.Write(context.Background(), "a/hello.txt", strings.NewReader("abcd"), filer.CreateParentDirectories)
			require.NoError(t, err)

			// Check file was created
			_, err = f.Stat(context.Background(), "a/hello.txt")
			assert.NoError(t, err)

			// Run rm command
			_, _, err = testcli.RequireErrorRun(t, ctx, "fs", "rm", path.Join(tmpDir, "a"))
			assert.ErrorIs(t, err, fs.ErrInvalid)
			assert.ErrorAs(t, err, &filer.DirectoryNotEmptyError{})
		})
	}
}

func TestFsRmForNonExistentFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			_, tmpDir := testCase.setupFiler(t)

			// Expect error if file does not exist
			_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "rm", path.Join(tmpDir, "does-not-exist"))
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

func TestFsRmDirRecursively(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)

			// Create a directory
			err := f.Mkdir(context.Background(), "a")
			require.NoError(t, err)

			// Create a file in the directory
			err = f.Write(context.Background(), "a/hello.txt", strings.NewReader("abcd"), filer.CreateParentDirectories)
			require.NoError(t, err)

			// Check file was created
			_, err = f.Stat(context.Background(), "a/hello.txt")
			assert.NoError(t, err)

			// Run rm command
			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "rm", path.Join(tmpDir, "a"), "--recursive")
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// Assert directory was deleted
			_, err = f.Stat(context.Background(), "a")
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}
