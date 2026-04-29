package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

// tagsStyleKeys mirrors ucm/config/generate.tagsStyleKeys: keys whose string
// values must round-trip as double-quoted YAML so numeric-looking values
// don't decay to bare scalars. Matches by key name across nesting depths.
var tagsStyleKeys = map[string]yaml.Style{
	"tags":       yaml.DoubleQuotedStyle,
	"options":    yaml.DoubleQuotedStyle,
	"properties": yaml.DoubleQuotedStyle,
}

// getKey returns the parent's --key flag if set, otherwise derives a sane
// key from the resource name. UC FQNs (a.b.c) become a_b_c so the key is a
// valid map key in ucm.yml without further hand-editing.
func getKey(cmd *cobra.Command, fallbackName string) string {
	if f := cmd.Flag("key"); f != nil && f.Value.String() != "" {
		return f.Value.String()
	}
	return strings.ReplaceAll(fallbackName, ".", "_")
}

// writeResourceYAML marshals a single typed UC resource into the
// `resources.<kind>.<key>` shape expected by the ucm.yml loader and writes it
// to <outputDir>/<kind>_<key>.yml. The output is a self-contained fragment
// the user can either drop next to ucm.yml (and pick up via `include`) or
// merge by hand.
//
// Returns the absolute path that was written.
func writeResourceYAML(outputDir, kind, key string, resource any, force bool) (string, error) {
	abs, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output dir: %w", err)
	}

	v, err := convert.FromTyped(resource, dyn.NilValue)
	if err != nil {
		return "", fmt.Errorf("convert resource to dyn: %w", err)
	}

	// yamlsaver special-cases map[string]dyn.Value at the top level — pass
	// the wrapped shape directly so its key-ordering and style logic apply.
	payload := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			kind: dyn.V(map[string]dyn.Value{
				key: v,
			}),
		}),
	}

	outPath := filepath.Join(abs, fmt.Sprintf("%s_%s.yml", kind, key))
	saver := yamlsaver.NewSaverWithStyle(tagsStyleKeys)
	if err := saver.SaveAsYAML(payload, outPath, force); err != nil {
		return "", err
	}
	return outPath, nil
}
