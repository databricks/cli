package aircmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelativeScriptWarning(t *testing.T) {
	cases := []struct {
		name    string
		command string
		word    string // the command word reported, "" when no warning is expected
		suggest string // the path suggested under $CODE_SOURCE_PATH
	}{
		{"bare script", "python train.py", "train.py", "train.py"},
		{"dot-slash script", "./train.py --epochs 1", "./train.py", "train.py"},
		{"quoted nested script", `torchrun --nproc_per_node=8 "src/main.py"`, "src/main.py", "src/main.py"},
		{"code source path variable", `python "$CODE_SOURCE_PATH/train.py"`, "", ""},
		{"flag value", "python -m mod --config=train.py", "", ""},
		{"missing file", "python other.py", "", ""},
		{"directory", "ls src", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "train.py"), []byte("print(1)\n"), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.py"), []byte("print(1)\n"), 0o644))
			yaml := "experiment_name: e\ncompute:\n  num_accelerators: 1\n  accelerator_type: GPU_1xA10\ncode_source:\n  root_path: .\ncommand: " + tc.command + "\n"
			path := filepath.Join(dir, "run.yaml")
			require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))
			cfg, err := loadRunConfig(path)
			require.NoError(t, err)

			got := relativeScriptWarning(t.Context(), cfg, path)
			if tc.word == "" {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, "Warning: command refers to \""+tc.word+"\"")
			assert.Contains(t, got, "Use \"$CODE_SOURCE_PATH/"+tc.suggest+"\".")
		})
	}
}

func TestRelativeScriptWarningWithoutCodeSource(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "train.py"), []byte("print(1)\n"), 0o644))
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte("experiment_name: e\ncompute:\n  num_accelerators: 1\n  accelerator_type: GPU_1xA10\ncommand: python train.py\n"), 0o644))
	cfg, err := loadRunConfig(path)
	require.NoError(t, err)
	assert.Empty(t, relativeScriptWarning(t.Context(), cfg, path))
}
