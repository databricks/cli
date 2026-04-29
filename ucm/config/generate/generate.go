// Package generate emits ucm.yml bytes from an in-memory config.Root.
//
// The emitter converts the typed Root into a dyn.Value, flattens its top-level
// mapping into a map[string]dyn.Value (the shape libs/dyn/yamlsaver
// special-cases), and hands it off to yamlsaver.SaveAsYAML.
package generate

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/ucm/config"
	"go.yaml.in/yaml/v3"
)

// TagsStyleKeys lists the config keys whose string values the emitter should
// double-quote. yamlsaver matches by key name across every nesting level,
// so "tags" / "options" / "properties" are covered regardless of where they
// appear. Exported so the per-kind subcommands in cmd/ucm/generate can
// share the same style map.
var TagsStyleKeys = map[string]yaml.Style{
	"tags":       yaml.DoubleQuotedStyle,
	"options":    yaml.DoubleQuotedStyle,
	"properties": yaml.DoubleQuotedStyle,
}

// SaveToFile writes r as YAML to filename. If force is false and filename
// already exists, it returns an error without overwriting.
func SaveToFile(r *config.Root, filename string, force bool) error {
	data, err := rootToMap(r)
	if err != nil {
		return err
	}
	saver := yamlsaver.NewSaverWithStyle(TagsStyleKeys)
	return saver.SaveAsYAML(data, filename, force)
}

// rootToMap converts r into a map[string]dyn.Value keyed by top-level field
// name. That shape is what yamlsaver.SaveAsYAML is designed to consume —
// passing a typed struct directly would wrap it opaquely and panic in the
// YAML encoder.
func rootToMap(r *config.Root) (map[string]dyn.Value, error) {
	if r == nil {
		return nil, fmt.Errorf("nil config root")
	}
	v, err := convert.FromTyped(r, dyn.NilValue)
	if err != nil {
		return nil, fmt.Errorf("convert root to dyn: %w", err)
	}
	m, ok := v.AsMap()
	if !ok {
		return nil, fmt.Errorf("root did not convert to a mapping")
	}
	out := make(map[string]dyn.Value, m.Len())
	for _, p := range m.Pairs() {
		key, ok := p.Key.AsString()
		if !ok {
			return nil, fmt.Errorf("root key is not a string: %v", p.Key)
		}
		out[key] = p.Value
	}
	return out, nil
}
