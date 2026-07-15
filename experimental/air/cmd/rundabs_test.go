package aircmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	bundleresources "github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteBundleProject(t *testing.T) {
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("echo hi"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
	}

	root, cleanup, err := writeBundleProject(t.Context(), cfg, "train.yaml")
	require.NoError(t, err)
	defer cleanup()

	// databricks.yml carries the converted job and the appended dev target.
	body, err := os.ReadFile(filepath.Join(root, "databricks.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "ai_runtime_task:")
	assert.Contains(t, string(body), "immutable_folder: true")
	assert.Contains(t, string(body), "mode: development")

	// command.sh holds the run's command verbatim.
	script, err := os.ReadFile(filepath.Join(root, bundleCommandScript))
	require.NoError(t, err)
	assert.Equal(t, "echo hi", string(script))

	// cleanup removes the temp root.
	cleanup()
	_, err = os.Stat(root)
	assert.True(t, os.IsNotExist(err))
}

func TestWriteBundleProjectStagesCodeSource(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "train.py"), []byte("print()"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "pkg", "mod.py"), []byte("x=1"), 0o644))

	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
		CodeSource:     &codeSourceConfig{Type: "snapshot", Snapshot: &snapshotSourceConfig{RootPath: src}},
	}

	root, cleanup, err := writeBundleProject(t.Context(), cfg, "train.yaml")
	require.NoError(t, err)
	defer cleanup()

	// The user's tree is copied into the bundle root alongside our generated files.
	assert.FileExists(t, filepath.Join(root, "train.py"))
	assert.FileExists(t, filepath.Join(root, "pkg", "mod.py"))
	assert.FileExists(t, filepath.Join(root, "databricks.yml"))
	assert.FileExists(t, filepath.Join(root, bundleCommandScript))
}

func TestWriteBundleProjectCommandShadowProtection(t *testing.T) {
	// A command.sh in the user's tree must not shadow the one we generate from the
	// run's command: our write happens after the tree copy.
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, bundleCommandScript), []byte("STALE"), 0o644))

	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("FRESH"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
		CodeSource:     &codeSourceConfig{Type: "snapshot", Snapshot: &snapshotSourceConfig{RootPath: src}},
	}

	root, cleanup, err := writeBundleProject(t.Context(), cfg, "train.yaml")
	require.NoError(t, err)
	defer cleanup()

	script, err := os.ReadFile(filepath.Join(root, bundleCommandScript))
	require.NoError(t, err)
	assert.Equal(t, "FRESH", string(script))
}

func TestWriteBundleProjectRejectsUnconvertible(t *testing.T) {
	// A gate failure (here: a $CODE_SOURCE_PATH command) surfaces before any temp
	// directory is created.
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("cd $CODE_SOURCE_PATH && python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
	}
	_, _, err := writeBundleProject(t.Context(), cfg, "train.yaml")
	require.Error(t, err)
}

func TestBundleTargetsBlock(t *testing.T) {
	block := bundleTargetsBlock()
	assert.Contains(t, block, "targets:")
	assert.Contains(t, block, dabsTarget+":")
	assert.Contains(t, block, "mode: development")
	assert.Contains(t, block, "default: true")
}

func TestNewBundleCarrierCommand(t *testing.T) {
	// Construct the client directly (not via NewWorkspaceClient, which would try to
	// resolve "myprofile" from ~/.databrickscfg); the carrier only reads Config.Profile.
	w := &databricks.WorkspaceClient{Config: &config.Config{Profile: "myprofile"}}

	cmd := newBundleCarrierCommand(t.Context(), w, "/tmp/bundle-root")

	// air's active profile is forwarded so the bundle authenticates the same way.
	assert.Equal(t, "myprofile", cmd.Flag("profile").Value.String())

	// The flags ProcessBundle reads are declared.
	for _, name := range []string{"var", "target", "profile", "output"} {
		assert.NotNil(t, cmd.Flag(name), "flag %q must be declared", name)
	}

	// The bundle root and direct engine are seeded on the command's context.
	assert.Equal(t, "/tmp/bundle-root", env.Get(cmd.Context(), "DATABRICKS_BUNDLE_ROOT"))
	assert.Equal(t, "direct", env.Get(cmd.Context(), "DATABRICKS_BUNDLE_ENGINE"))
}

func TestNewBundleCarrierCommandNoProfile(t *testing.T) {
	// With no profile on the client, the profile flag is left empty (the bundle
	// falls back to its own default resolution).
	w := &databricks.WorkspaceClient{Config: &config.Config{}}

	cmd := newBundleCarrierCommand(t.Context(), w, "/tmp/bundle-root")
	assert.Empty(t, cmd.Flag("profile").Value.String())
}

func TestIsRunnableJob(t *testing.T) {
	assert.True(t, isRunnableJob(bundleresources.Reference{Resource: &resources.Job{}}))
	assert.False(t, isRunnableJob(bundleresources.Reference{Resource: &resources.Pipeline{}}))
}
