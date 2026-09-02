package structwalk_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShadowedEmbedIsVisitedTwice shows the core bug: structwalk visits a
// field that appears at the same JSON path through two different embedded
// struct chains without noticing that one shadows the other.
//
// resources.Pipeline embeds both BaseResource and pipelines.CreatePipeline,
// and both declare json:"id". encoding/json calls that ambiguous and drops the
// field entirely, so neither field is ever serialized. structwalk visits both
// paths anyway and emits "id" twice.
//
// resources.Cluster embeds BaseResource (Lifecycle.PreventDestroy) alongside
// its own Lifecycle *LifecycleWithStarted (which also embeds Lifecycle); again
// two paths to "lifecycle.prevent_destroy" and both are visited.
func TestShadowedEmbedIsVisitedTwice(t *testing.T) {
	// ----- Pipeline: id -----
	// Fill both shadowed fields with non-zero values so neither is
	// dropped by the omitempty skip that masks the bug at zero value.
	pipe := &resources.Pipeline{}
	pipe.BaseResource.ID = "base-id"  //nolint:staticcheck // explicit: sets the shadowed BaseResource field, not the promoted CreatePipeline.Id
	pipe.CreatePipeline.Id = "sdk-id" //nolint:staticcheck // explicit: sets the promoted SDK field, not the shadowed BaseResource.ID

	pipeVisits := map[string]int{}
	require.NoError(t, structwalk.Walk(pipe, func(path *structpath.PathNode, _ any, _ *reflect.StructField) {
		pipeVisits[path.String()]++
	}))

	// encoding/json emits nothing for "id" because both fields declare it at
	// the same embedding depth → ambiguous. The walk should match that: zero
	// visits. Two visits is the bug.
	var blob []byte
	blob, _ = json.Marshal(pipe)
	var pipeJSON map[string]any
	require.NoError(t, json.Unmarshal(blob, &pipeJSON))

	jsonEmitsID := pipeJSON["id"] != nil
	walkVisitsID := pipeVisits["id"]
	assert.Equal(t, jsonEmitsID, walkVisitsID > 0,
		"structwalk and encoding/json disagree about whether 'id' exists on resources.Pipeline; "+
			"json emits=%v, walk visits=%d times", jsonEmitsID, walkVisitsID)

	// ----- Cluster: lifecycle.prevent_destroy -----
	clus := &resources.Cluster{}
	clus.BaseResource.Lifecycle = resources.Lifecycle{PreventDestroy: true} //nolint:staticcheck // explicit: sets the embedded BaseResource field shadowed by Cluster.Lifecycle
	clus.Lifecycle = &resources.LifecycleWithStarted{Lifecycle: resources.Lifecycle{PreventDestroy: true}}

	clusVisits := map[string]int{}
	require.NoError(t, structwalk.Walk(clus, func(path *structpath.PathNode, _ any, _ *reflect.StructField) {
		clusVisits[path.String()]++
	}))

	blob, _ = json.Marshal(clus)
	var clusJSON map[string]any
	require.NoError(t, json.Unmarshal(blob, &clusJSON))

	// encoding/json serializes Cluster.Lifecycle (shallower named field) and
	// drops BaseResource.Lifecycle (deeper, shadowed). The walk visits both.
	t.Logf("cluster lifecycle.prevent_destroy: json=%v, walk visits=%d",
		nestedGet(clusJSON, "lifecycle", "prevent_destroy"),
		clusVisits["lifecycle.prevent_destroy"])
}

// TestShadowedEmbedCausesStructdiffDuplicate shows what happens downstream:
// when both shadowed fields are non-zero and they differ, structdiff emits the
// same path twice — once per shadowed declaration. prepareChanges in the direct
// engine uses a map and so the second entry silently overwrites the first, but
// which value wins is arbitrary and depends on iteration order in the embedding.
func TestShadowedEmbedCausesStructdiffDuplicate(t *testing.T) {
	before := &resources.Pipeline{}
	before.BaseResource.ID = "old-base-id"  //nolint:staticcheck // explicit: sets the shadowed BaseResource field
	before.CreatePipeline.Id = "old-sdk-id" //nolint:staticcheck // explicit: sets the promoted SDK field

	after := &resources.Pipeline{}
	after.BaseResource.ID = "new-base-id"  //nolint:staticcheck // explicit: sets the shadowed BaseResource field
	after.CreatePipeline.Id = "new-sdk-id" //nolint:staticcheck // explicit: sets the promoted SDK field

	changes, err := structdiff.GetStructDiff(before, after, nil)
	require.NoError(t, err)

	paths := map[string]int{}
	for _, ch := range changes {
		if ch.Path != nil {
			paths[ch.Path.String()]++
		}
	}

	var dupes []string
	for p, n := range paths {
		if n > 1 {
			dupes = append(dupes, p)
		}
	}
	slices.Sort(dupes)

	// encoding/json says "id" is ambiguous and drops it, so a diff library
	// that agrees with encoding/json would report zero changes at "id".
	// Reporting it twice is the bug; zero is correct.
	if len(dupes) > 0 {
		t.Logf("structdiff reports the following paths more than once: %v", dupes)
		t.Logf("encoding/json would emit neither entry: they are ambiguous fields")
	}
	// Assert the current (buggy) behavior so a fix requires updating this test.
	assert.Equal(t, []string{"id"}, dupes,
		"expected 'id' to be reported twice (shadowed embeds, both non-zero); "+
			"if this fails the fix is working — remove the duplicate from this assertion")
}

func nestedGet(m map[string]any, keys ...string) any {
	var v any = m
	for _, k := range keys {
		mm, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = mm[k]
	}
	return v
}
