package aircmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToBundleBasic(t *testing.T) {
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_8xH100", NumAccelerators: 16},
		MaxRetries:     new(2),
		TimeoutMinutes: new(30),
	}

	b := convertToBundle(cfg)

	assert.Equal(t, "exp", b.Bundle.Name)
	job, ok := b.Resources.Jobs["exp"]
	require.True(t, ok, "job keyed by experiment name")
	require.Len(t, job.Tasks, 1)
	task := job.Tasks[0]
	assert.Equal(t, "exp", task.TaskKey)
	assert.Equal(t, aiRuntimeEnvironmentKey, task.EnvironmentKey)
	assert.Equal(t, 2, task.MaxRetries)
	assert.Equal(t, 1800, task.TimeoutSeconds)

	require.Len(t, task.AiRuntimeTask.Deployments, 1)
	dep := task.AiRuntimeTask.Deployments[0]
	// command_path is relative to the bundle root; translate_paths rewrites it to
	// the deployed location.
	assert.Equal(t, bundleCommandScript, dep.CommandPath)
	assert.Equal(t, "GPU_8xH100", dep.Compute.AcceleratorType)
	assert.Equal(t, 16, dep.Compute.AcceleratorCount)

	// No env vars -> no profile and no task key, so the wire form matches a
	// var-free submit exactly.
	assert.Empty(t, job.EnvironmentVariables)
	assert.Empty(t, task.EnvironmentVariablesKey)
}

func TestConvertToBundlePermissions(t *testing.T) {
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
		Permissions: []permission{
			{Level: "CAN_VIEW", GroupName: new("users")},
			{Level: "CAN_MANAGE", UserName: new("a@b.com")},
		},
	}

	job := convertToBundle(cfg).Resources.Jobs["exp"]
	require.Len(t, job.Permissions, 2)
	assert.Equal(t, exportedPermission{Level: "CAN_VIEW", GroupName: "users"}, job.Permissions[0])
	assert.Equal(t, exportedPermission{Level: "CAN_MANAGE", UserName: "a@b.com"}, job.Permissions[1])

	// No permissions declared -> field omitted (no empty permissions block).
	assert.Empty(t, convertToBundle(&runConfig{
		ExperimentName: "exp", Command: new("x"),
		Compute: &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
	}).Resources.Jobs["exp"].Permissions)
}

func TestParsePermissions(t *testing.T) {
	got, err := parsePermissions([]string{
		"CAN_VIEW=group_name:users",
		"CAN_MANAGE=user_name:a@b.com",
		"CAN_RUN=service_principal_name:1234-abcd",
	})
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "CAN_VIEW", got[0].Level)
	assert.Equal(t, "users", *got[0].GroupName)
	assert.Equal(t, "a@b.com", *got[1].UserName)
	assert.Equal(t, "1234-abcd", *got[2].ServicePrincipalName)

	// Malformed inputs error rather than silently drop.
	for _, bad := range []string{"CAN_VIEW", "CAN_VIEW=users", "CAN_VIEW=bogus:x"} {
		_, err := parsePermissions([]string{bad})
		require.Error(t, err, "expected error for %q", bad)
	}
}

func TestConvertToBundleEnvVarsAndSecrets(t *testing.T) {
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
		EnvVariables:   map[string]string{"FOO": "bar", "LOG_LEVEL": "INFO"},
		Secrets:        map[string]string{"TOKEN": "myscope/mykey"},
	}

	b := convertToBundle(cfg)
	job := b.Resources.Jobs["exp"]

	// The task references the single profile by key.
	assert.Equal(t, aiRuntimeEnvVarsKey, job.Tasks[0].EnvironmentVariablesKey)

	// One profile, keyed to match, carrying plain values inline and the secret as a
	// {{secrets/scope/key}} reference (common Jobs env-var API; resolved by Jobs at
	// run time). Verified against staging: this shape persists on runs/get.
	require.Len(t, job.EnvironmentVariables, 1)
	prof := job.EnvironmentVariables[0]
	assert.Equal(t, aiRuntimeEnvVarsKey, prof.EnvironmentVariablesKey)
	assert.Equal(t, "bar", prof.Variables["FOO"])
	assert.Equal(t, "INFO", prof.Variables["LOG_LEVEL"])
	assert.Equal(t, "{{secrets/myscope/mykey}}", prof.Variables["TOKEN"])
}

func TestCheckBundleConvertibleAllowsEnvVars(t *testing.T) {
	// env_variables and secrets are now representable via the common Jobs env-var
	// API, so the gate must NOT reject them (regression guard for the lift).
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
		EnvVariables:   map[string]string{"FOO": "bar"},
		Secrets:        map[string]string{"TOKEN": "s/k"},
	}
	assert.NoError(t, checkBundleConvertible(cfg))
}

