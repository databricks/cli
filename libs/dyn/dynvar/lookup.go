package dynvar

import (
	"errors"

	"github.com/databricks/cli/libs/dyn"
)

// Lookup is the type of lookup functions that can be used with [Resolve].
type Lookup func(path dyn.Path) (dyn.Value, error)

// ErrSkipResolution is returned by a lookup function to indicate that the
// resolution of a variable reference should be skipped.
var ErrSkipResolution = errors.New("skip resolution")

// DefaultLookup is the default lookup function used by [Resolve].
func DefaultLookup(in dyn.Value) Lookup {
	return func(path dyn.Path) (dyn.Value, error) {
		return dyn.GetByPath(in, path)
	}
}
