package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

// VectorSearchIndexSpec validates that each vector_search_indexes resource
// declares the spec block matching its index_type. The CreateIndex API rejects
// mismatched combinations at deploy time; reject them at validate time so the
// failure is visible before the deploy starts.
func VectorSearchIndexSpec() bundle.ReadOnlyMutator {
	return &vectorSearchIndexSpec{}
}

type vectorSearchIndexSpec struct{ bundle.RO }

func (*vectorSearchIndexSpec) Name() string {
	return "validate:vector_search_index_spec"
}

func (*vectorSearchIndexSpec) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	root := dyn.NewPath(dyn.Key("resources"), dyn.Key("vector_search_indexes"))

	for name, idx := range b.Config.Resources.VectorSearchIndexes {
		if idx == nil {
			continue
		}
		path := root.Append(dyn.Key(name))
		diags = diags.Extend(validateVectorSearchIndexSpec(b, idx.IndexType, idx.DeltaSyncIndexSpec != nil, idx.DirectAccessIndexSpec != nil, path))
	}

	return diags
}

func validateVectorSearchIndexSpec(b *bundle.Bundle, indexType vectorsearch.VectorIndexType, hasDeltaSync, hasDirectAccess bool, path dyn.Path) diag.Diagnostics {
	switch indexType {
	case vectorsearch.VectorIndexTypeDeltaSync:
		if !hasDeltaSync {
			return missingSpecDiag(b, path, "delta_sync_index_spec", indexType)
		}
		if hasDirectAccess {
			return incompatibleSpecDiag(b, path, "direct_access_index_spec", indexType)
		}
	case vectorsearch.VectorIndexTypeDirectAccess:
		if !hasDirectAccess {
			return missingSpecDiag(b, path, "direct_access_index_spec", indexType)
		}
		if hasDeltaSync {
			return incompatibleSpecDiag(b, path, "delta_sync_index_spec", indexType)
		}
	}
	return nil
}

func missingSpecDiag(b *bundle.Bundle, path dyn.Path, field string, indexType vectorsearch.VectorIndexType) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("vector_search_indexes: missing %s for index_type %q", field, indexType),
		Locations: b.Config.GetLocations(path.String()),
		Paths:     []dyn.Path{path.Append(dyn.Key(field))},
	}}
}

func incompatibleSpecDiag(b *bundle.Bundle, path dyn.Path, field string, indexType vectorsearch.VectorIndexType) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("vector_search_indexes: %s is not allowed when index_type is %q", field, indexType),
		Locations: b.Config.GetLocations(path.String()),
		Paths:     []dyn.Path{path.Append(dyn.Key(field))},
	}}
}
