package agents

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bindMountTestChildEnv = "DATABRICKS_CLI_TEST_BIND_MOUNT_CHILD"

// A bind mount names the same directory under a second path that no symlink
// resolution can map back, the way a case-insensitive filesystem does on macOS
// and Windows, so it exercises the same identity check on Linux. Mounting needs
// a private mount namespace, so the test re-runs itself under `unshare -rm`
// (an unprivileged user namespace) and skips where those are not allowed.
func TestPluginInstallAppliesToBindMountedProjectPath(t *testing.T) {
	if os.Getenv(bindMountTestChildEnv) == "" {
		unshare, err := exec.LookPath("unshare")
		if err != nil {
			t.Skip("unshare not found")
		}
		if err := exec.Command(unshare, "-rm", "true").Run(); err != nil {
			t.Skipf("unprivileged mount namespaces are not available: %v", err)
		}
		cmd := exec.Command(unshare, "-rm", os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), bindMountTestChildEnv+"=1")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", out)
		if strings.Contains(string(out), "--- SKIP: "+t.Name()) {
			t.Skipf("%s", out)
		}
		assert.Contains(t, string(out), "--- PASS: "+t.Name())
		return
	}

	project, sub := projectFixture(t)
	alias := filepath.Join(filepath.Dir(project), "alias")
	require.NoError(t, os.Mkdir(alias, 0o755))
	// Some hosts (e.g. AppArmor-restricted user namespaces) allow the
	// namespace but deny the mount.
	if err := syscall.Mount(project, alias, "", syscall.MS_BIND, ""); err != nil {
		t.Skipf("bind mount not permitted: %v", err)
	}
	t.Cleanup(func() { assert.NoError(t, syscall.Unmount(alias, 0)) })

	for _, cwd := range []string{project, sub} {
		assert.True(t, pluginInstall{Scope: "project", ProjectPath: alias}.appliesTo(cwd), cwd)
	}
	// The mount does not make an unrelated directory count.
	assert.False(t, pluginInstall{Scope: "project", ProjectPath: alias}.appliesTo(filepath.Dir(project)))
}
