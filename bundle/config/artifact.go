package config

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ArtifactType string

const ArtifactPythonWheel ArtifactType = `whl`

type ArtifactFile struct {
	Source     string             `json:"source"`
	RemotePath string             `json:"-" bundle:"readonly"`
	Libraries  []*compute.Library `json:"-" bundle:"readonly"`
}

// Artifact defines a single local code artifact that can be
// built/uploaded/referenced in the context of this bundle.
type Artifact struct {
	Type ArtifactType `json:"type"`

	// The local path to the directory with a root of artifact,
	// for example, where setup.py is for Python projects
	Path string `json:"path"`

	// The relative or absolute path to the built artifact files
	// (Python wheel, Java jar and etc) itself
	Files        []ArtifactFile `json:"files"`
	BuildCommand string         `json:"build"`
}

func (a *Artifact) Build(ctx context.Context) ([]byte, error) {
	if a.BuildCommand == "" {
		return nil, fmt.Errorf("no build property defined")
	}

	out := make([][]byte, 0)
	commands := strings.Split(a.BuildCommand, " && ")
	for _, command := range commands {
		buildParts := strings.Split(command, " ")
		cmd := exec.CommandContext(ctx, buildParts[0], buildParts[1:]...)
		cmd.Dir = a.Path
		res, err := cmd.CombinedOutput()
		if err != nil {
			return res, err
		}
		out = append(out, res)
	}
	return bytes.Join(out, []byte{}), nil
}

func (a *Artifact) NormalisePaths() {
	for _, f := range a.Files {
		// If no libraries attached, nothing to normalise, skipping
		if f.Libraries == nil {
			continue
		}

		wsfsBase := "/Workspace"
		remotePath := path.Join(wsfsBase, f.RemotePath)
		for i := range f.Libraries {
			lib := f.Libraries[i]
			switch a.Type {
			case ArtifactPythonWheel:
				lib.Whl = remotePath
			}
		}

	}
}

// This function determines if artifact files needs to be uploaded.
// During the bundle processing we analyse which library uses which artifact file.
// If artifact file is used as a library, we store the reference to this library in artifact file Libraries field.
// If artifact file has libraries it's been used in, it means than we need to upload this file.
// Otherwise this artifact file is not used and we skip uploading
func (af *ArtifactFile) NeedsUpload() bool {
	return af.Libraries != nil
}
