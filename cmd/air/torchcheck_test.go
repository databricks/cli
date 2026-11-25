package aircmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissingTorchWarning(t *testing.T) {
	const torchScript = "import os\nimport torch.nn as nn\n"
	cases := []struct {
		name        string
		environment string
		script      string
		want        bool
	}{
		{"default environment", "", torchScript, true},
		{"standard 6", "environment:\n  version: 6\n", torchScript, true},
		{"from torch import", "", "from torch import nn\n", true},
		{"other deps only", "environment:\n  version: 6\n  dependencies:\n    - numpy\n", torchScript, true},
		{"databricks ai environment", "environment:\n  version: databricks_ai_v6\n", torchScript, false},
		{"torch dependency", "environment:\n  dependencies:\n    - torch==2.6.0\n", torchScript, false},
		{"requirements file", "environment:\n  dependencies:\n    - -r requirements.txt\n", torchScript, false},
		{"long requirements option", "environment:\n  dependencies:\n    - --requirement=requirements.txt\n", torchScript, false},
		{"editable package", "environment:\n  dependencies:\n    - -e ./mypkg\n", torchScript, false},
		{"index option installs nothing", "environment:\n  dependencies:\n    - --extra-index-url https://download.pytorch.org/whl/cu128\n    - numpy\n", torchScript, true},
		{"standard 4 includes torch", "environment:\n  version: 4\n", torchScript, false},
		{"custom image", "environment:\n  unity_catalog_image: main.s.img:v1\n", torchScript, false},
		{"no torch import", "", "import torchvision_stub\n# import torch\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "train.py"), []byte(tc.script), 0o644))
			yaml := "experiment_name: e\ncompute:\n  num_accelerators: 1\n  accelerator_type: GPU_1xA10\ncode_source:\n  root_path: .\ncommand: python train.py\n" + tc.environment
			path := filepath.Join(dir, "run.yaml")
			require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))
			cfg, err := loadRunConfig(path)
			require.NoError(t, err)

			got := missingTorchWarning(t.Context(), cfg, path)
			if tc.want {
				assert.Contains(t, got, "train.py contains a torch import, but Standard environment 6 does not include PyTorch")
				assert.Contains(t, got, "may fail")
				assert.Contains(t, got, "Consider using databricks_ai_v6")
			} else {
				assert.Empty(t, got)
			}
		})
	}
}

func writeTorchRunConfig(t *testing.T, dir, codeSource string) *runConfig {
	t.Helper()
	yaml := "experiment_name: e\ncompute:\n  num_accelerators: 1\n  accelerator_type: GPU_1xA10\ncode_source:\n  root_path: .\n" + codeSource + "command: python train.py\n"
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))
	cfg, err := loadRunConfig(path)
	require.NoError(t, err)
	return cfg
}

func TestMissingTorchWarningHonorsRuntimeImageOverride(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "train.py"), []byte("import torch\n"), 0o644))
	cfg := writeTorchRunConfig(t, dir, "")
	path := filepath.Join(dir, "run.yaml")

	// Submission uses DATABRICKS_DL_RUNTIME_IMAGE when environment.version is unset.
	assert.Empty(t, missingTorchWarning(env.Set(t.Context(), dlRuntimeImageEnv, "databricks_ai_v6"), cfg, path))
	assert.Contains(t, missingTorchWarning(env.Set(t.Context(), dlRuntimeImageEnv, "CLIENT-GPU-6"), cfg, path), "Standard environment 6 does not include PyTorch")
}

func TestMissingTorchWarningScansPinnedCommit(t *testing.T) {
	t.Run("import only in the commit", func(t *testing.T) {
		repo := newTestRepo(t)
		writeRepoFile(t, repo, "src/train.py", "import torch\n")
		sha := commitAll(t, repo, "torch")
		// The working tree no longer imports torch, but the pinned commit does.
		writeRepoFile(t, repo, "src/train.py", "print('hi')\n")
		cfg := writeTorchRunConfig(t, repo, "  git:\n    commit: "+sha+"\n")

		assert.Contains(t, missingTorchWarning(t.Context(), cfg, filepath.Join(repo, "run.yaml")), "src/train.py contains a torch import")
	})

	t.Run("import only in the working tree", func(t *testing.T) {
		repo := newTestRepo(t)
		writeRepoFile(t, repo, "src/train.py", "print('hi')\n")
		sha := commitAll(t, repo, "no torch")
		writeRepoFile(t, repo, "src/train.py", "import torch\n")
		cfg := writeTorchRunConfig(t, repo, "  git:\n    commit: "+sha+"\n")

		assert.Empty(t, missingTorchWarning(t.Context(), cfg, filepath.Join(repo, "run.yaml")))
	})

	t.Run("subdirectory root reports a root-relative path", func(t *testing.T) {
		repo := newTestRepo(t)
		writeRepoFile(t, repo, "proj/model/train.py", "from torch import nn\n")
		writeRepoFile(t, repo, "other/x.py", "import torch\n")
		sha := commitAll(t, repo, "torch")
		dir := filepath.Join(repo, "proj")
		cfg := writeTorchRunConfig(t, dir, "  git:\n    commit: "+sha+"\n  include_paths:\n    - model\n")

		assert.Contains(t, missingTorchWarning(t.Context(), cfg, filepath.Join(dir, "run.yaml")), "Warning: model/train.py contains a torch import")
	})

	t.Run("single file include", func(t *testing.T) {
		repo := newTestRepo(t)
		writeRepoFile(t, repo, "train.py", "import torch\n")
		sha := commitAll(t, repo, "torch")
		cfg := writeTorchRunConfig(t, repo, "  git:\n    commit: "+sha+"\n  include_paths:\n    - train.py\n")

		assert.Contains(t, missingTorchWarning(t.Context(), cfg, filepath.Join(repo, "run.yaml")), "Warning: train.py contains a torch import")
	})

	t.Run("single file include excludes other imports", func(t *testing.T) {
		repo := newTestRepo(t)
		writeRepoFile(t, repo, "train.py", "print('hi')\n")
		writeRepoFile(t, repo, "other.py", "import torch\n")
		sha := commitAll(t, repo, "torch outside include")
		cfg := writeTorchRunConfig(t, repo, "  git:\n    commit: "+sha+"\n  include_paths:\n    - train.py\n")

		assert.Empty(t, missingTorchWarning(t.Context(), cfg, filepath.Join(repo, "run.yaml")))
	})
}
