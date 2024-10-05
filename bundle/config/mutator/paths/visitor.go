package paths

import "github.com/databricks/cli/libs/dyn"

type PathKind int

const (
	// PathKindLibrary is a path to a library file
	PathKindLibrary = iota

	// PathKindNotebook is a path to a notebook file
	PathKindNotebook

	// PathKindWorkspaceFile is a path to a regular workspace file,
	// notebooks are not allowed because they are uploaded a special
	// kind of workspace object.
	PathKindWorkspaceFile

	// PathKindWithPrefix is a path that starts with './'
	PathKindWithPrefix

	// PathKindDirectory is a path to directory
	PathKindDirectory
)

type VisitFunc func(path dyn.Path, kind PathKind, value dyn.Value) (dyn.Value, error)
