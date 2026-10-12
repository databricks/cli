package localenv

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memberLayout creates a uv workspace root with one member, bundles/a.
func memberLayout(t *testing.T) (root, member string) {
	t.Helper()
	root = t.TempDir()
	member = filepath.Join(root, "bundles", "a")
	require.NoError(t, os.MkdirAll(member, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, pyprojectFile), []byte("[tool.uv.workspace]\nmembers = [\"bundles/*\"]\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(member, pyprojectFile), []byte("[project]\nname = \"a\"\n"), 0o644))
	return root, member
}

func TestDiscoverWorkspaceRoot(t *testing.T) {
	root, member := memberLayout(t)
	greenfield := filepath.Join(root, "new")
	require.NoError(t, os.MkdirAll(greenfield, 0o755))

	tests := []struct {
		name       string
		projectDir string
		pm         fakePM
		wantRoot   string
		wantRel    string
	}{
		{"member", member, fakePM{root: root}, root, filepath.Join("..", "..")},
		{"standalone", member, fakePM{root: member}, "", ""},
		// The created pyproject.toml is not a member yet; see discoverWorkspaceRoot.
		{"greenfield", greenfield, fakePM{root: root}, "", ""},
		{"uv failure", member, fakePM{rootErr: errors.New("unrecognized subcommand 'workspace'")}, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Pipeline{ProjectDir: tc.projectDir, PM: tc.pm}
			gotRoot, gotRel := p.discoverWorkspaceRoot(t.Context())
			assert.Equal(t, tc.wantRoot, gotRoot)
			assert.Equal(t, tc.wantRel, gotRel)
		})
	}
}

func TestDiscoverWorkspaceRootSymlinkedSameDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	_, member := memberLayout(t)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(member, link))
	// uv prints the resolved path; ProjectDir is the symlinked one.
	p := &Pipeline{ProjectDir: link, PM: fakePM{root: member}}
	got, _ := p.discoverWorkspaceRoot(t.Context())
	assert.Empty(t, got)
}

// caseFoldedPath returns path with its last element in upper case, and skips
// the test on a case-sensitive file system, where that path does not exist.
func caseFoldedPath(t *testing.T, path string) string {
	t.Helper()
	folded := filepath.Join(filepath.Dir(path), strings.ToUpper(filepath.Base(path)))
	if _, err := os.Stat(folded); err != nil {
		t.Skip("the file system is case-sensitive")
	}
	return folded
}

func TestDiscoverWorkspaceRootCaseMismatchStandalone(t *testing.T) {
	// The shell keeps the letter case the user typed; uv prints the case on disk.
	_, member := memberLayout(t)
	p := &Pipeline{ProjectDir: caseFoldedPath(t, member), PM: fakePM{root: member}}
	got, _ := p.discoverWorkspaceRoot(t.Context())
	assert.Empty(t, got)
}

func TestDiscoverWorkspaceRootCaseMismatchMember(t *testing.T) {
	root, _ := memberLayout(t)
	foldedRoot := caseFoldedPath(t, root)
	p := &Pipeline{ProjectDir: filepath.Join(foldedRoot, "bundles", "a"), PM: fakePM{root: root}}
	gotRoot, gotRel := p.discoverWorkspaceRoot(t.Context())
	assert.Equal(t, foldedRoot, gotRoot)
	assert.Equal(t, filepath.Join("..", ".."), gotRel)
}

// newMemberPipeline returns a pipeline for the member of a memberLayout.
func newMemberPipeline(t *testing.T) (p *Pipeline, root, member string) {
	t.Helper()
	root, member = memberLayout(t)
	// The member needs requires-python so the existing merge accepts it.
	require.NoError(t, os.WriteFile(filepath.Join(member, pyprojectFile),
		[]byte("[project]\nname = \"a\"\nversion = \"0.1.0\"\nrequires-python = \">=3.10\"\n"), 0o644))
	p = newServerlessTestPipeline(t, member)
	p.PM = fakePM{py: "3.12", dbc: "17.2.0", root: root}
	return p, root, member
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
	assert.NotContains(t, codes(res.Warnings), WarnWorkspaceConstraintsReplaced, "first run must not warn")
}

func TestPipelineWorkspaceMemberIdempotent(t *testing.T) {
	p, root, member := newMemberPipeline(t)
	_, err := p.Run(t.Context())
	require.NoError(t, err)
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.Empty(t, res.BackupPath)
	assert.Empty(t, res.WorkspaceBackupPath)
	assert.NotContains(t, codes(res.Warnings), WarnWorkspaceConstraintsReplaced)
	for _, dir := range []string{root, member} {
		matches, err := filepath.Glob(filepath.Join(dir, "pyproject.toml.*.bak"))
		require.NoError(t, err)
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
	assert.Contains(t, codes(res.Warnings), WarnWorkspaceConstraintsReplaced)
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
	p := &Pipeline{ProjectDir: filepath.Join(link, "bundles", "a"), PM: fakePM{root: resolved}}
	got, _ := p.discoverWorkspaceRoot(t.Context())
	assert.Equal(t, link, got)
}

// symlinkBelowRootLayout reaches the member through a link to the workspace's
// bundles folder (link -> <root>/bundles), so the link sits below the root.
func symlinkBelowRootLayout(t *testing.T) (resolvedRoot, projectDir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	root, _ := memberLayout(t)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(filepath.Join(root, "bundles"), link))
	resolvedRoot, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	return resolvedRoot, filepath.Join(link, "a")
}

