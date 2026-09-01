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

// baseResourceFields returns the paths a resource gains from BaseResource, plus any extra
// fields the resource declares alongside it. They are lost together, by one cause.
func baseResourceFields(extra ...string) []string {
	return append([]string{"id", "url", "modified_status", "lifecycle.prevent_destroy"}, extra...)
}

// TestResourceTypesAgreeWithJSON feeds every resource type in config.Resources through
// structstest.Check. Driving it off the struct by reflection means a newly added resource
// is covered without touching this test.
func TestResourceTypesAgreeWithJSON(t *testing.T) {
	rt := reflect.TypeFor[config.Resources]()

	var checked int
	for field := range rt.Fields() {
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
			report, stale := report.Filter(known)
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
				"%s (%s) disagrees with encoding/json:%s", group, elem, report)
		})
		checked++
	}

	// A guard against the loop silently matching nothing, which would make the whole
	// test vacuous.
	require.Greater(t, checked, 20, "expected every resource group to be checked")
}
