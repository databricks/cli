package resourcemutator

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type normalizeMaterializationSetMaterializedFeatureMembers struct{}

func NormalizeMaterializationSetMaterializedFeatureMembers() bundle.Mutator {
	return &normalizeMaterializationSetMaterializedFeatureMembers{}
}

func (m *normalizeMaterializationSetMaterializedFeatureMembers) Name() string {
	return "NormalizeMaterializationSetMaterializedFeatureMembers"
}

// quotedFeatureNames renders a feature name list the way FeatureStoreMessages.quotedIds does.
func quotedFeatureNames(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = "'" + name + "'"
	}
	return strings.Join(quoted, ", ")
}

// duplicateMemberKeyDiagnostic rejects a set listing two members with the same member key, which is
// the identity resources.MaterializedFeatureMemberKey builds and the server's own
// MaterializationSetMemberKey shares.
//
// It mirrors MaterializationSetUtils.validateMaterializationSetMemberKeysAreUnique: group the
// members by key, then name every feature holding a key more than one member shares, once and
// sorted.
func duplicateMemberKeyDiagnostic(
	b *bundle.Bundle,
	setKey string,
	members []ml.MaterializedFeature,
) *diag.Diagnostic {
	indicesByKey := map[string][]int{}
	for i, member := range members {
		_, memberKey := resources.MaterializedFeatureMemberKey(member)
		indicesByKey[memberKey] = append(indicesByKey[memberKey], i)
	}

	featureNames := map[string]bool{}
	var indices []int
	for _, group := range indicesByKey {
		if len(group) < 2 {
			continue
		}
		for _, i := range group {
			featureNames[members[i].FeatureName] = true
			indices = append(indices, i)
		}
	}
	if len(featureNames) == 0 {
		return nil
	}

	setPath := structpath.NewPath(nil, "resources", "materialization_sets", setKey)
	slices.Sort(indices)

	var paths []*structpath.PathNode
	var locations []diag.Location
	for _, i := range indices {
		path := structpath.NewPath(setPath, "materialized_features", i)
		paths = append(paths, path)
		locations = append(locations, b.Config.GetLocationsOf(path)...)
	}

	return &diag.Diagnostic{
		Severity: diag.Error,
		Summary: fmt.Sprintf(
			"Features %s in materialization set '%s' each declares more than one materialized "+
				"feature with the following same fields: %s, which is not allowed.",
			quotedFeatureNames(slices.Sorted(maps.Keys(featureNames))), setKey,
			resources.MaterializedFeatureMemberKeyFieldDescription),
		Paths:     paths,
		Locations: locations,
	}
}

func (m *normalizeMaterializationSetMaterializedFeatureMembers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map iteration order is randomized; sort by name for stable output.
	for _, setKey := range slices.Sorted(maps.Keys(b.Config.Resources.MaterializationSets)) {
		set := b.Config.Resources.MaterializationSets[setKey]
		if set == nil {
			continue
		}

		if duplicate := duplicateMemberKeyDiagnostic(b, setKey, set.MaterializedFeatures); duplicate != nil {
			diags = diags.Append(*duplicate)
		}
	}

	return diags
}
