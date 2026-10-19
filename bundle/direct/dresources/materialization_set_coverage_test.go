package dresources

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memberFieldsTheUpdateEndpointAccepts are the member fields UpdateMaterializationSet can write,
// so they need no rule to reach the server. Everything else about a member is write-once, and the
// endpoint ignores it rather than failing, which is why an unclassified field is silent.
var memberFieldsTheUpdateEndpointAccepts = map[string]bool{
	"pipeline_schedule_state": true,
}

// setFieldsWithoutRules are the set's own fields that no rule covers: the member slice, whose
// elements are diffed by key, and the id, which the generated config already classifies.
var setFieldsWithoutRules = map[string]bool{
	"materialized_features": true,
}

// classifiedFields collects every field pattern either lifecycle config mentions, from both the
// hand-written and generated halves, since the planner consults them in sequence.
func classifiedFields(t *testing.T) map[string]bool {
	t.Helper()

	handWritten := GetResourceConfig("materialization_sets")
	require.NotNil(t, handWritten)
	generated := GetGeneratedResourceConfig("materialization_sets")
	require.NotNil(t, generated)

	classified := map[string]bool{}
	for _, cfg := range []*ResourceLifecycleConfig{handWritten, generated} {
		for _, rules := range [][]FieldRule{
			cfg.RecreateOnChanges,
			cfg.IgnoreRemoteChanges,
			cfg.IgnoreLocalChanges,
			cfg.ProvidedIDFields,
			cfg.UpdatableIDFields,
			cfg.NormalizeSlash,
			cfg.StableOutputFields,
		} {
			for _, rule := range rules {
				classified[rule.Field.String()] = true
			}
		}
		for _, rule := range cfg.BackendDefaults {
			classified[rule.Field.String()] = true
		}
	}
	return classified
}

// memberFieldPath builds the path the planner hands to OverrideChangeDesc for one member field.
func memberFieldPath(t *testing.T, memberKey, field string) *structpath.PathNode {
	t.Helper()
	path, err := structpath.ParsePath(
		"materialized_features[member='" + memberKey + "']." + field)
	require.NoError(t, err)
	return path
}

// TestMaterializationSetAllMemberFieldsAreClassified keeps the hand-written recreate list
// exhaustive. TestNoUpdateResourcesCoverAllFields stops at HasDoUpdate, and this resource has one,
// so nothing else holds the line: a member field added to the proto later would be classified as a
// plain update, and the endpoint would accept the request and write nothing.
func TestMaterializationSetAllMemberFieldsAreClassified(t *testing.T) {
	classified := classifiedFields(t)

	for field := range reflect.TypeFor[ml.MaterializedFeature]().Fields() {
		name, ok := jsonFieldName(field)
		if !ok || memberFieldsTheUpdateEndpointAccepts[name] {
			continue
		}
		assert.Truef(t, classified["materialized_features[*]."+name],
			"member field %q is declared in neither materialization_sets.yml nor its generated "+
				"half; add it to recreate_on_changes if the endpoint cannot write it, or to "+
				"memberFieldsTheUpdateEndpointAccepts if it can",
			name,
		)
	}
}

// TestMaterializationSetAllSetFieldsAreClassified does the same for the set's own fields.
func TestMaterializationSetAllSetFieldsAreClassified(t *testing.T) {
	classified := classifiedFields(t)

	for field := range reflect.TypeFor[ml.MaterializationSet]().Fields() {
		name, ok := jsonFieldName(field)
		if !ok || setFieldsWithoutRules[name] {
			continue
		}
		assert.Truef(t, classified[name],
			"set field %q is declared in neither materialization_sets.yml nor its generated half",
			name,
		)
	}
}

// TestMaterializationSetDestinationFieldsAreSuppressed pins the fields OverrideChangeDesc skips.
// They are input-only, so the read never returns them and the generated rule that would absorb
// the difference cannot fire; a field dropped from this list silently re-plans as a recreate.
func TestMaterializationSetDestinationFieldsAreSuppressed(t *testing.T) {
	offline := &ml.MaterializedFeature{
		OfflineStoreConfig: &ml.OfflineStoreConfig{CatalogName: "c", SchemaName: "s", TableNamePrefix: "p"},
	}
	online := &ml.MaterializedFeature{
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "store", CatalogName: "c"},
	}

	for _, field := range []string{"catalog_name", "schema_name", "table_name_prefix"} {
		_, offlineKey := materializationSetMemberKey(*offline)
		_, onlineKey := materializationSetMemberKey(*online)
		assert.Truef(t,
			isRedactedDestinationField(memberFieldPath(t, offlineKey, "offline_store_config."+field)),
			"offline_store_config.%s must be suppressed", field)
		assert.Truef(t,
			isRedactedDestinationField(memberFieldPath(t, onlineKey, "online_store_config."+field)),
			"online_store_config.%s must be suppressed", field)
	}

	// online_store_name is returned by the read, so a repoint of it must stay visible.
	_, onlineKey := materializationSetMemberKey(*online)
	assert.False(t,
		isRedactedDestinationField(memberFieldPath(t, onlineKey, "online_store_config.online_store_name")))
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return "", false
	}
	return strings.TrimSuffix(tag, ",omitempty"), true
}
