package config

import (
	"github.com/databricks/cli/libs/exec"
)

type Artifacts map[string]*Artifact

type ArtifactType string

const ArtifactPythonWheel ArtifactType = `whl`

const ArtifactJar ArtifactType = `jar`

// ArtifactTarball is a gzipped tar of source files, built by DABs itself from
// `include` paths and/or a `git` ref rather than by a user `build` command.
// Uploaded and referenced like any other artifact file (e.g. as an AI Runtime
// task's code_source_path).
const ArtifactTarball ArtifactType = `tgz`

// Values returns all valid ArtifactType values
func (ArtifactType) Values() []ArtifactType {
	return []ArtifactType{
		ArtifactPythonWheel,
		ArtifactJar,
		ArtifactTarball,
	}
}

// ArtifactGit pins a `tgz` artifact to a git ref, so the tarball is a snapshot of
// that ref rather than of the working tree. Commit wins over Branch when both set.
type ArtifactGit struct {
	Branch string `json:"branch,omitempty"`
	Commit string `json:"commit,omitempty"`
}

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

	// Include lists bundle-root-relative paths to pack into a `tgz` artifact,
	// filtered like bundle file sync (.gitignore-honored). It composes files from
	// anywhere in the bundle (e.g. a code dir plus a sibling env file), so entries
	// stay bundle-root-relative — broader than the air CLI's code-source-root
	// include. Mutually exclusive with `build`.
	Include []string `json:"include,omitempty"`

	// Git, when set on a `tgz` artifact, snapshots the given ref instead of the
	// working tree. Mutually exclusive with `build`.
	Git *ArtifactGit `json:"git,omitempty"`
}
