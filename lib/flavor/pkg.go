package flavor

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/libraries"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/databricks-sdk-go/workspaces"
)

type Project interface {
	Root() string
	WorkspacesClient() *workspaces.WorkspacesClient
	GetDevelopmentClusterId(ctx context.Context) (clusterId string, err error)
}

type Flavor interface {
	// Detected returns true on successful metadata checks
	Detected(Project) bool

	// LocalArtifacts show (cached) relevant files that _should_ exist
	// on local filesystem
	LocalArtifacts(context.Context, Project) (Artifacts, error)
}

type Notebook struct {
	LocalAbsolute  string
	RemoteRelative string
}

type Artifact struct {
	libraries.Library
	Notebook *Notebook
	Flavor   Flavor
}

type notebookLanguageFormat struct {
	Language  workspace.ImportLanguage
	Format    workspace.ImportFormat
	Overwrite bool
}

var extMap = map[string]notebookLanguageFormat{
	".scala": {"SCALA", "SOURCE", true},
	".py":    {"PYTHON", "SOURCE", true},
	".sql":   {"SQL", "SOURCE", true},
	".r":     {"R", "SOURCE", true},
	".dbc":   {"", "DBC", false},
}

func (a Artifact) IsLibrary() bool {
	return a.Library.String() != "unknown"
}

func (a Artifact) NotebookInfo() (*notebookLanguageFormat, bool) {
	if a.Notebook == nil {
		return nil, false
	}
	ext := strings.ToLower(filepath.Ext(a.Notebook.LocalAbsolute))
	f, ok := extMap[ext]
	return &f, ok
}

type Kind int

const (
	LocalNotebook Kind = iota
	LocalJar
	LocalWheel
	LocalEgg
	RegistryLibrary
)

func (k Kind) RequiresBuild() bool {
	switch k {
	case LocalJar, LocalWheel, LocalEgg:
		return true
	default:
		return false
	}
}

func (a Artifact) KindAndLocation() (Kind, string) {
	if a.Notebook != nil {
		return LocalNotebook, a.Notebook.LocalAbsolute
	}
	if a.Jar != "" {
		return LocalJar, a.Jar
	}
	if a.Whl != "" {
		return LocalWheel, a.Whl
	}
	if a.Egg != "" {
		return LocalEgg, a.Egg
	}
	return RegistryLibrary, ""
}

type Artifacts []Artifact

func (a Artifacts) RequiresBuild() bool {
	for _, v := range a {
		k, _ := v.KindAndLocation()
		if k.RequiresBuild() {
			return true
		}
	}
	return false
}

func (a Artifacts) HasLibraries() bool {
	for _, v := range a {
		if v.IsLibrary() {
			return true
		}
	}
	return false
}
