package config

import (
	"github.com/databricks/cli/libs/exec"
)

type Artifacts map[string]*Artifact

type ArtifactType string

const ArtifactPythonWheel ArtifactType = `whl`

type ArtifactFile struct {
	Source string `json:"source"`

	// Patched is populated if DynamicVersion is set and patching was successful
	Patched string `json:"patched,omitempty" bundle:"readonly"`

	RemotePath string `json:"remote_path,omitempty" bundle:"readonly"`
}

// Artifact defines a single local code artifact that can be
// built/uploaded/referenced in the context of this bundle.
type Artifact struct {
	Type ArtifactType `json:"type,omitempty"`

	// The local path to the directory with a root of artifact,
	// for example, where setup.py is for Python projects
	Path string `json:"path,omitempty"`

	// The relative or absolute path to the built artifact files
	// (Python wheel, Java jar and etc) itself
	Files        []ArtifactFile `json:"files,omitempty"`
	BuildCommand string         `json:"build,omitempty"`

	Executable exec.ExecutableType `json:"executable,omitempty"`

	DynamicVersion bool `json:"dynamic_version,omitempty"`
}
