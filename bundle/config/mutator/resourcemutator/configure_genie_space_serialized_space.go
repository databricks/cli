package resourcemutator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

const serializedSpaceFieldName = "serialized_space"

type configureGenieSpaceSerializedSpace struct{}

func ConfigureGenieSpaceSerializedSpace() bundle.Mutator {
	return &configureGenieSpaceSerializedSpace{}
}

func (c configureGenieSpaceSerializedSpace) Name() string {
	return "ConfigureGenieSpaceSerializedSpace"
}

func (c configureGenieSpaceSerializedSpace) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("genie_spaces"),
		dyn.AnyKey(),
	)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if path, ok := v.Get(filePathFieldName).AsString(); ok {
				contents, err := b.SyncRoot.ReadFile(path)
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to read serialized genie space from file_path %s: %w", path, err)
				}
				return dyn.Set(v, serializedSpaceFieldName, dyn.V(string(contents)))
			}

			// Marshal an inline structured serialized_space to a JSON string so
			// both config-side and state-side carry the same plain string.
			// Otherwise YAML decodes small ints as Go `int` while state JSON
			// round-trip decodes them as `float64`, and structdiff reports
			// false drift on every plan.
			ss := v.Get(serializedSpaceFieldName)
			switch ss.Kind() {
			case dyn.KindMap, dyn.KindSequence:
				jsonBytes, err := json.Marshal(ss.AsAny())
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to marshal inline serialized_space: %w", err)
				}
				return dyn.Set(v, serializedSpaceFieldName, dyn.V(string(jsonBytes)))
			}
			return v, nil
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
