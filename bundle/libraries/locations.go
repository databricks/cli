package libraries

import (
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
)

// location is a declared library location with its pattern from the configuration root.
type location struct {
	resources.LibraryLocation

	// pattern matches the field from the configuration root.
	pattern dyn.Pattern
}

// locations returns the library locations of every resource type that implements
// [resources.LibraryDeclarer]: by resource type in canonical order, and for each
// type in the order of its declarations.
func locations() []location {
	var out []location
	for key, declarer := range config.ResourceTypesImplementing[resources.LibraryDeclarer]() {
		base := dyn.NewPattern(dyn.Key("resources"), dyn.Key(key), dyn.AnyKey())
		for _, l := range declarer.LibraryLocations() {
			out = append(out, location{LibraryLocation: l, pattern: base.Append(l.Pattern...)})
		}
	}
	return out
}

// valuePatterns returns the patterns of the values in the location that can
// name a local library: the "whl" and "jar" entries of the libraries in a list,
// every dependency in a list, or the file itself.
func (l location) valuePatterns() []dyn.Pattern {
	switch l.Kind {
	case resources.LibraryList:
		return []dyn.Pattern{
			l.pattern.Append(dyn.AnyIndex(), dyn.Key("whl")),
			l.pattern.Append(dyn.AnyIndex(), dyn.Key("jar")),
		}
	case resources.DependencyList:
		return []dyn.Pattern{l.pattern.Append(dyn.AnyIndex())}
	case resources.LibraryFile:
		return []dyn.Pattern{l.pattern}
	default:
		panic(fmt.Sprintf("unsupported library kind %d at %s", l.Kind, l.pattern))
	}
}
