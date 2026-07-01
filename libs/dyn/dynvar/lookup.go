package dynvar

import (
	"errors"

	"github.com/databricks/cli/libs/dyn"
)

// Lookup is the type of lookup functions that can be used with [Resolve].
type Lookup func(path dyn.Path) (dyn.Value, error)

// SuggestFn is called when a variable reference cannot be resolved.
// It receives the original reference key (e.g., "bundle.nme") and returns
// a suggested correction (e.g., "bundle.name"), or "" if no suggestion is found.
type SuggestFn func(key string) string

// ResolveOption configures the [Resolve] function.
type ResolveOption func(*resolver)

// WithSuggestFn sets a suggestion function that is called when a reference
// cannot be resolved due to a missing key. The suggestion is appended to
// the error message as "Did you mean ${...}?".
func WithSuggestFn(fn SuggestFn) ResolveOption {
	return func(r *resolver) {
		r.suggestFn = fn
	}
}

// ErrSkipResolution is returned by a lookup function to indicate that the
// resolution of a variable reference should be skipped.
var ErrSkipResolution = errors.New("skip resolution")

// DefaultLookup is the default lookup function used by [Resolve].
func DefaultLookup(in dyn.Value) Lookup {
	return func(path dyn.Path) (dyn.Value, error) {
		return dyn.GetByPath(in, path)
	}
}
