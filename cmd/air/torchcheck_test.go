package aircmd

import (
	"os"
	"path/filepath"
	"testing"

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
				assert.Contains(t, got, "train.py imports torch")
				assert.Contains(t, got, "environment.version: databricks_ai_v6")
			} else {
				assert.Empty(t, got)
			}
		})
	}
}
