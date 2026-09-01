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

// knownStateDivergences and knownRemoteDivergences enumerate what disagrees today, so a
// new disagreement fails the test while these are worked through.
var (
	knownStateDivergences = map[string][]string{
		// Free-form any fields: structwalk documents that it does not traverse an interface,
		// so drift inside a serialized dashboard or a cluster policy definition is invisible
		// to it and to structdiff.
		"dashboards":       {"serialized_dashboard"},
		"genie_spaces":     {"serialized_space"},
		"cluster_policies": {"definition", "policy_family_definition_overrides"},
	}

	knownRemoteDivergences = map[string][]string{
		"dashboards":       {"serialized_dashboard"},
		"genie_spaces":     {"serialized_space"},
		"cluster_policies": {"definition", "policy_family_definition_overrides"},
	}
)

func TestStateTypeAgreesWithJSON(t *testing.T) {
	testAgreesWithJSON(t, (*Adapter).StateType, knownStateDivergences)
}

func TestRemoteTypeAgreesWithJSON(t *testing.T) {
	testAgreesWithJSON(t, (*Adapter).RemoteType, knownRemoteDivergences)
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

			report = report.Filter(known[resourceType])
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
