package localenv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rootPM reports a fixed workspace root (or error) from WorkspaceRoot.
type rootPM struct {
	fakePM
	root string
	err  error
}

func (f rootPM) WorkspaceRoot(context.Context, string) (string, error) { return f.root, f.err }

func memberLayout(t *testing.T) (root, member string) {
	root = t.TempDir()
	member = filepath.Join(root, "bundles", "a")
	require.NoError(t, os.MkdirAll(member, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, pyprojectFile), []byte("[tool.uv.workspace]\nmembers = [\"bundles/*\"]\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(member, pyprojectFile), []byte("[project]\nname = \"a\"\n"), 0o644))
	return root, member
}

func TestDiscoverWorkspaceRootMember(t *testing.T) {
	root, member := memberLayout(t)
	p := &Pipeline{ProjectDir: member, PM: rootPM{root: root}}
	assert.Equal(t, root, p.discoverWorkspaceRoot(t.Context()))
}

func TestDiscoverWorkspaceRootStandalone(t *testing.T) {
	_, member := memberLayout(t)
	p := &Pipeline{ProjectDir: member, PM: rootPM{root: member}}
	assert.Empty(t, p.discoverWorkspaceRoot(t.Context()))
}

func TestDiscoverWorkspaceRootSymlinkedSameDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	_, member := memberLayout(t)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(member, link))
	// uv prints the resolved path; ProjectDir is the symlinked one.
	p := &Pipeline{ProjectDir: link, PM: rootPM{root: member}}
	assert.Empty(t, p.discoverWorkspaceRoot(t.Context()))
}

func TestDiscoverWorkspaceRootSkipsGreenfield(t *testing.T) {
	root := t.TempDir()
	member := filepath.Join(root, "new")
	require.NoError(t, os.MkdirAll(member, 0o755))
	p := &Pipeline{ProjectDir: member, PM: rootPM{root: root}}
	assert.Empty(t, p.discoverWorkspaceRoot(t.Context()))
}

func TestDiscoverWorkspaceRootUvFailure(t *testing.T) {
	_, member := memberLayout(t)
	p := &Pipeline{ProjectDir: member, PM: rootPM{err: errors.New("unrecognized subcommand 'workspace'")}}
	assert.Empty(t, p.discoverWorkspaceRoot(t.Context()))
}

func TestWorkspaceDirArgs(t *testing.T) {
	assert.Equal(t, []string{"workspace", "dir"}, (&uvManager{}).workspaceDirArgs())
}

func newMemberPipeline(t *testing.T) (p *Pipeline, root, member string) {
	root, member = memberLayout(t)
	// The member needs requires-python so the existing merge accepts it.
	require.NoError(t, os.WriteFile(filepath.Join(member, pyprojectFile),
		[]byte("[project]\nname = \"a\"\nversion = \"0.1.0\"\nrequires-python = \">=3.10\"\n"), 0o644))
	p = newServerlessTestPipeline(t, member)
	p.PM = fakePM{py: "3.12", dbc: "17.2.0", root: root}
	return p, root, member
}

func warningCodes(res *Result) []string {
	var codes []string
	for _, w := range res.Warnings {
		codes = append(codes, w.Code)
	}
	return codes
}

func TestPipelineWorkspaceMember(t *testing.T) {
	p, root, member := newMemberPipeline(t)
	res, err := p.Run(t.Context())
	require.NoError(t, err)

	rootBytes, err := os.ReadFile(filepath.Join(root, pyprojectFile))
	require.NoError(t, err)
	memberBytes, err := os.ReadFile(filepath.Join(member, pyprojectFile))
	require.NoError(t, err)
	assert.Contains(t, string(rootBytes), "constraint-dependencies")
	assert.Contains(t, string(rootBytes), "[tool.uv.workspace]")
	assert.NotContains(t, string(memberBytes), "constraint-dependencies")
	assert.Contains(t, string(memberBytes), "requires-python")
	assert.Contains(t, string(memberBytes), `"databricks-connect~=17.2.0"`)

	assert.Equal(t, "../../.venv", res.VenvPath)
	assert.Equal(t, filepath.ToSlash(root), res.WorkspaceRoot)
	assert.Equal(t, filepath.ToSlash(filepath.Join(root, backupFile)), res.WorkspaceBackupPath)
	assert.Equal(t, filepath.ToSlash(filepath.Join(member, backupFile)), res.BackupPath)
	assert.NotContains(t, warningCodes(res), WarnWorkspaceConstraintsReplaced, "first run must not warn")
}

