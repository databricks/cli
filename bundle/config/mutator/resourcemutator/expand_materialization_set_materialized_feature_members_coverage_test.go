package resourcemutator_test

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memberFieldsNotCarried are the member fields a shorthand entry never contributes to the members
// it expands into. All but the first are OUTPUT_ONLY in the spec, so the schema generator already
// keeps them out of the bundle schema and no config can set them.
//
// materialized_feature_id is the one field a shorthand entry can set and have ignored: the server
// assigns one per member, so a single declared id cannot belong to all of them.
var memberFieldsNotCarried = map[string]bool{
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

// memberWithEveryFieldSet returns a materialized feature whose every field holds a non-zero value,
// so that a field missing from every expanded member is visible as a zero.
func memberWithEveryFieldSet(t *testing.T) ml.MaterializedFeature {
	t.Helper()

	var member ml.MaterializedFeature
	value := reflect.ValueOf(&member).Elem()
	for field := range reflect.TypeFor[ml.MaterializedFeature]().Fields() {
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
			entry := reflect.MakeMap(field.Type)
			entry.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf("v"))
			target.Set(entry)
		case reflect.Slice:
			target.Set(reflect.MakeSlice(field.Type, 1, 1))
		default:
			require.Failf(t, "unhandled field kind",
				"field %s has kind %s, which this test does not know how to populate",
				field.Name, field.Type.Kind())
		}
	}
	return member
}

// TestExpandMaterializationSetMaterializedFeatureMembersCoversEveryMemberField keeps each expansion's field list
// exhaustive. Members are built by naming their fields rather than by copying the entry, so a
// member field added to the proto later reaches no expanded member and nothing else notices.
func TestExpandMaterializationSetMaterializedFeatureMembersCoversEveryMemberField(t *testing.T) {
	entry := memberWithEveryFieldSet(t)

	for name, set := range map[string]*resources.MaterializationSet{
		"sawtooth_materialized_features": {
			SawtoothMaterializedFeatures: []ml.MaterializedFeature{entry},
		},
		"batch_offline_and_online_materialized_features": {
			BatchOfflineAndOnlineMaterializedFeatures: []ml.MaterializedFeature{entry},
		},
	} {
		t.Run(name, func(t *testing.T) {
			expanded := expandSet(t, set)
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

				if memberFieldsNotCarried[field.Name] {
					assert.Falsef(t, carried,
						"field %s is listed in memberFieldsNotCarried but reached an expanded "+
							"member; drop it from that list or stop carrying it", field.Name)
					continue
				}
				assert.Truef(t, carried,
					"field %s reached no member of the %s expansion. Add it to the member "+
						"literals in expand_materialization_set_materialized_feature_members.go, or to "+
						"memberFieldsNotCarried with the reason it is server-owned",
					field.Name, name)
			}
		})
	}
}
