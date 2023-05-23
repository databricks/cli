package config

import "github.com/databricks/databricks-sdk-go/service/workspace"

// Artifact defines a single local code artifact that can be
// built/uploaded/referenced in the context of this bundle.
type Artifact struct {
	Notebook      *NotebookArtifact      `json:"notebook,omitempty"`
	PythonPackage *PythonPackageArtifact `json:"python_package,omitempty"`
}

type NotebookArtifact struct {
	Path string `json:"path"`

	// Language is detected during build step.
	Language workspace.Language `json:"language,omitempty" bundle:"readonly"`

	// Paths are synthesized during build step.
	LocalPath  string `json:"local_path,omitempty" bundle:"readonly"`
	RemotePath string `json:"remote_path,omitempty" bundle:"readonly"`
}

type PythonPackageArtifact struct {
	Path string `json:"path"`

	// Paths are synthesized during build step.
	LocalPath  string `json:"local_path,omitempty" bundle:"readonly"`
	RemotePath string `json:"remote_path,omitempty" bundle:"readonly"`
}
