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

// sawtoothMaterializedFeatureMembers builds the three materialized features a sawtooth-window
// feature needs to declare for its materialization: a batch offline materialized feature, a batch
// online materialized feature, and a streaming online materialized feature, where the batch offline
// and batch online use a DERIVED cron schedule and the streaming online and batch online must have
// the same online destination.
//
// materialized_feature_id is absent here because it is assigned server side.
func sawtoothMaterializedFeatureMembers(entry resources.SawtoothMaterializedFeatures) []ml.MaterializedFeature {
	return []ml.MaterializedFeature{
		// Batch offline.
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OfflineStoreConfig:    entry.OfflineStoreConfig,
			CronSchedule:          entry.CronSchedule,
			CronScheduleTrigger:   entry.CronScheduleTrigger,
			TableTrigger:          entry.TableTrigger,
		},
		// Batch online.
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OnlineStoreConfig:     entry.OnlineStoreConfig,
			CronSchedule:          entry.CronSchedule,
			CronScheduleTrigger:   entry.CronScheduleTrigger,
			TableTrigger:          entry.TableTrigger,
		},
		// Streaming online.
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OnlineStoreConfig:     entry.OnlineStoreConfig,
			StreamingMode:         entry.StreamingMode,
		},
	}
}

// batchOfflineAndOnlineMaterializedFeatureMembers builds two materializations of the specified
// feature to the online and offline destination.
func batchOfflineAndOnlineMaterializedFeatureMembers(entry resources.BatchOfflineAndOnlineMaterializedFeatures) []ml.MaterializedFeature {
	return []ml.MaterializedFeature{
		// Batch offline.
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OfflineStoreConfig:    entry.OfflineStoreConfig,
			CronSchedule:          entry.CronSchedule,
			CronScheduleTrigger:   entry.CronScheduleTrigger,
			TableTrigger:          entry.TableTrigger,
		},
		// Batch online.
		{
			FeatureName:           entry.FeatureName,
			Tags:                  entry.Tags,
			BudgetPolicyId:        entry.BudgetPolicyId,
			Notifications:         entry.Notifications,
			PipelineScheduleState: entry.PipelineScheduleState,
			OnlineStoreConfig:     entry.OnlineStoreConfig,
			CronSchedule:          entry.CronSchedule,
			CronScheduleTrigger:   entry.CronScheduleTrigger,
			TableTrigger:          entry.TableTrigger,
		},
	}
}

// declaredMaterializedFeature is an expanded member together with the entry that declared it, so a
// duplicate can name where it was written.
type declaredMaterializedFeature struct {
	member ml.MaterializedFeature
	path   *structpath.PathNode
}

// declaredMaterializationSetMembers is every member the set asks for, in the order the request
// carries them: the members written out, then each shorthand entry's expansion.
func declaredMaterializationSetMembers(
	setKey string,
	set *resources.MaterializationSet,
) []declaredMaterializedFeature {
	setPath := structpath.NewPath(nil, "resources", "materialization_sets", setKey)
	var declared []declaredMaterializedFeature

	for i, member := range set.MaterializedFeatures {
		declared = append(declared, declaredMaterializedFeature{
			member: member,
			path:   structpath.NewPath(setPath, "materialized_features", i),
		})
	}
	for i, entry := range set.SawtoothMaterializedFeatures {
		path := structpath.NewPath(setPath, "sawtooth_materialized_features", i)
		for _, member := range sawtoothMaterializedFeatureMembers(entry) {
			declared = append(declared, declaredMaterializedFeature{member: member, path: path})
		}
	}
	for i, entry := range set.BatchOfflineAndOnlineMaterializedFeatures {
		path := structpath.NewPath(setPath, "batch_offline_and_online_materialized_features", i)
		for _, member := range batchOfflineAndOnlineMaterializedFeatureMembers(entry) {
			declared = append(declared, declaredMaterializedFeature{member: member, path: path})
		}
	}

	return declared
}

// quotedFeatureNames renders a feature name list the way FeatureStoreMessages.quotedIds does.
func quotedFeatureNames(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = "'" + name + "'"
	}
	return strings.Join(quoted, ", ")
}

