package structstest_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/structstest"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/stretchr/testify/assert"
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

// walkDuplicates lists the paths structwalk visits twice for a resource, because the resource
// embeds BaseResource alongside an SDK type that declares the same json name, or two structs
// that each carry a Lifecycle. encoding/json serializes the shallower one and nothing else, so
// the second visit is a field that cannot reach the wire under that name -- and structdiff
// reports a change at the path twice. Ratcheted by name: fixing structwalk to resolve a name
// the way encoding/json does empties these, and a new shadowed field has to be added here.
var walkDuplicates = map[string][]string{
	"job_runs":       {"lifecycle.prevent_destroy"},
	"pipelines":      {"id"},
	"clusters":       {"lifecycle.prevent_destroy"},
	"apps":           {"id", "url", "lifecycle.prevent_destroy"},
	"alerts":         {"id"},
	"sql_warehouses": {"lifecycle.prevent_destroy"},
}

// freeFormFields lists the any-typed fields of each resource. Everything at or below one is
// invisible to the packages.
var freeFormFields = map[string][]string{
	"dashboards":       {"serialized_dashboard"},
	"genie_spaces":     {"serialized_space"},
	"cluster_policies": {"definition", "policy_family_definition_overrides"},
}

// freeFormFieldNames reduces the reported paths to the distinct top-level field each sits under,
// so the expectation does not depend on the filler's choice of map key.
func freeFormFieldNames(paths []string) []string {
	var out []string
	for _, path := range paths {
		name, _, _ := strings.Cut(strings.SplitN(path, ":", 2)[0], ".")
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
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
			// A known limitation: structwalk does not traverse an interface and structaccess cannot
			// validate a path through one, so a free-form field is opaque to both. Which resources
			// have one is stable, so it is ratcheted by name: a new free-form field is a new blind
			// spot and has to be added here deliberately.
			assert.ElementsMatch(t, walkDuplicates[group], report.WalkDuplicated,
				"paths structwalk visits twice changed for %s", group)
			report.WalkDuplicated = nil

			assert.ElementsMatch(t, freeFormFields[group], freeFormFieldNames(report.InsideFreeFormField),
				"free-form any fields changed for %s", group)
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
				"%s (%s) disagrees with encoding/json:%s", group, elem, report)
		})
		checked++
	}

	// A guard against the loop silently matching nothing, which would make the whole
	// test vacuous.
	require.Greater(t, checked, 20, "expected every resource group to be checked")
}
