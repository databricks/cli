package config

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ArtifactType string

const ArtifactPythonWheel ArtifactType = `whl`

// Artifact defines a single local code artifact that can be
// built/uploaded/referenced in the context of this bundle.
type Artifact struct {
	Type ArtifactType `json:"type"`

	// The local path to the directory with a root of artifact,
	// for example, where setup.py is for Python projects
	Path string `json:"path"`

	// The relative or absolute path to the built artifact file
	// (Python wheel, Java jar and etc) itself
	File         string `json:"file"`
	BuildCommand string `json:"build"`
	RemotePath   string `json:"-" bundle:"readonly"`
}

func (a *Artifact) Build(ctx context.Context) ([]byte, error) {
	if a.BuildCommand == "" {
		return nil, fmt.Errorf("no build property defined")
	}

	buildParts := strings.Split(a.BuildCommand, " ")
	cmd := exec.CommandContext(ctx, buildParts[0], buildParts[1:]...)
	cmd.Dir = a.Path
	return cmd.CombinedOutput()
}

func (a *Artifact) Library() compute.Library {
	lib := compute.Library{}
	switch a.Type {
	case ArtifactPythonWheel:
		lib.Whl = a.RemotePath
	}

	return lib
}
