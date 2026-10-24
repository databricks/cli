package resourcemutator_test

import (
	"maps"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serverOwnedMemberFields are the member fields no authoring type declares. All but the first are
// OUTPUT_ONLY in the spec, so the schema generator keeps them out of the bundle schema;
// materialized_feature_id is left out because the server assigns one per member and a single
// declared id cannot belong to all of them.
var serverOwnedMemberFields = map[string]bool{
	"MaterializedFeatureId":   true,
	"MaterializationSetId":    true,
	"IsOnline":                true,
	"JobId":                   true,
	"PipelineId":              true,
	"TableName":               true,
	"LastMaterializationTime": true,
	"LatestBackfillOperation": true,
}

// memberFieldsDerivedByConfigLayer are neither declared nor carried: the dyn-to-typed conversion
// rebuilds them per member from the fields that member holds. ForceSendFields comes back naming
// each expanded member's own pointer fields, so an explicitly set zero value survives the
// expansion without the mutator naming it.
var memberFieldsDerivedByConfigLayer = map[string]bool{
	"ForceSendFields": true,
}

// entryWithEveryFieldSet returns an authoring entry whose every field holds a non-zero value, so
// that a field reaching no expanded member is visible as a zero.
func entryWithEveryFieldSet[T any](t *testing.T) T {
	t.Helper()

	var entry T
	value := reflect.ValueOf(&entry).Elem()
	for field := range reflect.TypeFor[T]().Fields() {
		target := value.FieldByName(field.Name)
		switch field.Type.Kind() {
		case reflect.String:
			target.SetString("x")
		case reflect.Bool:
			target.SetBool(true)
		case reflect.Int, reflect.Int32, reflect.Int64:
			target.SetInt(1)
		case reflect.Pointer:
			target.Set(reflect.New(field.Type.Elem()))
		case reflect.Map:
			pair := reflect.MakeMap(field.Type)
			pair.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf("v"))
			target.Set(pair)
		case reflect.Slice:
			target.Set(reflect.MakeSlice(field.Type, 1, 1))
		default:
			require.Failf(t, "unhandled field kind",
				"field %s has kind %s, which this test does not know how to populate",
				field.Name, field.Type.Kind())
		}
	}
	return entry
}

func notCarried(extra ...string) map[string]bool {
	out := maps.Clone(serverOwnedMemberFields)
	for _, name := range extra {
		out[name] = true
	}
	return out
}

// TestExpandMaterializationSetMaterializedFeatureMembersCoversEveryMemberField keeps each
// expansion's field list exhaustive and each authoring type complete against the API. Members are
// built by naming their fields, and the authoring types are hand-written, so a member field added
// to the proto later reaches no expanded member and nothing else notices.
func TestExpandMaterializationSetMaterializedFeatureMembersCoversEveryMemberField(t *testing.T) {
	for name, tc := range map[string]struct {
		set        *resources.MaterializationSet
		notCarried map[string]bool
	}{
		"sawtooth_materialized_features": {
			set: &resources.MaterializationSet{
				SawtoothMaterializedFeatures: []resources.SawtoothMaterializedFeatures{
					entryWithEveryFieldSet[resources.SawtoothMaterializedFeatures](t),
				},
			},
			notCarried: notCarried(),
		},
		"batch_offline_and_online_materialized_features": {
			set: &resources.MaterializationSet{
				BatchOfflineAndOnlineMaterializedFeatures: []resources.BatchOfflineAndOnlineMaterializedFeatures{
					entryWithEveryFieldSet[resources.BatchOfflineAndOnlineMaterializedFeatures](t),
				},
			},
			// A feature on the batch path has no streaming mode, so the type declares none.
			notCarried: notCarried("StreamingMode"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			expanded := expandSet(t, tc.set)
			require.NotEmpty(t, expanded.MaterializedFeatures)

			for field := range reflect.TypeFor[ml.MaterializedFeature]().Fields() {
				if memberFieldsDerivedByConfigLayer[field.Name] {
					continue
				}

				carried := false
				for _, member := range expanded.MaterializedFeatures {
					if !reflect.ValueOf(member).FieldByName(field.Name).IsZero() {
						carried = true
						break
					}
				}

				if tc.notCarried[field.Name] {
					assert.Falsef(t, carried,
						"field %s is listed as not carried but reached an expanded member; drop it "+
							"from that list or stop carrying it", field.Name)
					continue
				}
				assert.Truef(t, carried,
					"field %s reached no member of the %s expansion. Declare it on the authoring "+
						"type in bundle/config/resources/materialization_set.go and carry it in "+
						"expand_materialization_set_materialized_feature_members.go, or list it as "+
						"not carried with the reason it is server-owned",
					field.Name, name)
			}
		})
	}
}
