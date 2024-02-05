package config

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/libs/exec"
)

type Artifacts map[string]*Artifact

func (artifacts Artifacts) SetConfigFilePath(path string) {
	for _, artifact := range artifacts {
		artifact.ConfigFilePath = path
	}
}

type ArtifactType string

const ArtifactPythonWheel ArtifactType = `whl`

type ArtifactFile struct {
	Source     string `json:"source"`
	RemotePath string `json:"remote_path" bundle:"readonly"`
}

// Artifact defines a single local code artifact that can be
// built/uploaded/referenced in the context of this bundle.
type Artifact struct {
	Type ArtifactType `json:"type"`

	// The local path to the directory with a root of artifact,
	// for example, where setup.py is for Python projects
	Path string `json:"path,omitempty"`

	// The relative or absolute path to the built artifact files
	// (Python wheel, Java jar and etc) itself
	Files        []ArtifactFile `json:"files,omitempty"`
	BuildCommand string         `json:"build,omitempty"`

	Executable exec.ExecutableType `json:"executable,omitempty"`

	paths.Paths
}

func (a *Artifact) Build(ctx context.Context) ([]byte, error) {
	if a.BuildCommand == "" {
		return nil, fmt.Errorf("no build property defined")
	}

	var e *exec.Executor
	var err error
	if a.Executable != "" {
		e, err = exec.NewCommandExecutorWithExecutable(a.Path, a.Executable)
	} else {
		e, err = exec.NewCommandExecutor(a.Path)
		a.Executable = e.ShellType()
	}
	if err != nil {
		return nil, err
	}
	return e.Exec(ctx, a.BuildCommand)
}
