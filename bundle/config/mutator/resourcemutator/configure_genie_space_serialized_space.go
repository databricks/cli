package resourcemutator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

	// Skip converting the configuration if there is nothing to configure.
	if len(b.Config.Resources.GenieSpaces) == 0 {
		return nil
	}

	pattern := structpath.MustParsePattern("resources.genie_spaces.*")

	err := structvar.ForEach(b.Config.View(), pattern, func(p *structpath.PathNode, v structvar.View) error {
		filePath, hasFilePath := v.Get(filePathFieldName).AsString()
		ss := v.Get(serializedSpaceFieldName)
		ssPath := structpath.NewStringKey(p, serializedSpaceFieldName)

		if hasFilePath {
			// file_path and serialized_space are two ways to provide the same
			// content. Accepting both is ambiguous, so reject it instead of
			// silently picking one.
			if ss.IsValid() && ss.Kind() != structvar.KindNil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "both file_path and serialized_space are set; specify only one",
					Locations: ss.Locations(),
				})
				return nil
			}
			contents, err := b.SyncRoot.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read serialized genie space from file_path %s: %w", filePath, err)
			}
			return b.Config.Set(ssPath, string(contents))
		}

		// Marshal an inline structured serialized_space to a JSON string so
		// both config-side and state-side carry the same plain string.
		// Otherwise YAML decodes small ints as Go `int` while state JSON
		// round-trip decodes them as `float64`, and structdiff reports
		// false drift on every plan.
		switch ss.Kind() {
		case structvar.KindInvalid, structvar.KindNil, structvar.KindString:
			// KindInvalid means serialized_space is absent (neither it nor
			// file_path is set); leave it for backend validation to reject.
			return nil
		case structvar.KindMap:
			// A top-level sequence would be valid JSON but is meaningless for a
			// genie space, so KindSequence is not accepted here and falls through
			// to the default rejection below.
			jsonBytes, err := json.Marshal(ss.AsAny())
			if err != nil {
				return fmt.Errorf("failed to marshal inline serialized_space: %w", err)
			}
			return b.Config.Set(ssPath, string(jsonBytes))
		default:
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("serialized_space must be a string or map, got %s", ss.Kind()),
				Locations: ss.Locations(),
			})
			return nil
		}
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
