package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRelativeScriptWarning(t *testing.T) {
	snapshot := []string{"train.py", "src/main.py", "configs/base.yaml", "scripts/train.sh", "launch.sh"}
	cases := []struct {
		name    string
		command string
		word    string // the command word reported, "" when no warning is expected
		suggest string // the path suggested under $CODE_SOURCE_PATH
	}{
		{"bare script", "python train.py", "train.py", "train.py"},
		{"python3 with flags", "python3 -u train.py --epochs 1", "train.py", "train.py"},
		{"dot-slash script", "python ./train.py", "./train.py", "train.py"},
		{"quoted nested script", `torchrun --nproc_per_node=8 "src/main.py"`, "src/main.py", "src/main.py"},
		{"flag value with space", "torchrun --nproc_per_node 8 train.py", "train.py", "train.py"},
		{"accelerate launch", "accelerate launch --num_processes 2 src/main.py", "src/main.py", "src/main.py"},
		{"deepspeed", "deepspeed train.py", "train.py", "train.py"},
		{"bash script", "bash scripts/train.sh", "scripts/train.sh", "scripts/train.sh"},
		{"sh script", "sh scripts/train.sh", "scripts/train.sh", "scripts/train.sh"},
		{"environment assignment prefix", "NCCL_DEBUG=INFO python train.py", "train.py", "train.py"},
		{"trailing newline", "python train.py\n", "train.py", "train.py"},
		{"path before a comment", "python train.py # entrypoint", "train.py", "train.py"},
		{"code source path variable", `python "$CODE_SOURCE_PATH/train.py"`, "", ""},
		{"absolute path", "python /opt/train.py", "", ""},
		{"module", "python -m mod --config configs/base.yaml", "", ""},
		{"inline code", `bash -c "python train.py"`, "", ""},
		{"bundled inline code flag", `bash -lc 'python train.py'`, "", ""},
		{"sh bundled inline code flag", `sh -ec 'python train.py'`, "", ""},
		{"bash login flag before script", "bash -l scripts/train.sh", "scripts/train.sh", "scripts/train.sh"},
		{"not in snapshot", "python other.py configs/base.yaml", "", ""},
		{"path-like flag value first", "torchrun --master_addr 10.0.0.1 train.py", "", ""},
		{"parent directory", "python ../tools/prep.py", "", ""},
		{"not a launcher", "./train.py --epochs 1", "", ""},
		{"echo", `echo "Launching train.py"`, "", ""},
		{"env -C", `env -C "$CODE_SOURCE_PATH" python train.py`, "", ""},
		{"uv run --directory", `uv run --directory "$CODE_SOURCE_PATH" python train.py`, "", ""},
		{"wrapper", "./launch.sh train.py", "", ""},
		{"after an earlier command", `pip install -r "$CODE_SOURCE_PATH/req.txt" && python train.py`, "", ""},
		{"semicolon", "python train.py; echo done", "", ""},
		{"pipe", "python train.py | tee log.txt", "", ""},
		{"background", "python train.py & wait", "", ""},
		{"cd into code source first", `cd "$CODE_SOURCE_PATH" && python train.py`, "", ""},
		{"subshell cd", `(cd "$CODE_SOURCE_PATH" && python train.py --epochs 1)`, "", ""},
		{"pushd", `pushd "$CODE_SOURCE_PATH"; python train.py`, "", ""},
		{"quoted cd in bash -lc", `bash -lc 'cd "$CODE_SOURCE_PATH" && python train.py'`, "", ""},
		{"command substitution", "python $(echo train.py)", "", ""},
		{"backticks", "python `echo train.py`", "", ""},
		{"multi-line command", "pip install -r requirements.txt\npython train.py", "", ""},
		{"comment", `python "$CODE_SOURCE_PATH/train.py" # was train.py`, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &runConfig{Command: &tc.command}
			got := relativeScriptWarning(cfg, snapshot)
			if tc.word == "" {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, "Warning: command may refer to \""+tc.word+"\"")
			assert.Contains(t, got, "If it does, \"$CODE_SOURCE_PATH/"+tc.suggest+"\" is safer.")
		})
	}
}

func TestRelativeScriptWarningContainers(t *testing.T) {
	head := `python "$CODE_SOURCE_PATH/train.py"`
	worker := "python train.py"
	cfg := &runConfig{Containers: []containerConfig{
		{Name: "head", Command: &head},
		{Name: "worker", Command: &worker},
	}}
	got := relativeScriptWarning(cfg, []string{"train.py"})
	assert.Contains(t, got, "Warning: containers[worker].command may refer to \"train.py\"")
	assert.Contains(t, got, "If it does, \"$CODE_SOURCE_PATH/train.py\" is safer.")
	assert.Contains(t, got, "https://docs.databricks.com/aws/en/machine-learning/ai-runtime/connecting/cli/yaml-config#command")
	assert.Contains(t, got, "https://learn.microsoft.com/azure/databricks/machine-learning/ai-runtime/connecting/cli/yaml-config#command")
}
