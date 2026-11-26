package libraries

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
)

// location is a declared library location with its pattern from the configuration root.
type location struct {
	resources.LibraryLocation

	// rootPattern matches the field from the configuration root.
	rootPattern dyn.Pattern
}

// locations returns the library locations of every resource type that implements
// [resources.LibraryDeclarer]: by resource type in canonical order, and for each
// type in the order of its declarations.
func locations() []location {
	var out []location
	for key, declarer := range config.ResourceTypesImplementing[resources.LibraryDeclarer]() {
		base := dyn.NewPattern(dyn.Key("resources"), dyn.Key(key), dyn.AnyKey())
		for _, l := range declarer.LibraryLocations() {
			out = append(out, location{LibraryLocation: l, rootPattern: base.Append(l.Pattern...)})
		}
	}
	return out
}

// locationsByKind returns the locations of [locations] ordered by kind, and
// within a kind in the order of [locations].
func locationsByKind() []location {
	out := locations()
	slices.SortStableFunc(out, func(a, b location) int {
		return cmp.Compare(a.Kind, b.Kind)
	})
	return out
}

// valuePatterns returns the patterns of the values in the location that can
// name a local library: the "whl" and "jar" entries of the libraries in a list,
// every dependency in a list, or the file itself.
func (l location) valuePatterns() []dyn.Pattern {
	switch l.Kind {
	case resources.LibraryList:
		return []dyn.Pattern{
			l.rootPattern.Append(dyn.AnyIndex(), dyn.Key("whl")),
			l.rootPattern.Append(dyn.AnyIndex(), dyn.Key("jar")),
		}
	case resources.DependencyList:
		return []dyn.Pattern{l.rootPattern.Append(dyn.AnyIndex())}
	case resources.LibraryFile:
		return []dyn.Pattern{l.rootPattern}
	default:
		panic(fmt.Sprintf("unsupported library kind %d at %s", l.Kind, l.rootPattern))
	}
}
