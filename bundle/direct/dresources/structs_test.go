package dresources

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/structstest"
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
			if len(report.InsideFreeFormField) > 0 {
				// A known limitation: structwalk does not traverse an interface and structaccess
				// cannot validate a path through one, so a free-form field is opaque to both.
				t.Logf("%d path(s) inside a free-form any field: %v",
					len(report.InsideFreeFormField), report.InsideFreeFormField)
				report.InsideFreeFormField = nil
			}
			if len(report.SelfMarshalingScalars) > 0 {
				// A known structwalk limitation, tracked as one item rather than one entry per
				// timestamp field, because every new SDK time field joins it.
				t.Logf("%d self-marshaling scalar field(s) structwalk does not visit: %v",
					len(report.SelfMarshalingScalars), report.SelfMarshalingScalars)
				report.SelfMarshalingScalars = nil
			}
			require.True(t, report.Empty(),
				"%s (%s) disagrees with encoding/json:%s", resourceType, typ, report)
		})
	}
}
