package bundle

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// Default is a value to set at every path matching Pattern where it is not set yet.
type Default struct {
	Pattern string
	Value   any
}

type defaultValue struct {
	pattern *structpath.PatternNode
	key     string
	value   any
}

type setDefaults struct {
	defaults []defaultValue
}

func (m *setDefaults) Name() string {
	return "SetDefaults"
}

func (m *setDefaults) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	for _, d := range m.defaults {
		err := structvar.ForEach(b.Config.View(), d.pattern, func(p *structpath.PathNode, v structvar.View) error {
			if v.Get(d.key).IsValid() {
				return nil
			}
			return b.Config.Set(structpath.NewStringKey(p, d.key), d.value)
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// SetDefaults sets each default value at every path matching its pattern where no value is set.
// Defaults are applied in order.
func SetDefaults(ctx context.Context, b *Bundle, defaults []Default) {
	m := &setDefaults{}
	for _, d := range defaults {
		pat, err := structpath.ParsePattern(d.Pattern)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("internal error: invalid pattern: %s: %w", d.Pattern, err))
			return
		}

		key, ok := pat.StringKey()
		if !ok || key == "" {
			logdiag.LogError(ctx, fmt.Errorf("internal error: invalid pattern: %s", d.Pattern))
			return
		}

		m.defaults = append(m.defaults, defaultValue{pattern: pat.Parent(), key: key, value: d.Value})
	}

	ApplyContext(ctx, b, m)
}
