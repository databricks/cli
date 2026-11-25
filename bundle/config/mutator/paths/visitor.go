package paths

import (
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/dyn"
)

// VisitFunc is called with every local path in the configuration and the field that declares it.
type VisitFunc func(path dyn.Path, field resources.LocalPath, value dyn.Value) (dyn.Value, error)

// localPath is a declared field with its pattern from the configuration root.
type localPath struct {
	pattern dyn.Pattern
	field   resources.LocalPath
}

// localPaths returns the fields of every resource type that implements
// [resources.LocalPathDeclarer], in canonical order, followed by those of artifacts.
func localPaths() []localPath {
	var out []localPath
	for key, declarer := range config.ResourceTypesImplementing[resources.LocalPathDeclarer]() {
		base := dyn.NewPattern(dyn.Key("resources"), dyn.Key(key), dyn.AnyKey())
		for _, field := range declarer.LocalPaths() {
			out = append(out, localPath{pattern: base.Append(field.Pattern...), field: field})
		}
	}
	for _, field := range artifactLocalPaths() {
		out = append(out, localPath{pattern: field.Pattern, field: field})
	}
	return out
}

// VisitPaths visits all local paths in bundle configuration
func VisitPaths(root dyn.Value, fn VisitFunc) (dyn.Value, error) {
	newRoot := root
	for _, lp := range localPaths() {
		var err error
		newRoot, err = dyn.MapByPattern(newRoot, lp.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			ok, err := isLocalPath(lp.field.Filter, v)
			if err != nil || !ok {
				return v, err
			}
			return fn(p, lp.field, v)
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return newRoot, nil
}

func isLocalPath(filter resources.LocalPathFilter, v dyn.Value) (bool, error) {
	switch filter {
	case resources.AllLocalPaths:
		return true, nil
	case resources.OnlyLocalLibraries:
		s, ok := v.AsString()
		return ok && libraries.IsLibraryLocal(s), nil
	case resources.OnlyPipFlagPaths:
		s, ok := v.AsString()
		if !ok {
			return false, nil
		}
		_, _, ok = libraries.IsLocalPathInPipFlag(s)
		return ok, nil
	default:
		return false, fmt.Errorf("unsupported local path filter: %d", filter)
	}
}
