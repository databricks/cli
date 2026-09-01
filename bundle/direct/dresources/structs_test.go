package dresources

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/structstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StateType and RemoteType are the types the direct engine actually reads fields out of:
// the plan resolves a ${resources...} reference by walking them, the state file is the
// JSON encoding of StateType, and RemoteType is what a refresh decodes into. So a
// disagreement between them and encoding/json is not latent the way it is for the config
// types -- it is a field the plan cannot see or the state file cannot carry.
//
// assertJSONRoundTrip in serialize_test.go covers the neighbouring question, whether a
// wrapper loses fields across Marshal -> Unmarshal. This covers whether the libs/structs
// packages and encoding/json name and reach the same fields in the first place.

// knownDivergences is where a per-path disagreement would be recorded. It is empty: the state
// and remote types agree with encoding/json on every path today, and the two limitations that
// remain -- free-form any fields and types that marshal themselves as a scalar -- are reported
// as categories rather than paths. A new disagreement fails the test rather than landing here
// silently.
var knownDivergences = map[string][]string{}

// freeFormFields lists the any-typed fields of each resource's state and remote types. Unlike
// the config types, cluster policies have none: the state type carries definition as the string
// the API takes, which ConfigureClusterPolicyDefinition has already normalized.
var freeFormFields = map[string][]string{
	"dashboards":   {"serialized_dashboard"},
	"genie_spaces": {"serialized_space"},
}

// topLevelNames reduces reported paths to the distinct top-level field each sits under, so the
// expectation does not depend on the filler's choice of map key.
func topLevelNames(paths []string) []string {
	var out []string
	for _, path := range paths {
		name, _, _ := strings.Cut(strings.SplitN(path, ":", 2)[0], ".")
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

func TestStateTypeAgreesWithJSON(t *testing.T) {
	testAgreesWithJSON(t, (*Adapter).StateType, knownDivergences)
}

func TestRemoteTypeAgreesWithJSON(t *testing.T) {
	testAgreesWithJSON(t, (*Adapter).RemoteType, knownDivergences)
}

// testAgreesWithJSON runs the check for every registered resource, so a newly supported
// resource type is covered without touching this test.
func testAgreesWithJSON(t *testing.T, typeOf func(*Adapter) reflect.Type, known map[string][]string) {
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		t.Run(resourceType, func(t *testing.T) {
			typ := typeOf(adapter)
			report, err := structstest.Check(typ)
			require.NoError(t, err)

			report, stale := report.Filter(known[resourceType])
			require.Empty(t, stale,
				"these recorded divergences no longer occur -- remove them from the list: %v", stale)
			// Free-form any fields are opaque to the packages; which resources have one is stable,
			// so it is ratcheted by name rather than logged away.
			assert.ElementsMatch(t, freeFormFields[resourceType], topLevelNames(report.InsideFreeFormField),
				"free-form any fields changed for %s", resourceType)
			report.InsideFreeFormField = nil
			if len(report.SelfMarshalingScalars) > 0 {
				// A known structwalk limitation. The ratchet is on the *types* that behave this
				// way, not the paths: a new field of a type already known to hide itself tells us
				// nothing, while a new such type is a finding.
				assert.Subset(t, structstest.KnownSelfMarshalingTypes, report.SelfMarshalingTypes,
					"a Go type that marshals itself as a scalar and so is invisible to structwalk")
				t.Logf("%d self-marshaling scalar field(s): %v",
					len(report.SelfMarshalingScalars), report.SelfMarshalingScalars)
				report.SelfMarshalingScalars = nil
				report.SelfMarshalingTypes = nil
			}
			require.True(t, report.Empty(),
				"%s (%s) disagrees with encoding/json:%s", resourceType, typ, report)
		})
	}
}
