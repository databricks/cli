package resourcemutator

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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

	spaces := b.Config.Resources.GenieSpaces
	for _, name := range slices.Sorted(maps.Keys(spaces)) {
		g := spaces[name]
		if g == nil {
			continue
		}
		ssPath := structpath.NewPath(nil, "resources", "genie_spaces", name, serializedSpaceFieldName)

		if g.FilePath != "" {
			// file_path and serialized_space are two ways to provide the same
			// content. Accepting both is ambiguous, so reject it instead of
			// silently picking one.
			if g.SerializedSpace != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "both file_path and serialized_space are set; specify only one",
					Locations: b.Config.GetLocationsOfExact(ssPath),
				})
				continue
			}
			contents, err := b.SyncRoot.ReadFile(g.FilePath)
			if err != nil {
				return diags.Extend(diag.FromErr(fmt.Errorf("failed to read serialized genie space from file_path %s: %w", g.FilePath, err)))
			}
			if err := b.Config.Set(ssPath, string(contents)); err != nil {
				return diags.Extend(diag.FromErr(err))
			}
			continue
		}

		// Marshal an inline structured serialized_space to a JSON string so
		// both config-side and state-side carry the same plain string.
		// Otherwise YAML decodes small ints as Go `int` while state JSON
		// round-trip decodes them as `float64`, and structdiff reports
		// false drift on every plan. A top-level sequence would be valid JSON
		// but is meaningless for a genie space, so it is rejected.
		s, ok, kind, err := inlineJSON(g.SerializedSpace)
		if err != nil {
			return diags.Extend(diag.FromErr(fmt.Errorf("failed to marshal inline serialized_space: %w", err)))
		}
		if kind != "" {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "serialized_space must be a string or map, got " + kind,
				Locations: b.Config.GetLocationsOfExact(ssPath),
			})
			continue
		}
		if ok {
			if err := b.Config.Set(ssPath, s); err != nil {
				return diags.Extend(diag.FromErr(err))
			}
		}
	}

	return diags
}
