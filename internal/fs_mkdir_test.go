package internal

import (
	"context"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccFsMkdir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, tmpDir := tc.setupFiler(t)

			// create directory "a"
			stdout, stderr := RequireSuccessfulRun(t, "fs", "mkdir", path.Join(tmpDir, "a"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// assert directory "a" is created
			info, err := f.Stat(context.Background(), "a")
			require.NoError(t, err)
			assert.Equal(t, "a", info.Name())
			assert.Equal(t, true, info.IsDir())
		})
	}
}

func TestAccFsMkdirCreatesIntermediateDirectories(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, tmpDir := tc.setupFiler(t)

			// create directory "a/b/c"
			stdout, stderr := RequireSuccessfulRun(t, "fs", "mkdir", path.Join(tmpDir, "a", "b", "c"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())

			// assert directory "a" is created
			infoA, err := f.Stat(context.Background(), "a")
			require.NoError(t, err)
			assert.Equal(t, "a", infoA.Name())
			assert.Equal(t, true, infoA.IsDir())

			// assert directory "b" is created
			infoB, err := f.Stat(context.Background(), "a/b")
			require.NoError(t, err)
			assert.Equal(t, "b", infoB.Name())
			assert.Equal(t, true, infoB.IsDir())

			// assert directory "c" is created
			infoC, err := f.Stat(context.Background(), "a/b/c")
			require.NoError(t, err)
			assert.Equal(t, "c", infoC.Name())
			assert.Equal(t, true, infoC.IsDir())
		})
	}
}

func TestAccFsMkdirWhenDirectoryAlreadyExists(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		tc := testCase

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, tmpDir := tc.setupFiler(t)

			// create directory "a"
			err := f.Mkdir(context.Background(), "a")
			require.NoError(t, err)

			// assert run is successful without any errors
			stdout, stderr := RequireSuccessfulRun(t, "fs", "mkdir", path.Join(tmpDir, "a"))
			assert.Equal(t, "", stderr.String())
			assert.Equal(t, "", stdout.String())
		})
	}
}

func TestAccFsMkdirWhenFileExistsAtPath(t *testing.T) {
	t.Parallel()

	t.Run("dbfs", func(t *testing.T) {
		t.Parallel()

		f, tmpDir := setupDbfsFiler(t)

		// create file "hello"
		err := f.Write(context.Background(), "hello", strings.NewReader("abc"))
		require.NoError(t, err)

		// assert mkdir fails
		_, _, err = RequireErrorRun(t, "fs", "mkdir", path.Join(tmpDir, "hello"))

		// Different cloud providers return different errors.
		regex := regexp.MustCompile(`(^|: )Path is a file: .*$|(^|: )Cannot create directory .* because .* is an existing file\.$|(^|: )mkdirs\(hadoopPath: .*, permission: rwxrwxrwx\): failed$`)
		assert.Regexp(t, regex, err.Error())
	})

	t.Run("uc-volumes", func(t *testing.T) {
		t.Parallel()

		f, tmpDir := setupUcVolumesFiler(t)

		// create file "hello"
		err := f.Write(context.Background(), "hello", strings.NewReader("abc"))
		require.NoError(t, err)

		// assert mkdir fails
		_, _, err = RequireErrorRun(t, "fs", "mkdir", path.Join(tmpDir, "hello"))

		assert.ErrorAs(t, err, &filer.FileAlreadyExistsError{})
	})
}