func TestDiscoverWorkspaceRootSymlinkBelowRoot(t *testing.T) {
	resolvedRoot, projectDir := symlinkBelowRootLayout(t)
	// Joining "../.." onto the link path would give the link's parent, not the
	// root; the root must still name the real workspace root.
	p := &Pipeline{ProjectDir: projectDir, PM: fakePM{root: resolvedRoot}}
	gotRoot, gotRel := p.discoverWorkspaceRoot(t.Context())
	assert.Equal(t, resolvedRoot, gotRoot)
	// Relative to the member's real folder (<root>/bundles/a).
	assert.Equal(t, filepath.Join("..", ".."), gotRel)
}

func TestPipelineWorkspaceMemberVenvPathWithResolvedUvRoot(t *testing.T) {
	// Real uv prints the symlink-resolved root (macOS /private/var).
	p, root, _ := newMemberPipeline(t)
	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	p.PM = fakePM{py: "3.12", dbc: "17.2.0", root: resolved}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "../../.venv", res.VenvPath)
}

func TestPipelineWorkspaceMemberSymlinkBelowRoot(t *testing.T) {
	resolvedRoot, projectDir := symlinkBelowRootLayout(t)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, pyprojectFile),
		[]byte("[project]\nname = \"a\"\nversion = \"0.1.0\"\nrequires-python = \">=3.10\"\n"), 0o644))
	p := newServerlessTestPipeline(t, projectDir)
	p.PM = fakePM{py: "3.12", dbc: "17.2.0", root: resolvedRoot}
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	// Relative to the member's real folder (<root>/bundles/a), which is what the
	// OS resolves a relative path against.
	assert.Equal(t, "../../.venv", res.VenvPath)
	rootBytes, err := os.ReadFile(filepath.Join(resolvedRoot, pyprojectFile))
	require.NoError(t, err)
	assert.Contains(t, string(rootBytes), "constraint-dependencies")
}

func TestPipelineWorkspaceReplacesUserRootPinsWithWarning(t *testing.T) {
	p, root, _ := newMemberPipeline(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, pyprojectFile),
		[]byte("[tool.uv.workspace]\nmembers = [\"bundles/*\"]\n\n[tool.uv]\nconstraint-dependencies = [\"urllib3<2\"]\n"), 0o644))

	res, err := p.Run(t.Context())
	require.NoError(t, err)
	assert.Contains(t, codes(res.Warnings), WarnWorkspaceConstraintsReplaced)
}

func TestApplyMergeRootFailureBeforeAnyWriteIsNotDiskMutated(t *testing.T) {
	// The member merge is a no-op and the root file cannot be read, so nothing on
	// disk changed and the failure must say so.
	_, member := memberLayout(t)
	memberPath := filepath.Join(member, pyprojectFile)
	current, err := os.ReadFile(memberPath)
	require.NoError(t, err)
	p := &Pipeline{ProjectDir: member, res: NewResult()}
	p.res.Phases = initialPhases()
	missing := filepath.Join(t.TempDir(), pyprojectFile)

	err = p.applyMerge(t.Context(), current, &rootMerge{path: missing, merged: []byte("x")}, false)
	require.Error(t, err)
	require.NotNil(t, p.res.Error)
	assert.False(t, p.res.Error.DiskMutated)
	assert.Contains(t, p.res.Error.Error(), "workspace root")
}

func TestDiscoverWorkspaceRootSymlinkChangesDepth(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	// <root>/link -> <root>/deep/bundles: the logical path reaches the root in
	// two levels, but the OS resolves a relative path from the real member folder
	// (<root>/deep/bundles/a), three levels below the root.
	root := t.TempDir()
	real := filepath.Join(root, "deep", "bundles", "a")
	require.NoError(t, os.MkdirAll(real, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(real, pyprojectFile), []byte("[project]\nname = \"a\"\n"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(root, "deep", "bundles"), filepath.Join(root, "link")))

	p := &Pipeline{ProjectDir: filepath.Join(root, "link", "a"), PM: fakePM{root: root}}
	gotRoot, gotRel := p.discoverWorkspaceRoot(t.Context())
	assert.Equal(t, root, gotRoot)
	assert.Equal(t, filepath.Join("..", "..", ".."), gotRel)
}

func TestPipelineWorkspaceMemberDryRunReRunPlansNoRootWrite(t *testing.T) {
	p, _, _ := newMemberPipeline(t)
	_, err := p.Run(t.Context())
	require.NoError(t, err)

	p.Check = true
	res, err := p.Run(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res.Plan)
	assert.Empty(t, res.Plan.WouldWriteWorkspace)
	assert.Empty(t, res.Plan.WouldBackupWorkspace)
	assert.NotEmpty(t, res.WorkspaceRoot)
}
