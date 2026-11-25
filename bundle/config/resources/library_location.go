package resources

import "github.com/databricks/cli/libs/dyn"

// LibraryLocation declares a field of a resource type that holds libraries.
// Local libraries in the field are uploaded on deploy, and the field is updated
// to refer to them by their remote path.
//
// A library location is declared apart from the [LocalPath] fields of the type:
// it names the field that holds the libraries, such as a list, while a local
// path names the values that are translated. Not every library location is
// translated as a local path.
type LibraryLocation struct {
	// Pattern matches the field, relative to a single resource of the type.
	Pattern dyn.Pattern

	// Kind says what the field holds.
	Kind LibraryKind

	// SkipPatchedWheels leaves the wheels in the field as they are when a
	// dynamic_version artifact builds a patched wheel for them.
	SkipPatchedWheels bool
}

// LibraryKind says what a library location holds.
//
// Where the locations of several kinds are visited one kind after the other,
// they are visited in the order in which the kinds are declared here.
type LibraryKind int

const (
	// LibraryList is a list of library objects. Their "whl" and "jar" entries
	// may name local libraries. Glob patterns in these entries are expanded.
	LibraryList LibraryKind = iota

	// DependencyList is a list of dependency strings, which may name local
	// libraries with pip extras, such as "./dist/foo.whl[extra]". Glob patterns
	// in these strings are expanded.
	DependencyList

	// LibraryFile is a single local file, such as an archive. Unlike the
	// libraries in lists, it is not expanded as a glob pattern, not switched to
	// a patched wheel and not checked for duplicate names.
	LibraryFile
)

// LibraryDeclarer is implemented by resource types with fields that hold libraries.
type LibraryDeclarer interface {
	// LibraryLocations returns the fields in the order in which they are visited.
	LibraryLocations() []LibraryLocation
}
