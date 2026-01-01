package resourcemutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

const (
	serializedSpaceFieldName = "serialized_space"
)

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

	// Configure serialized_space field for all genie spaces.
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Include "serialized_space" field if "file_path" is set.
			path, ok := v.Get(filePathFieldName).AsString()
			if !ok {
				return v, nil
			}

			// Warn if both file_path and serialized_space are set.
			existingSpace := v.Get(serializedSpaceFieldName)
			if existingSpace.Kind() != dyn.KindInvalid && existingSpace.Kind() != dyn.KindNil {
				resourceName := p[len(p)-1].Key()
				diags = diags.Append(diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("genie space %q has both file_path and serialized_space set; file_path takes precedence and serialized_space will be overwritten", resourceName),
				})
			}

			contents, err := b.SyncRoot.ReadFile(path)
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("failed to read serialized genie space from file_path %s: %w", path, err)
			}

			return dyn.Set(v, serializedSpaceFieldName, dyn.V(string(contents)))
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
