package resources

import "github.com/databricks/cli/libs/dyn"

// LocalPath declares a field of a resource type that holds a path to a local
// file or directory. The path is made relative to the bundle root when the
// configuration is loaded, and is translated as Mode says at the point When says.
type LocalPath struct {
	// Pattern matches the field, relative to a single resource of the type.
	Pattern dyn.Pattern

	// Mode says how the path is translated.
	Mode TranslateMode

	// When says at which point the path is translated.
	When TranslateWhen

	// Filter restricts which values of the field are local paths.
	Filter LocalPathFilter

	// AllowOutsideSyncRoot allows the path to point outside the sync root.
	// Libraries can, for example, because they are uploaded on their own
	// instead of being synchronized with the bundle files.
	AllowOutsideSyncRoot bool
}

// TranslateWhen says at which point a local path is translated.
type TranslateWhen int

const (
	// TranslateAfterLoad translates the path once the configuration is fully
	// loaded, in the TranslatePaths mutator.
	TranslateAfterLoad TranslateWhen = iota

	// TranslateOnLoad translates the path of a resource declared in YAML right
	// after its paths are normalized, in the TranslatePathsDashboards mutator,
	// because initializing the resource reads the file at the path.
	TranslateOnLoad

	// TranslateNever leaves the path relative to the bundle root.
	TranslateNever
)

// LocalPathFilter restricts which values of a field are local paths.
type LocalPathFilter int

const (
	// AllLocalPaths treats every value of the field as a local path.
	AllLocalPaths LocalPathFilter = iota

	// OnlyLocalLibraries treats only values that refer to a local library as local paths.
	// Other values, such as PyPI package names, are left as they are.
	OnlyLocalLibraries

	// OnlyPipFlagPaths treats only pip flags with a local path, such as
	// "-r ./requirements.txt", as local paths.
	OnlyPipFlagPaths
)

// LocalPathDeclarer is implemented by resource types with fields that hold local paths.
type LocalPathDeclarer interface {
	// LocalPaths returns the fields in the order in which they are visited.
	LocalPaths() []LocalPath
}

// LocalPathSkipper is implemented by resource types where the fields declared
// through [LocalPathDeclarer] do not hold local paths for some resources.
type LocalPathSkipper interface {
	// SkipLocalPaths reports whether the declared fields of this resource are left as they are.
	SkipLocalPaths() bool
}
