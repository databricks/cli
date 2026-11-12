package selection

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommitMessage(t *testing.T) {
	msg := "empty commit to run tests\n" +
		"\n" +
		"aws: TestAccept/bundle/resources/jobs\n" +
		"cloud: TestAccept/bundle/resources/pipelines   \n" +
		"  gcp:TestAccept/a TestAccept/b\t\n" +
		"azure:   TestAccept/c/DMS=true\r\n" +
		"dbr: TestAccept/unknown/prefix\n" +
		"AWS: TestAccept/uppercase/prefix\n" +
		"see aws: TestAccept/not/at/line/start\n" +
		"aws: fix flaky TestAccept retries\n" +
		"Co-authored-by: Someone <someone@example.com>\n" +
		"aws:\n"

	assert.Equal(t, []CommitName{
		{Cloud: "aws", Name: "TestAccept/bundle/resources/jobs"},
		{Cloud: "cloud", Name: "TestAccept/bundle/resources/pipelines"},
		{Cloud: "gcp", Name: "TestAccept/a"},
		{Cloud: "gcp", Name: "TestAccept/b"},
		{Cloud: "azure", Name: "TestAccept/c/DMS=true"},
	}, ParseCommitMessage(msg))
}

func TestParseCommitMessageNoMatches(t *testing.T) {
	assert.Empty(t, ParseCommitMessage(""))
	assert.Empty(t, ParseCommitMessage("Fix a bug\n\nThe aws: prefix is mentioned mid-line only.\n"))
	assert.Empty(t, ParseCommitMessage("cloud: make TestAccept faster\n"))
}

func TestNamesForCloud(t *testing.T) {
	names := []CommitName{
		{Cloud: "aws", Name: "TestAccept/a"},
		{Cloud: "cloud", Name: "TestAccept/b"},
		{Cloud: "gcp", Name: "TestAccept/c"},
		{Cloud: "azure", Name: "TestAccept/d"},
		{Cloud: "gcp", Name: "TestAccept/b"},
	}
	assert.Equal(t, []string{"TestAccept/a", "TestAccept/b"}, NamesForCloud(names, "aws"))
	assert.Equal(t, []string{"TestAccept/b", "TestAccept/c"}, NamesForCloud(names, "gcp"))
	assert.Equal(t, []string{"TestAccept/b", "TestAccept/d"}, NamesForCloud(names, "azure"))
	// A local run selects the tests named for any cloud.
	assert.Equal(t, []string{"TestAccept/a", "TestAccept/b", "TestAccept/c", "TestAccept/d"}, NamesForCloud(names, ""))
	assert.Empty(t, NamesForCloud(nil, "aws"))
}

func TestResolveNames(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cmd", "version"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cmd", "version", "out.test.toml"), []byte("EnvMatrix.DMS = [\"\", \"true\"]\nEnvMatrix.X = [\"1\", \"2\"]\n"), 0o600))

	testDirs := map[string]bool{
		"bundle/resources/jobs":            true,
		"bundle/resources/jobs/update":     true,
		"bundle/resources/jobsx":           true,
		"bundle/resources/pipelines/basic": true,
		"cmd/version":                      true,
	}

	named, unmatched := ResolveNames(root, []string{
		// A dir and everything under it, but not a sibling that shares the prefix.
		"TestAccept/bundle/resources/jobs",
		// A path that is not itself a test dir selects the dirs under it.
		"TestAccept/bundle/resources/pipelines/",
		// A variant of a dir.
		"TestAccept/cmd/version/DMS=true/X=1",
		"TestAccept/cmd/version/DMS=",
		"TestAccept/bundle/nothing",
		"TestAccept/cmd/version/notavariant",
		// A misspelled key or value matches no variant.
		"TestAccept/cmd/version/DSM=true",
		"TestAccept/cmd/version/DMS=ture",
		// A dir without a matrix has no variants to name.
		"TestAccept/bundle/resources/jobsx/DMS=true",
		"TestAcceptance/cmd/version",
		"TestFoo",
	}, testDirs)

	assert.Equal(t, Named{
		"bundle/resources/jobs":            {nil},
		"bundle/resources/jobs/update":     {nil},
		"bundle/resources/pipelines/basic": {nil},
		"cmd/version":                      {{"DMS=true", "X=1"}, {"DMS="}},
	}, named)
	assert.Equal(t, []string{
		"TestAccept/bundle/nothing",
		"TestAccept/cmd/version/notavariant",
		"TestAccept/cmd/version/DSM=true",
		"TestAccept/cmd/version/DMS=ture",
		"TestAccept/bundle/resources/jobsx/DMS=true",
		"TestAcceptance/cmd/version",
		"TestFoo",
	}, unmatched)

	all, unmatched := ResolveNames(root, []string{"TestAccept"}, testDirs)
	assert.Len(t, all, len(testDirs))
	assert.Empty(t, unmatched)
}

func TestNamedMatches(t *testing.T) {
	named := Named{
		"all":     {nil},
		"variant": {{"DMS=true", "X=1"}, {"DMS="}},
	}
	assert.True(t, named.Has("all"))
	assert.False(t, named.Has("other"))

	assert.True(t, named.Matches("all", []string{"DMS=true"}))
	assert.True(t, named.Matches("variant", []string{"DMS=true", "X=1"}))
	assert.True(t, named.Matches("variant", []string{"DMS=", "X=2"}))
	// Each name's filters hold together: DMS=true alone was not named with X=2.
	assert.False(t, named.Matches("variant", []string{"DMS=true", "X=2"}))
	assert.False(t, named.Matches("other", nil))

	var empty Named
	assert.False(t, empty.Has("all"))
	assert.False(t, empty.Matches("all", nil))
}

func TestLastCommitMessage(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}
	t.Chdir(t.TempDir())

	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", args...).CombinedOutput()
		require.NoError(t, err, "git %v\n%s", args, out)
		return string(out)
	}

	git("init", "-q", "-b", "main")
	git("config", "user.email", "test@example.invalid")
	git("config", "user.name", "Test")
	require.NoError(t, os.WriteFile("a.txt", []byte("a"), 0o600))
	git("add", "a.txt")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")

	git("checkout", "-q", "-b", "feature")
	git("commit", "-q", "--allow-empty", "-m", "run tests\n\naws: TestAccept/x")
	msg, err := LastCommitMessage()
	require.NoError(t, err)
	assert.Equal(t, "run tests\n\naws: TestAccept/x", strings.TrimSpace(msg))
	head := git("rev-parse", "HEAD")

	// The merge commit GitHub creates for refs/pull/N/merge: the message of the PR's head
	// commit is read instead of "Merge <head> into <base>".
	git("checkout", "-q", "main")
	git("commit", "-q", "--allow-empty", "-m", "main advances")
	git("merge", "-q", "--no-ff", "feature", "-m", "Merge "+head[:40]+" into "+base[:40])
	msg, err = LastCommitMessage()
	require.NoError(t, err)
	assert.Equal(t, "run tests\n\naws: TestAccept/x", strings.TrimSpace(msg))

	// Any other merge commit is the last commit itself.
	git("checkout", "-q", "feature")
	git("merge", "-q", "--no-ff", "main", "-m", "Merge branch 'main' into feature")
	msg, err = LastCommitMessage()
	require.NoError(t, err)
	assert.Equal(t, "Merge branch 'main' into feature", strings.TrimSpace(msg))
}