func TestCheckBundleConvertibleRejectsCodeSourcePathCommand(t *testing.T) {
	// A command that reads $CODE_SOURCE_PATH assumes the air run harness a bundle
	// doesn't provide; the gate must still reject it with an actionable message.
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("cd $CODE_SOURCE_PATH && python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
	}
	err := checkBundleConvertible(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), codeSourcePathVar)
}

func TestCheckBundleConvertibleCodeSource(t *testing.T) {
	base := func() *runConfig {
		return &runConfig{
			ExperimentName: "exp",
			Command:        new("python train.py"),
			Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
		}
	}

	// A plain working-tree snapshot is convertible: it uploads via immutable folder.
	ok := base()
	ok.CodeSource = &codeSourceConfig{Type: "snapshot", Snapshot: &snapshotSourceConfig{RootPath: "."}}
	assert.NoError(t, checkBundleConvertible(ok))

	// A git-pinned snapshot can't be represented (working-tree upload only).
	git := base()
	git.CodeSource = &codeSourceConfig{Type: "snapshot", Snapshot: &snapshotSourceConfig{RootPath: ".", Git: &gitRef{Commit: new("abc123")}}}
	err := checkBundleConvertible(git)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git")

	// A UC Volume destination isn't supported (Workspace Files only).
	vol := base()
	vol.CodeSource = &codeSourceConfig{Type: "snapshot", Snapshot: &snapshotSourceConfig{RootPath: ".", RemoteVolume: new("/Volumes/c/s/v")}}
	err = checkBundleConvertible(vol)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote_volume")
}

func TestStageCodeSource(t *testing.T) {
	// A working-tree snapshot copies the tree into the bundle root; include_paths
	// restricts to the named subpaths.
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "train.py"), []byte("print()"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "pkg", "mod.py"), []byte("x=1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "ignore.txt"), []byte("no"), 0o644))

	t.Run("whole tree", func(t *testing.T) {
		dst := t.TempDir()
		snap := &snapshotSourceConfig{RootPath: src}
		require.NoError(t, stageCodeSource(t.Context(), snap, "train.yaml", dst))
		assert.FileExists(t, filepath.Join(dst, "train.py"))
		assert.FileExists(t, filepath.Join(dst, "pkg", "mod.py"))
		assert.FileExists(t, filepath.Join(dst, "ignore.txt"))
	})

	t.Run("include_paths only", func(t *testing.T) {
		dst := t.TempDir()
		snap := &snapshotSourceConfig{RootPath: src, IncludePaths: []string{"train.py", "pkg"}}
		require.NoError(t, stageCodeSource(t.Context(), snap, "train.yaml", dst))
		assert.FileExists(t, filepath.Join(dst, "train.py"))
		assert.FileExists(t, filepath.Join(dst, "pkg", "mod.py"))
		assert.NoFileExists(t, filepath.Join(dst, "ignore.txt"))
	})
}

func TestRenderBundleIncludesTargetsAndConvertGate(t *testing.T) {
	// renderBundle (what --dry-run shows and what the run path deploys) must include
	// the converted job, the appended dev targets block, and the immutable_folder
	// flag that routes deploy through the content-addressed snapshot path.
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
		EnvVariables:   map[string]string{"FOO": "bar"},
	}
	out, err := renderBundle(cfg, "train.yaml")
	require.NoError(t, err)
	assert.Contains(t, out, "ai_runtime_task:")
	assert.Contains(t, out, "FOO: bar")
	assert.Contains(t, out, "targets:")
	assert.Contains(t, out, "mode: development")
	assert.Contains(t, out, "immutable_folder: true")

	// The convertibility gate still applies: an unconvertible config errors instead
	// of rendering a lossy bundle.
	bad := &runConfig{
		ExperimentName: "exp",
		Command:        new("cd $CODE_SOURCE_PATH && python train.py"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
	}
	_, err = renderBundle(bad, "train.yaml")
	require.Error(t, err)
}

func TestMarshalBundleEnvVarsRoundTrip(t *testing.T) {
	// The emitted YAML must carry the env-var profile so `bundle deploy` sends it.
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("x"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xA10", NumAccelerators: 1},
		EnvVariables:   map[string]string{"FOO": "bar"},
	}
	out, err := marshalBundle(convertToBundle(cfg), "train.yaml")
	require.NoError(t, err)
	body := string(out)
	assert.Contains(t, body, "environment_variables_key: default")
	assert.Contains(t, body, "environment_variables:")
	assert.Contains(t, body, "FOO: bar")
	// Header provenance line is present.
	assert.Contains(t, body, "Generated by `air export-bundle`")
}
