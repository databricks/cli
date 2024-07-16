package internal

import (
	"context"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccFsCat(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, tmpDir := tc.setupFiler(t)
			err := f.Write(context.Background(), "hello.txt", strings.NewReader("abcd"), filer.CreateParentDirectories)
			require.NoError(t, err)

			stdout, stderr := RequireSuccessfulRun(t, "fs", "cat", path.Join(tmpDir, "hello.txt"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "abcd", stdout.String())
		})
	}
}

func TestAccFsCatOnADir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, tmpDir := tc.setupFiler(t)
			err := f.Mkdir(context.Background(), "dir1")
			require.NoError(t, err)

			_, _, err = RequireErrorRun(t, "fs", "cat", path.Join(tmpDir, "dir1"))
			assert.ErrorAs(t, err, &filer.NotAFile{})
		})
	}
}

func TestAccFsCatOnNonExistentFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tmpDir := tc.setupFiler(t)

			_, _, err := RequireErrorRun(t, "fs", "cat", path.Join(tmpDir, "non-existent-file"))
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

func TestAccFsCatForDbfsInvalidScheme(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "cat", "dab:/non-existent-file")
	assert.ErrorContains(t, err, "invalid scheme: dab")
}

func TestAccFsCatDoesNotSupportOutputModeJson(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := TemporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	err = f.Write(ctx, "hello.txt", strings.NewReader("abc"))
	require.NoError(t, err)

	_, _, err = RequireErrorRun(t, "fs", "cat", "dbfs:"+path.Join(tmpDir, "hello.txt"), "--output=json")
	assert.ErrorContains(t, err, "json output not supported")
}
