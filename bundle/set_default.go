package bundle

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/logdiag"
)

type setDefault struct {
	pattern dyn.Pattern
	key     dyn.Path
	value   any
}

func SetDefaultMutator(pattern dyn.Pattern, key string, value any) Mutator {
	return &setDefault{
		pattern: pattern,
		key:     dyn.NewPath(dyn.Key(key)),
		value:   value,
	}
}

func (m *setDefault) Name() string {
	return fmt.Sprintf("SetDefaultMutator(%v, %v, %v)", m.pattern, m.key, m.value)
}

func (m *setDefault) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return m.apply(v, dyn.EmptyPath, m.pattern)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// setDefaultPathError prefixes an error with the full path of the value it was
// raised on, because apply visits sub-values and dyn errors only carry the path
// relative to the value they were raised on.
type setDefaultPathError struct {
	path dyn.Path
	err  error
}

func (e setDefaultPathError) Error() string {
	return fmt.Sprintf("%s: %s", e.path, e.err)
}

// Unwrap lets wildcard visits and callers still classify the underlying dyn error.
func (e setDefaultPathError) Unwrap() error {
	return e.err
}

// apply walks the pattern one component at a time so that a pure reference
// anywhere along the way is skipped, not only where the pattern ends.
func (m *setDefault) apply(v dyn.Value, prefix dyn.Path, pattern dyn.Pattern) (dyn.Value, error) {
	// A default cannot be written into a pure reference to a whole block
	// (e.g. "${resources.jobs.base.schedule}"), so leave it to the referenced value.
	if s, ok := v.AsString(); ok && dynvar.IsPureVariableReference(s) {
		return v, nil
	}

	var nv dyn.Value
	var err error
	if len(pattern) == 0 {
		_, err = dyn.GetByPath(v, m.key)
		if dyn.IsNoSuchKeyError(err) {
			return dyn.SetByPath(v, m.key, dyn.V(m.value))
		}
		nv = v
	} else {
		nv, err = dyn.MapByPattern(v, pattern[:1], func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return m.apply(v, prefix.Append(p...), pattern[1:])
		})
	}
	if err == nil || len(prefix) == 0 {
		return nv, err
	}
	// Errors from deeper levels already carry their full path.
	if _, ok := errors.AsType[setDefaultPathError](err); ok {
		return nv, err
	}
	// The error is relative to v, so prefix it with the path of v itself.
	return nv, setDefaultPathError{path: prefix, err: err}
}

func SetDefault(ctx context.Context, b *Bundle, pattern string, value any) {
	pat, err := dyn.NewPatternFromString(pattern)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("internal error: invalid pattern: %s: %w", pattern, err))
		return
	}

	pat, key := pat.SplitKey()
	if pat == nil || key == "" {
		logdiag.LogError(ctx, fmt.Errorf("internal error: invalid pattern: %s", pattern))
		return
	}

	m := SetDefaultMutator(pat, key, value)
	ApplyContext(ctx, b, m)
}
