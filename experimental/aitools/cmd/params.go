package aitools

import (
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/sql"
)

// parseParams converts --param flag values into SDK parameter list items for
// the Databricks SQL Statement Execution API. Each input is either
// "name=value" (defaults to STRING server-side) or "name:TYPE=value" (typed,
// e.g. "since:DATE=2026-01-01"). An empty value becomes NULL on the wire
// because StatementParameterListItem.Value uses omitempty.
//
// The Databricks API only supports named markers (`:name`), not positional
// `?`, and parameter names must be unique within a statement.
func parseParams(raw []string) ([]sql.StatementParameterListItem, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	out := make([]sql.StatementParameterListItem, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, s := range raw {
		lhs, value, ok := strings.Cut(s, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --param %q: expected name=value or name:TYPE=value", s)
		}

		name, typ, _ := strings.Cut(lhs, ":")
		name = strings.TrimSpace(name)
		typ = strings.TrimSpace(typ)

		if name == "" {
			return nil, fmt.Errorf("invalid --param %q: name is empty", s)
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("duplicate --param name %q", name)
		}
		seen[name] = struct{}{}

		out = append(out, sql.StatementParameterListItem{
			Name:  name,
			Type:  typ,
			Value: value,
		})
	}
	return out, nil
}