func TestPipelineWorkspaceMemberIdempotent(t *testing.T) {
	p, root, member := newMemberPipeline(t)
	_, err := p.Run(t.Context())
	require.NoError(t, err)
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.Empty(t, res.BackupPath)
	assert.Empty(t, res.WorkspaceBackupPath)
	assert.NotContains(t, warningCodes(res), WarnWorkspaceConstraintsReplaced)
	for _, dir := range []string{root, member} {
		matches, _ := filepath.Glob(filepath.Join(dir, "pyproject.toml.*.bak"))
		assert.Empty(t, matches, "no timestamped backup in %s", dir)
	}
}

func TestPipelineWorkspaceConstraintsReplacedWarning(t *testing.T) {
	p, root, _ := newMemberPipeline(t)
	_, err := p.Run(t.Context())
	require.NoError(t, err)
	// Simulate another bundle's run: change the pins inside the managed block.
	path := filepath.Join(root, pyprojectFile)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	changed := strings.Replace(string(b), "constraint-dependencies = [", "constraint-dependencies = [\n    \"other==1.0\",", 1)
	require.NoError(t, os.WriteFile(path, []byte(changed), 0o644))

	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.Contains(t, warningCodes(res), WarnWorkspaceConstraintsReplaced)
}

func TestPipelineWorkspaceMemberNoConstraints(t *testing.T) {
	p, root, _ := newMemberPipeline(t)
	p.SkipConstraints = true
	before, err := os.ReadFile(filepath.Join(root, pyprojectFile))
	require.NoError(t, err)
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	after, err := os.ReadFile(filepath.Join(root, pyprojectFile))
	require.NoError(t, err)
	assert.Equal(t, before, after)
	assert.Empty(t, res.WorkspaceBackupPath)
	assert.NoFileExists(t, filepath.Join(root, backupFile))
}

func TestPipelineWorkspaceMemberDryRun(t *testing.T) {
	p, root, member := newMemberPipeline(t)
	p.Check = true
	before, err := os.ReadFile(filepath.Join(root, pyprojectFile))
	require.NoError(t, err)
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	after, err := os.ReadFile(filepath.Join(root, pyprojectFile))
	require.NoError(t, err)
	assert.Equal(t, before, after)
	require.NotNil(t, res.Plan)
	assert.Equal(t, filepath.ToSlash(filepath.Join(member, pyprojectFile)), res.Plan.WouldWrite)
	assert.Equal(t, filepath.ToSlash(filepath.Join(root, pyprojectFile)), res.Plan.WouldWriteWorkspace)
	assert.Equal(t, filepath.ToSlash(filepath.Join(root, backupFile)), res.Plan.WouldBackupWorkspace)
	assert.Contains(t, res.Plan.Diff, "workspace root")
}

func TestPipelineWorkspaceRootNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("directory permissions do not block writes on Windows or as root")
	}
	p, root, member := newMemberPipeline(t)
	require.NoError(t, os.Chmod(root, 0o555))
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	memberBefore, err := os.ReadFile(filepath.Join(member, pyprojectFile))
	require.NoError(t, err)

	res, err := p.Run(t.Context())
	require.Error(t, err)
	require.NotNil(t, res.Error)
	assert.Equal(t, ErrNotWritable, res.Error.Code)
	assert.Contains(t, res.Error.Error(), filepath.ToSlash(root))
	memberAfter, err := os.ReadFile(filepath.Join(member, pyprojectFile))
	require.NoError(t, err)
	assert.Equal(t, memberBefore, memberAfter)
}

func TestDiscoverWorkspaceRootKeepsProjectPathForm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	root, _ := memberLayout(t)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(root, link))
	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	// uv prints the symlink-resolved root; ProjectDir reaches the member through
	// the link. The root must come back in ProjectDir's form so the venv path
	// relative to the project stays "../../.venv".
	p := &Pipeline{ProjectDir: filepath.Join(link, "bundles", "a"), PM: rootPM{root: resolved}}
	assert.Equal(t, link, p.discoverWorkspaceRoot(t.Context()))
}
