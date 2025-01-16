package fs_test

import (
	"context"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsMkdir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)

			// create directory "a"
			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "mkdir", path.Join(tmpDir, "a"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// assert directory "a" is created
			info, err := f.Stat(context.Background(), "a")
			require.NoError(t, err)
			assert.Equal(t, "a", info.Name())
			assert.True(t, info.IsDir())
		})
	}
}

func TestFsMkdirCreatesIntermediateDirectories(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)

			// create directory "a/b/c"
			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "mkdir", path.Join(tmpDir, "a", "b", "c"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// assert directory "a" is created
			infoA, err := f.Stat(context.Background(), "a")
			require.NoError(t, err)
			assert.Equal(t, "a", infoA.Name())
			assert.True(t, infoA.IsDir())

			// assert directory "b" is created
			infoB, err := f.Stat(context.Background(), "a/b")
			require.NoError(t, err)
			assert.Equal(t, "b", infoB.Name())
			assert.True(t, infoB.IsDir())

			// assert directory "c" is created
			infoC, err := f.Stat(context.Background(), "a/b/c")
			require.NoError(t, err)
			assert.Equal(t, "c", infoC.Name())
			assert.True(t, infoC.IsDir())
		})
	}
}

func TestFsMkdirWhenDirectoryAlreadyExists(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)

			// create directory "a"
			err := f.Mkdir(context.Background(), "a")
			require.NoError(t, err)

			// assert run is successful without any errors
			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "mkdir", path.Join(tmpDir, "a"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())
		})
	}
}

func TestFsMkdirWhenFileExistsAtPath(t *testing.T) {
	t.Parallel()

	t.Run("dbfs", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		f, tmpDir := setupDbfsFiler(t)

		// create file "hello"
		err := f.Write(context.Background(), "hello", strings.NewReader("abc"))
		require.NoError(t, err)

		// assert mkdir fails
		_, _, err = testcli.RequireErrorRun(t, ctx, "fs", "mkdir", path.Join(tmpDir, "hello"))

		// Different cloud providers or cloud configurations return different errors.
		regex := regexp.MustCompile(`(^|: )Path is a file: .*$|(^|: )Cannot create directory .* because .* is an existing file\.$|(^|: )mkdirs\(hadoopPath: .*, permission: rwxrwxrwx\): failed$|(^|: )"The specified path already exists.".*$`)
		assert.Regexp(t, regex, err.Error())
	})

	t.Run("uc-volumes", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		f, tmpDir := setupUcVolumesFiler(t)

		// create file "hello"
		err := f.Write(context.Background(), "hello", strings.NewReader("abc"))
		require.NoError(t, err)

		// assert mkdir fails
		_, _, err = testcli.RequireErrorRun(t, ctx, "fs", "mkdir", path.Join(tmpDir, "hello"))

		assert.ErrorAs(t, err, &filer.FileAlreadyExistsError{})
	})
}
