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
			filePath, hasFilePath := v.Get(filePathFieldName).AsString()
			ss := v.Get(serializedSpaceFieldName)

			if hasFilePath {
				// file_path and serialized_space are two ways to provide the same
				// content. Accepting both is ambiguous, so reject it instead of
				// silently picking one.
				if ss.IsValid() && ss.Kind() != dyn.KindNil {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "both file_path and serialized_space are set; specify only one",
						Locations: ss.Locations(),
					})
					return v, nil
				}
				contents, err := b.SyncRoot.ReadFile(filePath)
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to read serialized genie space from file_path %s: %w", filePath, err)
				}
				return dyn.Set(v, serializedSpaceFieldName, dyn.V(string(contents)))
			}

			// Marshal an inline structured serialized_space to a JSON string so
			// both config-side and state-side carry the same plain string.
			// Otherwise YAML decodes small ints as Go `int` while state JSON
			// round-trip decodes them as `float64`, and structdiff reports
			// false drift on every plan.
			switch ss.Kind() {
			case dyn.KindInvalid, dyn.KindNil, dyn.KindString:
				// KindInvalid means serialized_space is absent (neither it nor
				// file_path is set); leave it for backend validation to reject.
				return v, nil
			case dyn.KindMap, dyn.KindSequence:
				jsonBytes, err := json.Marshal(ss.AsAny())
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to marshal inline serialized_space: %w", err)
				}
				return dyn.Set(v, serializedSpaceFieldName, dyn.V(string(jsonBytes)))
			default:
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("serialized_space must be a string, map, or sequence, got %s", ss.Kind()),
					Locations: ss.Locations(),
				})
				return v, nil
			}
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
