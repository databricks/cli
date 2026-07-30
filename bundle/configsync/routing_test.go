package configsync

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChooseSequenceBlockUsesBroadestScopeAtBlockBoundary(t *testing.T) {
	top := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	target := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	plan := sequencePlan{
		pairs: []sequencePair{{oldIndex: 0, newIndex: 0}, {oldIndex: 1, newIndex: 2}},
		adds:  []int{1},
	}

	block, err := chooseSequenceBlock(1, plan, map[int][]sourceRef{0: {top}, 1: {target}}, nil, "dev")
	require.NoError(t, err)
	assert.Equal(t, blockForRef(top), block)
}

func TestChooseSequenceBlockPlacesAppendAfterLastPhysicalElement(t *testing.T) {
	top := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	target := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	plan := sequencePlan{
		pairs: []sequencePair{{oldIndex: 0, newIndex: 0}, {oldIndex: 1, newIndex: 1}},
		adds:  []int{2},
	}

	block, err := chooseSequenceBlock(2, plan, map[int][]sourceRef{0: {top}, 1: {target}}, nil, "dev")
	require.NoError(t, err)
	assert.Equal(t, blockForRef(target), block)
}

func TestChooseSequenceBlockUsesTopLevelForSharedSurvivor(t *testing.T) {
	top := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	target := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	plan := sequencePlan{
		pairs: []sequencePair{{oldIndex: 0, newIndex: 0}},
		adds:  []int{1},
	}

	block, err := chooseSequenceBlock(1, plan, map[int][]sourceRef{0: {top, target}}, nil, "dev")
	require.NoError(t, err)
	assert.Equal(t, blockForRef(top), block)
}

func TestPairRenamesUsesExactContentInsteadOfShiftedIndices(t *testing.T) {
	changes := ResourceChanges{
		"tasks[task_key='m']": {
			Operation:   OperationRemove,
			configValue: map[string]any{"task_key": "m", "value": "top"},
		},
		"tasks[task_key='a']": {
			Operation:   OperationRemove,
			configValue: map[string]any{"task_key": "a", "value": "target"},
		},
		"tasks[task_key='a_new_target']": {
			Operation: OperationAdd,
			Value:     map[string]any{"task_key": "a_new_target", "value": "target"},
		},
		"tasks[task_key='z_new_top']": {
			Operation: OperationAdd,
			Value:     map[string]any{"task_key": "z_new_top", "value": "top"},
		},
		"tasks[task_key='brand_new']": {
			Operation: OperationAdd,
			Value:     map[string]any{"task_key": "brand_new", "value": "new"},
		},
	}

	pairs, pairedRemoves, unresolved := pairRenames(changes)
	assert.Empty(t, unresolved)
	assert.Equal(t, "a", pairs["tasks[task_key='a_new_target']"].oldKey)
	assert.Equal(t, "m", pairs["tasks[task_key='z_new_top']"].oldKey)
	_, paired := pairs["tasks[task_key='brand_new']"]
	assert.False(t, paired)
	assert.Len(t, pairedRemoves, 2)
}

func TestPairRenamesReportsIndistinguishableMappings(t *testing.T) {
	changes := ResourceChanges{
		"tasks[task_key='top']": {
			Operation:   OperationRemove,
			configValue: map[string]any{"task_key": "top", "value": "same"},
		},
		"tasks[task_key='target']": {
			Operation:   OperationRemove,
			configValue: map[string]any{"task_key": "target", "value": "same"},
		},
		"tasks[task_key='one']": {
			Operation: OperationAdd,
			Value:     map[string]any{"task_key": "one", "value": "same"},
		},
		"tasks[task_key='two']": {
			Operation: OperationAdd,
			Value:     map[string]any{"task_key": "two", "value": "same"},
		},
	}

	pairs, pairedRemoves, unresolved := pairRenames(changes)
	assert.Empty(t, pairs)
	assert.Empty(t, pairedRemoves)
	require.Len(t, unresolved, 1)
	assert.ElementsMatch(t, []string{"tasks[task_key='top']", "tasks[task_key='target']"}, unresolved[0].removePaths)
	assert.ElementsMatch(t, []string{"tasks[task_key='one']", "tasks[task_key='two']"}, unresolved[0].addPaths)
}

