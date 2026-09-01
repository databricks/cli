package structstest_test

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/structstest"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/stretchr/testify/require"
)

// knownDivergences lists the disagreements the bundle's resource types have with
// encoding/json today. Each entry is a bug somewhere other than this test; the test
// enumerates them so that a *new* disagreement fails while these are worked through.
//
// Nothing in the CLI marshals a resource config type with encoding/json today -- bundle
// validate -o json marshals the dyn tree -- so none of these is user-visible yet. They
// are one json.Marshal away from being so.
var knownDivergences = map[string][]string{
	// The resource type embeds another struct that declares MarshalJSON and does not
	// declare its own, so the embedded marshaler takes over and every field the outer
	// struct adds -- id, url, lifecycle, permissions -- never reaches the wire. Fixed by
	// giving the resource type the marshaler pair resources.Job has.
	"dashboards":             baseResourceFields("file_path", "permissions"),
	"genie_spaces":           baseResourceFields("file_path", "permissions"),
	"database_instances":     baseResourceFields("permissions"),
	"database_catalogs":      baseResourceFields(),
	"synced_database_tables": baseResourceFields(),
	"postgres_projects":      baseResourceFields("permissions"),
	"postgres_branches":      baseResourceFields(),
	"postgres_endpoints":     baseResourceFields(),
	"postgres_catalogs":      baseResourceFields(),
	"postgres_databases":     baseResourceFields(),
	"postgres_roles":         baseResourceFields(),
	"postgres_synced_tables": baseResourceFields(),
}

// interfaceFieldPaths are free-form any fields. structwalk documents that it does not
// traverse an interface, so these never reach a visit callback and structdiff never
// reports a change to one. Intentional, but it means drift in a serialized dashboard or a
// cluster policy definition is invisible to the packages.
var interfaceFieldPaths = map[string][]string{
	"dashboards":       {"serialized_dashboard"},
	"genie_spaces":     {"serialized_space"},
	"cluster_policies": {"definition", "policy_family_definition_overrides"},
}

// baseResourceFields returns the paths a resource gains from BaseResource, plus any extra
// fields the resource declares alongside it. They are lost together, by one cause.
func baseResourceFields(extra ...string) []string {
	return append([]string{"id", "url", "modified_status", "lifecycle.prevent_destroy"}, extra...)
}

// TestResourceTypesAgreeWithJSON feeds every resource type in config.Resources through
// structstest.Check. Driving it off the struct by reflection means a newly added resource
// is covered without touching this test.
func TestResourceTypesAgreeWithJSON(t *testing.T) {
	rt := reflect.TypeOf(config.Resources{})

	var checked int
	for i := range rt.NumField() {
		field := rt.Field(i)
		if field.Type.Kind() != reflect.Map {
			continue
		}
		elem := field.Type.Elem()
		if elem.Kind() != reflect.Pointer || elem.Elem().Kind() != reflect.Struct {
			continue
		}
		group := structtag.JSONTag(field.Tag.Get("json")).Name()

		t.Run(group, func(t *testing.T) {
			report, err := structstest.Check(elem)
			require.NoError(t, err)

			var known []string
			known = append(known, knownDivergences[group]...)
			known = append(known, interfaceFieldPaths[group]...)
			report = report.Filter(known)
			if len(report.SelfMarshalingScalars) > 0 {
				// A known structwalk limitation, tracked as one item rather than one entry per
				// timestamp field, because every new SDK time field joins it.
				t.Logf("%d self-marshaling scalar field(s) structwalk does not visit: %v",
					len(report.SelfMarshalingScalars), report.SelfMarshalingScalars)
				report.SelfMarshalingScalars = nil
			}
			require.True(t, report.Empty(),
				"%s (%s) disagrees with encoding/json:%s", group, elem, report)
		})
		checked++
	}

	// A guard against the loop silently matching nothing, which would make the whole
	// test vacuous.
	require.Greater(t, checked, 20, "expected every resource group to be checked")
}
