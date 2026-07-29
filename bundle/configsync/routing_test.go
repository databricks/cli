package configsync

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYamlFileIndexUsesColumnsForFlowStyleSequences(t *testing.T) {
	sequence := []dyn.Value{
		dyn.NewValue(nil, []dyn.Location{{File: "config.yml", Line: 10, Column: 12}}),
		dyn.NewValue(nil, []dyn.Location{{File: "config.yml", Line: 10, Column: 42}}),
	}

	assert.Equal(t, 0, yamlFileIndex(sequence, 0))
	assert.Equal(t, 1, yamlFileIndex(sequence, 1))
}

func TestSourceRefForScopeRejectsMultiplePhysicalBlocks(t *testing.T) {
	refs := []sourceRef{
		{file: "first.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"))},
		{file: "second.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"))},
	}

	_, _, err := sourceRefForScope(refs, "dev", true)
	require.ErrorContains(t, err, "multiple physical source blocks")
}

func TestChooseSequenceBlockRejectsInsertionAtBlockBoundary(t *testing.T) {
	top := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	target := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	plan := sequencePlan{
		pairs: []sequencePair{{oldIndex: 0, newIndex: 0}, {oldIndex: 1, newIndex: 2}},
		adds:  []int{1},
	}

	_, err := chooseSequenceBlock(1, plan, map[int]sourceRef{0: top, 1: target}, nil)
	require.ErrorContains(t, err, "no unique physical source block")
}

func TestChooseSequenceBlockPlacesAppendAfterLastPhysicalElement(t *testing.T) {
	top := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	target := sourceRef{file: "databricks.yml", path: dyn.NewPath(dyn.Key("targets"), dyn.Key("dev"), dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key("example"), dyn.Key("clusters"), dyn.Index(0))}
	plan := sequencePlan{
		pairs: []sequencePair{{oldIndex: 0, newIndex: 0}, {oldIndex: 1, newIndex: 1}},
		adds:  []int{2},
	}

	block, err := chooseSequenceBlock(2, plan, map[int]sourceRef{0: top, 1: target}, nil)
	require.NoError(t, err)
	assert.Equal(t, blockForRef(target), block)
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

func TestPlanSequenceChangesRejectsUnidentifiableSurvivor(t *testing.T) {
	oldValues := []any{
		map[string]any{"value": "top"},
		map[string]any{"value": "target"},
	}
	newValues := []any{
		map[string]any{"value": "edited"},
	}

	_, err := planSequenceChanges("items", oldValues, newValues)
	require.ErrorContains(t, err, "sequence correspondence is ambiguous")
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
	require.ErrorContains(t, err, "sequence correspondence is ambiguous")
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