// duplicateMemberKeyDiagnostic rejects a set asking for two members with the same member key, which
// is the identity resources.MaterializedFeatureMemberKey builds and the server's own
// MaterializationSetMemberKey shares.
//
// It mirrors MaterializationSetUtils.validateMaterializationSetMemberKeysAreUnique: group the
// members by key, then name every feature holding a key more than one member shares, once and
// sorted.
func duplicateMemberKeyDiagnostic(
	b *bundle.Bundle,
	setKey string,
	declared []declaredMaterializedFeature,
) *diag.Diagnostic {
	membersByKey := map[string][]declaredMaterializedFeature{}
	for _, entry := range declared {
		_, memberKey := resources.MaterializedFeatureMemberKey(entry.member)
		membersByKey[memberKey] = append(membersByKey[memberKey], entry)
	}

	featureNames := map[string]bool{}
	pathsByString := map[string]*structpath.PathNode{}
	for _, members := range membersByKey {
		if len(members) < 2 {
			continue
		}
		for _, entry := range members {
			featureNames[entry.member.FeatureName] = true
			pathsByString[entry.path.String()] = entry.path
		}
	}
	if len(featureNames) == 0 {
		return nil
	}

	var paths []*structpath.PathNode
	var locations []diag.Location
	for _, pathString := range slices.Sorted(maps.Keys(pathsByString)) {
		path := pathsByString[pathString]
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

// unsettableShorthandFields names the member fields the entry carries but the shape cannot use. A
// promoted field cannot be removed from the generated schema, so a shorthand rejects them here
// instead of omitting them.
func unsettableShorthandFields(member ml.MaterializedFeature, batchPair bool) []string {
	var fields []string
	if member.MaterializedFeatureId != "" {
		fields = append(fields, "materialized_feature_id")
	}
	if batchPair && member.StreamingMode != nil {
		fields = append(fields, "streaming_mode")
	}
	return fields
}

// unsettableFieldDiagnostics rejects each shorthand entry setting a field its shape cannot use,
// naming the entry through Paths so a reader is sent to the line that set it.
func unsettableFieldDiagnostics(b *bundle.Bundle, setKey string, set *resources.MaterializationSet) diag.Diagnostics {
	setPath := structpath.NewPath(nil, "resources", "materialization_sets", setKey)

	var diags diag.Diagnostics
	report := func(field, key string, index int, shape string) {
		path := structpath.NewPath(setPath, key, index)
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary: fmt.Sprintf("%s must not be set on a %s: %s", field, shape,
				unsettableFieldReasons[field]),
			Paths:     []*structpath.PathNode{path},
			Locations: b.Config.GetLocationsOf(path),
		})
	}

	for i, entry := range set.SawtoothMaterializedFeatures {
		for _, field := range unsettableShorthandFields(entry.MaterializedFeature, false) {
			report(field, "sawtooth_materialized_features", i, "sawtooth materialized feature")
		}
	}
	for i, entry := range set.BatchOfflineAndOnlineMaterializedFeatures {
		for _, field := range unsettableShorthandFields(entry.MaterializedFeature, true) {
			report(field, "batch_offline_and_online_materialized_features", i,
				"batch offline and online materialized feature")
		}
	}

	return diags
}

// unsettableFieldReasons is why each field cannot be set, in the message naming it.
var unsettableFieldReasons = map[string]string{
	"materialized_feature_id": "the server assigns one per expanded member",
	"streaming_mode":          "a feature on the batch path has none",
}

// Apply mutates typed state, and the Python path merges the dynamic tree back once the normalize
// sequence returns, so this must not be the last entry in applyNormalizeMutators: that tree picks up
// a typed mutation only when the next mutator enters.
//
// The duplicate check belongs here rather than in a mutator of its own: this clears both shorthand
// fields, and the dynamic tree is rebuilt from typed state on the next mutator's entry, so only
// while this runs can a diagnostic name the entry an author wrote.
func (m *normalizeMaterializationSetMaterializedFeatureMembers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map iteration order is randomized; sort by name for stable output.
	for _, setKey := range slices.Sorted(maps.Keys(b.Config.Resources.MaterializationSets)) {
		set := b.Config.Resources.MaterializationSets[setKey]
		if set == nil {
			continue
		}

		diags = diags.Extend(unsettableFieldDiagnostics(b, setKey, set))

		declared := declaredMaterializationSetMembers(setKey, set)
		if duplicate := duplicateMemberKeyDiagnostic(b, setKey, declared); duplicate != nil {
			diags = diags.Append(*duplicate)
		}

		// Assigned only when a shorthand entry contributed, so a set that writes every member out
		// keeps the list it declared, empty or unset as written.
		if len(set.SawtoothMaterializedFeatures) > 0 ||
			len(set.BatchOfflineAndOnlineMaterializedFeatures) > 0 {
			members := make([]ml.MaterializedFeature, 0, len(declared))
			for _, entry := range declared {
				members = append(members, entry.member)
			}
			set.MaterializedFeatures = members
		}

		set.SawtoothMaterializedFeatures = nil
		set.BatchOfflineAndOnlineMaterializedFeatures = nil
	}

	return diags
}
