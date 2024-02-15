package internal

import (
	"context"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccFsRmFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a file
			f, tmpDir := tc.setupFiler(t)
			err := f.Write(context.Background(), "hello.txt", strings.NewReader("abcd"), filer.CreateParentDirectories)
			require.NoError(t, err)

			// Check file was created
			_, err = f.Stat(context.Background(), "hello.txt")
			assert.NoError(t, err)

			// Run rm command
			stdout, stderr := RequireSuccessfulRun(t, "fs", "rm", path.Join(tmpDir, "hello.txt"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// Assert file was deleted
			_, err = f.Stat(context.Background(), "hello.txt")
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

// TODO: Check-in with the files team whether the integration testing should
// be limited to a single schema / volume.
func TestAccFsRmEmptyDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a directory
			f, tmpDir := tc.setupFiler(t)
			err := f.Mkdir(context.Background(), "a")
			require.NoError(t, err)

			// Check directory was created
			_, err = f.Stat(context.Background(), "a")
			assert.NoError(t, err)

			// Run rm command
			stdout, stderr := RequireSuccessfulRun(t, "fs", "rm", path.Join(tmpDir, "a"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// Assert directory was deleted
			_, err = f.Stat(context.Background(), "a")
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

func TestAccFsRmNonEmptyDirectory(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a directory
			f, tmpDir := tc.setupFiler(t)
			err := f.Mkdir(context.Background(), "a")
			require.NoError(t, err)

			// Create a file in the directory
			err = f.Write(context.Background(), "a/hello.txt", strings.NewReader("abcd"), filer.CreateParentDirectories)
			require.NoError(t, err)

			// Check file was created
			_, err = f.Stat(context.Background(), "a/hello.txt")
			assert.NoError(t, err)

			// Run rm command
			_, _, err = RequireErrorRun(t, "fs", "rm", path.Join(tmpDir, "a"))
			assert.ErrorIs(t, err, fs.ErrInvalid)
			assert.ErrorAs(t, err, &filer.DirectoryNotEmptyError{})
		})
	}
}

func TestAccFsRmForNonExistentFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tmpDir := tc.setupFiler(t)

			// Expect error if file does not exist
			_, _, err := RequireErrorRun(t, "fs", "rm", path.Join(tmpDir, "does-not-exist"))
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}

}

func TestAccFsRmDirRecursively(t *testing.T) {
	t.Parallel()

	t.Run("dbfs", func(t *testing.T) {
		t.Parallel()

		f, tmpDir := setupDbfsFiler(t)

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
		stdout, stderr := RequireSuccessfulRun(t, "fs", "rm", path.Join(tmpDir, "a"), "--recursive")
		assert.Equal(t, "", stderr.String())
		assert.Equal(t, "", stdout.String())

		// Assert directory was deleted
		_, err = f.Stat(context.Background(), "a")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("uc-volumes", func(t *testing.T) {
		t.Parallel()

		f, tmpDir := setupUcVolumesFiler(t)

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
		_, _, err = RequireErrorRun(t, "fs", "rm", path.Join(tmpDir, "a"), "--recursive")
		assert.ErrorContains(t, err, "files API does not support recursive delete")
	})
}
