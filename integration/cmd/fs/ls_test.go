package fs_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"strings"
	"testing"

	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fsTest struct {
	name       string
	setupFiler func(t testutil.TestingT) (filer.Filer, string)
}

var fsTests = []fsTest{
	{
		name:       "dbfs",
		setupFiler: setupDbfsFiler,
	},
	{
		name:       "uc-volumes",
		setupFiler: setupUcVolumesFiler,
	},
}

func setupLsFiles(t *testing.T, f filer.Filer) {
	err := f.Write(context.Background(), "a/hello.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)
	err = f.Write(context.Background(), "bye.txt", strings.NewReader("def"))
	require.NoError(t, err)
}

func TestFsLs(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)
			setupLsFiles(t, f)

			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "ls", tmpDir, "--output=json")
			assert.Equal(t, "", stderr.String())

			var parsedStdout []map[string]any
			err := json.Unmarshal(stdout.Bytes(), &parsedStdout)
			require.NoError(t, err)

			// assert on ls output
			assert.Len(t, parsedStdout, 2)

			assert.Equal(t, "a", parsedStdout[0]["name"])
			assert.Equal(t, true, parsedStdout[0]["is_directory"])
			assert.InDelta(t, float64(0), parsedStdout[0]["size"], 0.0001)

			assert.Equal(t, "bye.txt", parsedStdout[1]["name"])
			assert.Equal(t, false, parsedStdout[1]["is_directory"])
			assert.InDelta(t, float64(3), parsedStdout[1]["size"], 0.0001)
		})
	}
}

func TestFsLsWithAbsolutePaths(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)
			setupLsFiles(t, f)

			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "ls", tmpDir, "--output=json", "--absolute")
			assert.Equal(t, "", stderr.String())

			var parsedStdout []map[string]any
			err := json.Unmarshal(stdout.Bytes(), &parsedStdout)
			require.NoError(t, err)

			// assert on ls output
			assert.Len(t, parsedStdout, 2)

			assert.Equal(t, path.Join(tmpDir, "a"), parsedStdout[0]["name"])
			assert.Equal(t, true, parsedStdout[0]["is_directory"])
			assert.InDelta(t, float64(0), parsedStdout[0]["size"], 0.0001)

			assert.Equal(t, path.Join(tmpDir, "bye.txt"), parsedStdout[1]["name"])
			assert.Equal(t, false, parsedStdout[1]["is_directory"])
			assert.InDelta(t, float64(3), parsedStdout[1]["size"].(float64), 0.0001)
		})
	}
}

func TestFsLsOnFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			f, tmpDir := testCase.setupFiler(t)
			setupLsFiles(t, f)

			_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "ls", path.Join(tmpDir, "a", "hello.txt"), "--output=json")
			assert.Regexp(t, "not a directory: .*/a/hello.txt", err.Error())
			assert.ErrorAs(t, err, &filer.NotADirectory{})
		})
	}
}

func TestFsLsOnEmptyDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			_, tmpDir := testCase.setupFiler(t)

			stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "fs", "ls", tmpDir, "--output=json")
			assert.Equal(t, "", stderr.String())
			var parsedStdout []map[string]any
			err := json.Unmarshal(stdout.Bytes(), &parsedStdout)
			require.NoError(t, err)

			// assert on ls output
			assert.Empty(t, parsedStdout)
		})
	}
}

func TestFsLsForNonexistingDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range fsTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			_, tmpDir := testCase.setupFiler(t)

			_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "ls", path.Join(tmpDir, "nonexistent"), "--output=json")
			assert.ErrorIs(t, err, fs.ErrNotExist)
			assert.Regexp(t, "no such directory: .*/nonexistent", err.Error())
		})
	}
}

func TestFsLsWithoutScheme(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, _, err := testcli.RequireErrorRun(t, ctx, "fs", "ls", "/path-without-a-dbfs-scheme", "--output=json")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
