// Package direct is the ucm direct-deployment engine. It applies catalogs,
// schemas, and grants by calling the Databricks SDK directly — no terraform.
//
// The package is a ground-up implementation scoped to UCM's three-resource
// model. It deliberately does not port bundle/direct/** since cmd/ucm/CLAUDE.md
// forbids bundle imports from ucm/** and the UCM shape is simple enough
// (three concrete types, linear dependency order, no reference resolution)
// that a full Adapter/DAG framework would be overkill.
package direct

import (
	"reflect"
	"sort"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deployplan"
)

// CalculatePlan produces a deployplan.Plan for the given Ucm and state. The
// plan compares desired config against the recorded State, emitting:
//   - Create for keys present in config but absent from state
//   - Delete for keys present in state but absent from config
//   - Update for keys present in both when field values differ
//   - Skip  for keys present in both with identical field values
//
// Drift against remote UC objects is intentionally not detected here;
// double-checking remote state costs O(n) API calls per plan which is
// disproportionate for the M2 scope. The follow-up task is to layer a
// remote-read pass on top when we ship volumes/external_locations.
func CalculatePlan(u *ucm.Ucm, state *State) *deployplan.Plan {
	plan := deployplan.NewPlanTerraform()

	planStorageCredentials(u, state, plan)
	planExternalLocations(u, state, plan)
	planCatalogs(u, state, plan)
	planSchemas(u, state, plan)
	planVolumes(u, state, plan)
	planConnections(u, state, plan)
	planGrants(u, state, plan)

	return plan
}

func planStorageCredentials(u *ucm.Ucm, state *State, plan *deployplan.Plan) {
	desired := u.Config.Resources.StorageCredentials
	recorded := state.StorageCredentials

	keys := mergedKeys(desired, recorded)
	for _, key := range keys {
		planKey := "resources.storage_credentials." + key

		cfg, haveCfg := desired[key]
		rec, haveRec := recorded[key]
		switch {
		case haveCfg && !haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Create}
		case !haveCfg && haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Delete}
		case storageCredentialStateFromConfig(cfg).equal(rec):
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Skip}
		default:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Update}
		}
	}
}

func planExternalLocations(u *ucm.Ucm, state *State, plan *deployplan.Plan) {
	desired := u.Config.Resources.ExternalLocations
	recorded := state.ExternalLocations

	keys := mergedKeys(desired, recorded)
	for _, key := range keys {
		planKey := "resources.external_locations." + key

		cfg, haveCfg := desired[key]
		rec, haveRec := recorded[key]
		switch {
		case haveCfg && !haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Create}
		case !haveCfg && haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Delete}
		case externalLocationStateFromConfig(cfg).equal(rec):
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Skip}
		default:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Update}
		}
	}
}

func planCatalogs(u *ucm.Ucm, state *State, plan *deployplan.Plan) {
	desired := u.Config.Resources.Catalogs
	recorded := state.Catalogs

	keys := mergedKeys(desired, recorded)
	for _, key := range keys {
		planKey := "resources.catalogs." + key

		cfg, haveCfg := desired[key]
		rec, haveRec := recorded[key]
		switch {
		case haveCfg && !haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Create}
		case !haveCfg && haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Delete}
		case catalogStateFromConfig(cfg).equal(rec):
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Skip}
		default:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Update}
		}
	}
}

func planSchemas(u *ucm.Ucm, state *State, plan *deployplan.Plan) {
	desired := u.Config.Resources.Schemas
	recorded := state.Schemas

	keys := mergedKeys(desired, recorded)
	for _, key := range keys {
		planKey := "resources.schemas." + key

		cfg, haveCfg := desired[key]
		rec, haveRec := recorded[key]
		switch {
		case haveCfg && !haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Create}
		case !haveCfg && haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Delete}
		case schemaStateFromConfig(cfg).equal(rec):
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Skip}
		default:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Update}
		}
	}
}

func planVolumes(u *ucm.Ucm, state *State, plan *deployplan.Plan) {
	desired := u.Config.Resources.Volumes
	recorded := state.Volumes

	keys := mergedKeys(desired, recorded)
	for _, key := range keys {
		planKey := "resources.volumes." + key

		cfg, haveCfg := desired[key]
		rec, haveRec := recorded[key]
		switch {
		case haveCfg && !haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Create}
		case !haveCfg && haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Delete}
		case volumeStateFromConfig(cfg).equal(rec):
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Skip}
		default:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Update}
		}
	}
}

func planConnections(u *ucm.Ucm, state *State, plan *deployplan.Plan) {
	desired := u.Config.Resources.Connections
	recorded := state.Connections

	keys := mergedKeys(desired, recorded)
	for _, key := range keys {
		planKey := "resources.connections." + key

		cfg, haveCfg := desired[key]
		rec, haveRec := recorded[key]
		switch {
		case haveCfg && !haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Create}
		case !haveCfg && haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Delete}
		case connectionStateFromConfig(cfg).equal(rec):
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Skip}
		default:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Update}
		}
	}
}

func planGrants(u *ucm.Ucm, state *State, plan *deployplan.Plan) {
	desired := u.Config.Resources.Grants
	recorded := state.Grants

	keys := mergedKeys(desired, recorded)
	for _, key := range keys {
		planKey := "resources.grants." + key

		cfg, haveCfg := desired[key]
		rec, haveRec := recorded[key]
		switch {
		case haveCfg && !haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Create}
		case !haveCfg && haveRec:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Delete}
		case grantStateFromConfig(cfg).equal(rec):
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Skip}
		default:
			plan.Plan[planKey] = &deployplan.PlanEntry{Action: deployplan.Update}
		}
	}
}

// mergedKeys returns a sorted list of keys present in either map. Sorting
// keeps plan output stable across runs and matches the deployplan.GetActions
// discipline of emitting actions in lexical order.
func mergedKeys[V any, W any](a map[string]V, b map[string]W) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// equal is provided as methods on the state types so the planner stays
// declarative (one line per comparison) and the notion of equality for each
// resource type lives next to the state struct rather than scattered across
// the planner.
func (s CatalogState) equal(other *CatalogState) bool {
	return other != nil && reflect.DeepEqual(s, *other)
}

func (s SchemaState) equal(other *SchemaState) bool {
	return other != nil && reflect.DeepEqual(s, *other)
}

func (s GrantState) equal(other *GrantState) bool {
	if other == nil {
		return false
	}
	left := normalizedGrant(s)
	right := normalizedGrant(*other)
	return reflect.DeepEqual(left, right)
}

func (s StorageCredentialState) equal(other *StorageCredentialState) bool {
	return other != nil && reflect.DeepEqual(s, *other)
}

func (s ExternalLocationState) equal(other *ExternalLocationState) bool {
	return other != nil && reflect.DeepEqual(s, *other)
}

func (s VolumeState) equal(other *VolumeState) bool {
	return other != nil && reflect.DeepEqual(s, *other)
}

func (s ConnectionState) equal(other *ConnectionState) bool {
	return other != nil && reflect.DeepEqual(s, *other)
}

// normalizedGrant returns a copy of g with privileges sorted. The SDK sorts
// privileges server-side; normalizing before comparison avoids spurious
// Updates when the user reorders the config entries.
func normalizedGrant(g GrantState) GrantState {
	if len(g.Privileges) > 0 {
		privs := make([]string, len(g.Privileges))
		copy(privs, g.Privileges)
		sort.Strings(privs)
		g.Privileges = privs
	}
	return g
}
