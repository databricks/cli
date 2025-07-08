package bundle

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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
		return dyn.MapByPattern(v, m.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			_, err := dyn.GetByPath(v, m.key)
			switch {
			case dyn.IsNoSuchKeyError(err):
				return dyn.SetByPath(v, m.key, dyn.V(m.value))
			default:
				return v, err
			}
		})
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func SetDefault(ctx context.Context, b *Bundle, pattern string, value any) {
	pat, err := dyn.NewPatternFromString(pattern)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("Internal error: invalid pattern: %s: %w", pattern, err))
		return
	}

	pat, key := pat.SplitKey()
	if pat == nil || key == "" {
		logdiag.LogError(ctx, fmt.Errorf("Internal error: invalid pattern: %s", pattern))
		return
	}

	m := SetDefaultMutator(pat, key, value)
	ApplyContext(ctx, b, m)
}