func TestPairRenamesReportsSingleRenameWithFieldEditAsAmbiguous(t *testing.T) {
	changes := ResourceChanges{
		"tasks[task_key='task']": {
			Operation: OperationRemove,
			configValue: map[string]any{
				"task_key":        "task",
				"timeout_seconds": int64(10),
				"notebook_task":   map[string]any{"notebook_path": "/Workspace/unchanged"},
			},
		},
		"tasks[task_key='task_renamed']": {
			Operation: OperationAdd,
			Value: map[string]any{
				"task_key":        "task_renamed",
				"timeout_seconds": int64(20),
				"notebook_task":   map[string]any{"notebook_path": "/Workspace/unchanged"},
			},
		},
	}

	pairs, pairedRemoves, unresolved := pairRenames(changes)
	assert.Empty(t, pairs)
	assert.Empty(t, pairedRemoves)
	require.Len(t, unresolved, 1)
	assert.Equal(t, []string{"tasks[task_key='task']"}, unresolved[0].removePaths)
	assert.Equal(t, []string{"tasks[task_key='task_renamed']"}, unresolved[0].addPaths)
}

func TestPairRenamesReportsAmbiguousRemoveAndAddWithSharedNestedFields(t *testing.T) {
	changes := ResourceChanges{
		"tasks[task_key='removed']": {
			Operation: OperationRemove,
			configValue: map[string]any{
				"task_key":        "removed",
				"job_cluster_key": "shared_cluster",
				"timeout_seconds": int64(10),
				"notebook_task":   map[string]any{"notebook_path": "/Workspace/shared"},
			},
		},
		"tasks[task_key='added']": {
			Operation: OperationAdd,
			Value: map[string]any{
				"task_key":        "added",
				"job_cluster_key": "shared_cluster",
				"timeout_seconds": int64(20),
				"notebook_task":   map[string]any{"notebook_path": "/Workspace/shared"},
			},
		},
	}

	pairs, pairedRemoves, unresolved := pairRenames(changes)
	assert.Empty(t, pairs)
	assert.Empty(t, pairedRemoves)
	require.Len(t, unresolved, 1)
	assert.Equal(t, []string{"tasks[task_key='removed']"}, unresolved[0].removePaths)
	assert.Equal(t, []string{"tasks[task_key='added']"}, unresolved[0].addPaths)
}

func TestChangeUsesUnresolvedRename(t *testing.T) {
	taskRename := unresolvedRenameGroup{
		key:     "task_key",
		oldKeys: []string{"etl"},
		newKeys: []string{"etl_new"},
	}

	tests := []struct {
		name   string
		path   string
		change *ConfigChangeDesc
		group  unresolvedRenameGroup
		want   bool
	}{
		{
			name:   "dependency selector",
			path:   "tasks[task_key='consumer'].depends_on[task_key='etl']",
			change: &ConfigChangeDesc{Operation: OperationRemove},
			group:  taskRename,
			want:   true,
		},
		{
			name: "nested keyed value",
			path: "tasks[task_key='consumer'].depends_on[task_key='etl_new']",
			change: &ConfigChangeDesc{
				Operation: OperationAdd,
				Value:     map[string]any{"task_key": "etl_new"},
			},
			group: taskRename,
			want:  true,
		},
		{
			name:   "scalar keyed reference",
			path:   "tasks[task_key='consumer'].job_cluster_key",
			change: &ConfigChangeDesc{Operation: OperationReplace, configValue: "etl", Value: "etl_new"},
			group: unresolvedRenameGroup{
				key:     "job_cluster_key",
				oldKeys: []string{"etl"},
				newKeys: []string{"etl_new"},
			},
			want: true,
		},
		{
			name:   "unrelated scalar",
			path:   "timeout_seconds",
			change: &ConfigChangeDesc{Operation: OperationReplace, configValue: int64(10), Value: int64(20)},
			group:  taskRename,
		},
		{
			name:   "unrelated matching string",
			path:   "name",
			change: &ConfigChangeDesc{Operation: OperationReplace, configValue: "old", Value: "etl_new"},
			group:  taskRename,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, changeUsesUnresolvedRename(test.path, test.change, test.group))
		})
	}
}

