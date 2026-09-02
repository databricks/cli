package direct

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// escapedRefPrefix is the escape for a literal "${...}" in bundle configuration:
// the leading "$" tells the bundle not to treat what follows as a reference.
const escapedRefPrefix = "$${"

// unescapeRefs rewrites "$${" to "${" in every string field of state, in place.
// Configs use "$${...}" for placeholders that the Databricks runtime interprets,
// so the API must receive them with a single "$".
func unescapeRefs(state any) error {
	var paths []*structpath.PathNode
	var values []string

	err := structwalk.Walk(state, func(path *structpath.PathNode, val any, _ *reflect.StructField) {
		s, ok := val.(string)
		if !ok || !strings.Contains(s, escapedRefPrefix) {
			return
		}
		paths = append(paths, path)
		values = append(values, strings.ReplaceAll(s, escapedRefPrefix, "${"))
	})
	if err != nil {
		return err
	}

	for i, path := range paths {
		if err := structaccess.Set(state, path, values[i]); err != nil {
			return fmt.Errorf("unescaping %s: %w", path, err)
		}
	}

	return nil
}
