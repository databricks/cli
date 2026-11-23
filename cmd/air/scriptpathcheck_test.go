package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRelativeScriptWarning(t *testing.T) {
	snapshot := []string{"train.py", "src/main.py", "configs/base.yaml"}
	cases := []struct {
		name    string
		command string
		word    string // the command word reported, "" when no warning is expected
		suggest string // the path suggested under $CODE_SOURCE_PATH
	}{
		{"bare script", "python train.py", "train.py", "train.py"},
		{"dot-slash script", "./train.py --epochs 1", "./train.py", "train.py"},
		{"quoted nested script", `torchrun --nproc_per_node=8 "src/main.py"`, "src/main.py", "src/main.py"},
		{"flag value with equals", "python -m mod --config=configs/base.yaml", "configs/base.yaml", "configs/base.yaml"},
		{"flag value with space", "python -m mod --config configs/base.yaml", "configs/base.yaml", "configs/base.yaml"},
		{"code source path variable", `python "$CODE_SOURCE_PATH/train.py"`, "", ""},
		{"absolute path", "python /opt/train.py", "", ""},
		{"environment assignment", "SCRIPT=train.py python -m mod", "", ""},
		{"not in snapshot", "python other.py", "", ""},
		{"local file outside include_paths", "python tools/prep.py", "", ""},
		{"parent directory", "python ../tools/prep.py", "", ""},
		{"directory", "ls src", "", ""},
		{"cd into code source first", `cd "$CODE_SOURCE_PATH" && python train.py`, "", ""},
		{"subshell cd", `(cd "$CODE_SOURCE_PATH" && python train.py --epochs 1)`, "", ""},
		{"pushd", `pushd "$CODE_SOURCE_PATH"; python train.py`, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &runConfig{Command: &tc.command}
			got := relativeScriptWarning(cfg, snapshot)
			if tc.word == "" {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, "Warning: command refers to \""+tc.word+"\"")
			assert.Contains(t, got, "Use \"$CODE_SOURCE_PATH/"+tc.suggest+"\".")
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
	assert.Contains(t, got, "Warning: containers[worker].command refers to \"train.py\"")
	assert.Contains(t, got, "Use \"$CODE_SOURCE_PATH/train.py\".")
}