func TestPlanSequenceChangesPreservesSurvivorsAcrossLengthChanges(t *testing.T) {
	oldValues := []any{
		map[string]any{"label": "default"},
		map[string]any{"label": "maintenance"},
	}
	newValues := []any{
		map[string]any{"label": "default"},
		map[string]any{"label": "maintenance"},
		map[string]any{"label": "extra"},
	}

	plan, err := planSequenceChanges("clusters", oldValues, newValues)
	require.NoError(t, err)
	assert.Equal(t, []sequencePair{
		{oldIndex: 0, newIndex: 0, equal: true},
		{oldIndex: 1, newIndex: 1, equal: true},
	}, plan.pairs)
	assert.Empty(t, plan.removals)
	assert.Equal(t, []int{2}, plan.adds)
}

func TestPlanSequenceChangesMatchesEditedSurvivorByUniqueLabel(t *testing.T) {
	oldValues := []any{
		map[string]any{"label": "default", "num_workers": int64(1)},
		map[string]any{"label": "maintenance", "num_workers": int64(2)},
	}
	newValues := []any{
		map[string]any{"label": "maintenance", "num_workers": int64(20)},
	}

	plan, err := planSequenceChanges("clusters", oldValues, newValues)
	require.NoError(t, err)
	assert.Equal(t, []sequencePair{{oldIndex: 1, newIndex: 0}}, plan.pairs)
	assert.Equal(t, []int{0}, plan.removals)
	assert.Empty(t, plan.adds)
}

func TestPlanSequenceChangesReplacesClusterLabel(t *testing.T) {
	oldValues := []any{map[string]any{"label": "default"}}
	newValues := []any{map[string]any{"label": "maintenance"}}

	plan, err := planSequenceChanges("clusters", oldValues, newValues)
	require.NoError(t, err)
	assert.Equal(t, []sequencePair{{oldIndex: 0, newIndex: 0}}, plan.pairs)
	assert.Empty(t, plan.removals)
	assert.Empty(t, plan.adds)
}

func TestPlanSequenceChangesMatchesUniqueClusterRenameByContent(t *testing.T) {
	oldValues := []any{
		map[string]any{"label": "default", "num_workers": int64(7)},
		map[string]any{"label": "maintenance", "num_workers": int64(8)},
		map[string]any{"label": "stable", "num_workers": int64(9)},
	}
	newValues := []any{
		map[string]any{"label": "renamed", "num_workers": int64(8)},
		map[string]any{"label": "stable", "num_workers": int64(9)},
	}

	plan, err := planSequenceChanges("clusters", oldValues, newValues)
	require.NoError(t, err)
	assert.Equal(t, []sequencePair{
		{oldIndex: 1, newIndex: 0},
		{oldIndex: 2, newIndex: 1, equal: true},
	}, plan.pairs)
	assert.Equal(t, []int{0}, plan.removals)
	assert.Empty(t, plan.adds)
}

func TestPlanSequenceChangesRejectsUnidentifiableSurvivor(t *testing.T) {
	oldValues := []any{
		map[string]any{"value": "top"},
		map[string]any{"value": "target"},
	}
	newValues := []any{
		map[string]any{"value": "edited"},
	}

	_, err := planSequenceChanges("items", oldValues, newValues)
	require.ErrorContains(t, err, "sequence elements cannot be matched uniquely")
}

