package structwalk_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The structs below mirror the embedding patterns in the real resource types
// from bundle/config/resources/ and the SDK. They are local copies so that
// these tests are independent of the bundle package. Comments name the originals.
//
// Embedded field types must be exported so that structwalk, structdiff, and
// reflect descend into them (unexported anonymous fields are skipped by
// sf.PkgPath != "" / sf.IsExported() checks).

// ShadowBase mirrors bundle/config/resources.BaseResource, which every
// resource struct embeds anonymously. It contributes both "id" and "lifecycle"
// at the same embedding depth as the SDK types below.
type ShadowBase struct {
	// bundle/config/resources.BaseResource.ID — json:"id,omitempty"
	ID string `json:"id,omitempty"`

	// bundle/config/resources.BaseResource.Lifecycle — json:"lifecycle,omitempty"
	Lifecycle ShadowLifecycle `json:"lifecycle,omitempty"`
}

// ShadowLifecycle mirrors bundle/config/resources.Lifecycle.
type ShadowLifecycle struct {
	PreventDestroy bool `json:"prevent_destroy,omitempty"`
}

// ShadowLifecycleWithStarted mirrors bundle/config/resources.LifecycleWithStarted,
// which extends Lifecycle with start/stop control for clusters and apps.
type ShadowLifecycleWithStarted struct {
	ShadowLifecycle
	Started *bool `json:"started,omitempty"`
}

// ShadowSDKPipeline mirrors the Id field that pipelines.CreatePipeline carries
// at the same json name "id" as ShadowBase.ID, creating the ambiguity under test.
type ShadowSDKPipeline struct {
	// pipelines.CreatePipeline.Id — json:"id,omitempty"
	Id string `json:"id,omitempty"` //nolint:revive // mirroring the SDK field name exactly
}

// ShadowPipeline mirrors the embedding shape of bundle/config/resources.Pipeline:
//
//	type Pipeline struct {
//	    BaseResource             // ← has ID string `json:"id,omitempty"`
//	    pipelines.CreatePipeline // ← also has Id string `json:"id,omitempty"`
//	    ...
//	}
//
// Two anonymous embeds at the same depth both declare "id", making it
// ambiguous: encoding/json drops the field entirely; structwalk visits it twice.
type ShadowPipeline struct {
	ShadowBase
	ShadowSDKPipeline //nolint:govet // the repeated json "id" tag is the point: both embeds declare it, creating the ambiguity under test
}

// ShadowCluster mirrors the embedding shape of bundle/config/resources.Cluster:
//
//	type Cluster struct {
//	    BaseResource                    // ← has Lifecycle `json:"lifecycle,omitempty"`
//	    compute.ClusterSpec             // ← no lifecycle field
//	    Lifecycle *LifecycleWithStarted `json:"lifecycle,omitempty"` // direct named field
//	    ...
//	}
//
// A direct named field (Lifecycle) shadows the same name promoted from
// BaseResource: encoding/json uses the named field only; structwalk visits both.
type ShadowCluster struct {
	ShadowBase
	// Direct named field — shallower than ShadowBase.Lifecycle, so
	// encoding/json uses this one.
	Lifecycle *ShadowLifecycleWithStarted `json:"lifecycle,omitempty"`
}

// TestShadowedEmbedIsVisitedTwice shows the core bug: structwalk visits a
// field that appears at the same JSON path through two different embedded
// struct chains, without noticing that one shadows the other.
func TestShadowedEmbedIsVisitedTwice(t *testing.T) {
	// ----- Pipeline-shape: id -----
	// Fill both shadowed fields with non-zero values so neither is dropped
	// by the omitempty skip that hides the bug at zero value.
	pipe := &ShadowPipeline{}
	pipe.ID = "base-id"
	pipe.Id = "sdk-id"

	pipeVisits := map[string]int{}
	require.NoError(t, structwalk.Walk(pipe, func(path *structpath.PathNode, _ any, _ *reflect.StructField) {
		pipeVisits[path.String()]++
	}))

	// encoding/json calls "id" ambiguous (both embeds at the same depth) and
	// drops it — neither field is serialized. structwalk visits it twice.
	// The assertions below record the current (buggy) behaviour so a fix
	// requires updating them.
	blob, _ := json.Marshal(pipe)
	var pipeJSON map[string]any
	require.NoError(t, json.Unmarshal(blob, &pipeJSON))

	// Bug: structwalk visits "id" twice; encoding/json emits it zero times.
	assert.Equal(t, 2, pipeVisits["id"], "structwalk should visit 'id' twice (bug); fix → 0")
	assert.Nil(t, pipeJSON["id"], "encoding/json must not emit the ambiguous 'id'")

	// ----- Cluster-shape: lifecycle.prevent_destroy -----
	clus := &ShadowCluster{}
	// Set the promoted (shadowed) field from ShadowBase directly.
	clus.ShadowBase.Lifecycle = ShadowLifecycle{PreventDestroy: true}
	// Set the direct named field that shadows it.
	clus.Lifecycle = &ShadowLifecycleWithStarted{ShadowLifecycle: ShadowLifecycle{PreventDestroy: true}}

	clusVisits := map[string]int{}
	require.NoError(t, structwalk.Walk(clus, func(path *structpath.PathNode, _ any, _ *reflect.StructField) {
		clusVisits[path.String()]++
	}))

	blob, _ = json.Marshal(clus)
	var clusJSON map[string]any
	require.NoError(t, json.Unmarshal(blob, &clusJSON))

	// encoding/json serializes ShadowCluster.Lifecycle (named, shallower) and
	// ignores the promoted ShadowBase.Lifecycle. structwalk visits both.
	// Bug: walk visits twice; json emits once. Record the buggy values.
	assert.Equal(t, 2, clusVisits["lifecycle.prevent_destroy"],
		"structwalk should visit 'lifecycle.prevent_destroy' twice (bug); fix → 1")
	assert.Equal(t, true, nestedGet(clusJSON, "lifecycle", "prevent_destroy"),
		"encoding/json must emit 'lifecycle.prevent_destroy' from the named field")
}

// TestShadowedEmbedCausesStructdiffDuplicate shows what happens downstream:
// when both shadowed fields are non-zero and differ, structdiff emits the same
// path twice — once per shadowed declaration. prepareChanges in the direct
// engine uses a map, so the second entry silently overwrites the first; which
// value wins is arbitrary.
func TestShadowedEmbedCausesStructdiffDuplicate(t *testing.T) {
	before := &ShadowPipeline{}
	before.ID = "old-base-id"
	before.Id = "old-sdk-id"

	after := &ShadowPipeline{}
	after.ID = "new-base-id"
	after.Id = "new-sdk-id"

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
		t.Logf("encoding/json would emit neither: they are ambiguous fields")
	}
	// Assert the current (buggy) behaviour so a fix requires updating this test.
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
