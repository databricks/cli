package pipelines

import (
	"fmt"
	"strings"
)

// validates a fully-qualified table name and returns it with each
// segment backtick-quoted. Accepts two-part (schema.table, Hive)
// and three-part (catalog.schema.table, Unity Catalog) names.
func quoteTableName(name string) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", fmt.Errorf("invalid table name %q: expected catalog.schema.table (Unity Catalog) or schema.table (legacy Hive metastore)", name)
	}

	quoted := make([]string, len(parts))
	for i, part := range parts {
		if part == "" {
			return "", fmt.Errorf("invalid table name %q: empty identifier segment", name)
		}
		quoted[i] = "`" + strings.ReplaceAll(part, "`", "``") + "`"
	}
	return strings.Join(quoted, "."), nil
}