func TestPlanSequenceChangesTracksShiftedScalarSurvivor(t *testing.T) {
	oldValues := []any{"literal", "resolved-variable"}
	newValues := []any{"resolved-variable"}

	plan, err := planSequenceChanges("dependencies", oldValues, newValues)
	require.NoError(t, err)
	assert.Equal(t, []sequencePair{{oldIndex: 1, newIndex: 0, equal: true}}, plan.pairs)
	assert.Equal(t, []int{0}, plan.removals)
	assert.Empty(t, plan.adds)
}

func TestPlanSequenceChangesTracksPrependedScalar(t *testing.T) {
	oldValues := []any{"resolved-variable"}
	newValues := []any{"literal", "resolved-variable"}

	plan, err := planSequenceChanges("dependencies", oldValues, newValues)
	require.NoError(t, err)
	assert.Equal(t, []sequencePair{{oldIndex: 0, newIndex: 1, equal: true}}, plan.pairs)
	assert.Empty(t, plan.removals)
	assert.Equal(t, []int{0}, plan.adds)
}

func TestPlanSequenceChangesHandlesEditedSequenceWithDuplicateValues(t *testing.T) {
	oldValues := []any{"same", "before", "same"}
	newValues := []any{"same", "after", "same", "added"}

	plan, err := planSequenceChanges("dependencies", oldValues, newValues)
	require.NoError(t, err)
	assert.Equal(t, []sequencePair{
		{oldIndex: 0, newIndex: 0, equal: true},
		{oldIndex: 1, newIndex: 1},
		{oldIndex: 2, newIndex: 2, equal: true},
	}, plan.pairs)
	assert.Empty(t, plan.removals)
	assert.Equal(t, []int{3}, plan.adds)
}

func TestPlanSequenceChangesRejectsAmbiguousDuplicateScalar(t *testing.T) {
	oldValues := []any{"same", "same"}
	newValues := []any{"same"}

	_, err := planSequenceChanges("dependencies", oldValues, newValues)
	require.ErrorContains(t, err, "sequence elements cannot be matched uniquely")
}

func TestPlanSequenceChangesKeepsPairSharedByAmbiguousMatches(t *testing.T) {
	oldValues := []any{
		"before",
		"stable-old",
		"middle",
		"duplicate",
		"duplicate",
		"after",
	}
	newValues := []any{
		"before",
		"stable-new",
		"middle",
		"duplicate",
		"after",
	}

	plan, err := planSequenceChanges("dependencies", oldValues, newValues)
	require.ErrorContains(t, err, "sequence elements cannot be matched uniquely")
	assert.Equal(t, []sequencePair{
		{oldIndex: 0, newIndex: 0, equal: true},
		{oldIndex: 1, newIndex: 1},
		{oldIndex: 2, newIndex: 2, equal: true},
		{oldIndex: 5, newIndex: 4, equal: true},
	}, plan.pairs)
	assert.Empty(t, plan.removals)
	assert.Empty(t, plan.adds)
}

func TestSequencePairCanApplyIndependently(t *testing.T) {
	oldValues := []any{
		map[string]any{"label": "renamed", "num_workers": int64(1)},
		map[string]any{"label": "stable", "num_workers": int64(2)},
	}
	newValues := []any{
		map[string]any{"label": "replacement", "num_workers": int64(1)},
		map[string]any{"label": "stable", "num_workers": int64(20)},
	}

	assert.False(t, sequencePairCanApplyIndependently("clusters", oldValues, newValues, sequencePair{oldIndex: 0, newIndex: 0}))
	assert.True(t, sequencePairCanApplyIndependently("clusters", oldValues, newValues, sequencePair{oldIndex: 1, newIndex: 1}))

	oldDependencies := []any{"A", "A", "B"}
	newDependencies := []any{"B", "B", "A", "C"}
	assert.False(t, sequencePairCanApplyIndependently(
		"dependencies",
		oldDependencies,
		newDependencies,
		sequencePair{oldIndex: 2, newIndex: 3},
	))

	oldDependencies = []any{"before", "stable-old", "duplicate", "duplicate", "after"}
	newDependencies = []any{"before", "stable-new", "duplicate", "after"}
	assert.True(t, sequencePairCanApplyIndependently(
		"dependencies",
		oldDependencies,
		newDependencies,
		sequencePair{oldIndex: 1, newIndex: 1},
	))
}

func TestPlanSequenceChangesBoundsAmbiguousDuplicateMatches(t *testing.T) {
	oldValues := make([]any, 64)
	newValues := make([]any, 63)
	for index := range oldValues {
		oldValues[index] = "same"
	}
	for index := range newValues {
		newValues[index] = "same"
	}

	_, err := planSequenceChanges("dependencies", oldValues, newValues)
	require.ErrorContains(t, err, "sequence elements cannot be matched uniquely")
}

func TestChooseSequenceBlockRejectsReplacementAcrossBlocks(t *testing.T) {
	top := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	target := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	plan := sequencePlan{removals: []int{0, 1}, adds: []int{0}}

	_, err := chooseSequenceBlock(0, plan, map[int][]sourceRef{0: {top}, 1: {target}}, nil, "dev")
	require.ErrorContains(t, err, "replacement spans multiple physical source blocks")
}

func TestChooseSequenceBlockRejectsReplacementAcrossBlocksWithSurvivor(t *testing.T) {
	top := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	target := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	plan := sequencePlan{
		pairs:    []sequencePair{{oldIndex: 2, newIndex: 1}},
		removals: []int{0, 1},
		adds:     []int{0},
	}

	_, err := chooseSequenceBlock(0, plan, map[int][]sourceRef{0: {top}, 1: {target}, 2: {target}}, nil, "dev")
	require.ErrorContains(t, err, "replacement spans multiple physical source blocks")
}

func TestChooseSequenceBlockRejectsReplacementMovedPastMatchedElement(t *testing.T) {
	top := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	target := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	plan := sequencePlan{
		pairs:    []sequencePair{{oldIndex: 0, newIndex: 1}, {oldIndex: 2, newIndex: 2}},
		removals: []int{1},
		adds:     []int{0},
	}

	_, err := chooseSequenceBlock(0, plan, map[int][]sourceRef{0: {top}, 1: {target}, 2: {target}}, nil, "dev")
	require.ErrorContains(t, err, "replacement spans multiple physical source blocks")
}

func TestSequenceInsertionIndexUsesFirstEntryOfNextLogicalElement(t *testing.T) {
	parent := dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"))
	block := sourceBlock{file: "databricks.yml", parent: parent}
	ref := func(index int) sourceRef {
		return sourceRef{file: block.file, path: parent.Append(dyn.Index(index))}
	}
	refsByNewIndex := map[int][]sourceRef{
		0: {ref(0), ref(2)},
		2: {ref(1)},
	}

	assert.Equal(t, 1, sequenceInsertionIndex(1, refsByNewIndex, nil, block))
	assert.Equal(t, 3, sequenceInsertionIndex(3, refsByNewIndex, nil, block))
}

func TestCoalesceSequenceElementAddsToMissingList(t *testing.T) {
	changes := []FieldChange{
		{
			FilePath:        "databricks.yml",
			FieldCandidates: []string{"resources.jobs.example.tasks"},
			Change: &ConfigChangeDesc{
				Operation:          OperationAdd,
				Value:              []any{map[string]any{"task_key": "alpha"}},
				sequenceElementAdd: true,
			},
		},
		{
			FilePath:        "databricks.yml",
			FieldCandidates: []string{"resources.jobs.example.tasks"},
			Change: &ConfigChangeDesc{
				Operation:          OperationAdd,
				Value:              []any{map[string]any{"task_key": "beta"}},
				sequenceElementAdd: true,
			},
		},
	}

	result, err := coalesceSequenceElementAdds(changes)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, []any{
		map[string]any{"task_key": "alpha"},
		map[string]any{"task_key": "beta"},
	}, result[0].Change.Value)
}
