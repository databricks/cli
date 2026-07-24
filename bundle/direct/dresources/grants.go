package dresources

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

var grantResourceToSecurableType = map[string]string{
	"catalogs":              "catalog",
	"schemas":               "schema",
	"external_locations":    "external_location",
	"volumes":               "volume",
	"registered_models":     "function",
	"vector_search_indexes": "table",
}

type GrantsState struct {
	SecurableType string                        `json:"securable_type"`
	FullName      string                        `json:"full_name"`
	EmbeddedSlice []catalog.PrivilegeAssignment `json:"__embed__,omitempty"`
}

func PrepareGrantsInputConfig(inputConfig any, node string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(node, ".grants")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .grants", node)
	}

	resourceType, err := extractGrantResourceType(node)
	if err != nil {
		return nil, err
	}

	securableType, ok := grantResourceToSecurableType[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported grants resource type: %s", resourceType)
	}

	grantsPtr, ok := inputConfig.(*[]catalog.PrivilegeAssignment)
	if !ok {
		return nil, fmt.Errorf("expected *[]catalog.PrivilegeAssignment, got %T", inputConfig)
	}

	// Normalize the same way the backend does (uppercase privileges, sort) so
	// the config and the value read back by DoRead compare equal.
	normalizeAssignments(*grantsPtr)

	return &structvar.StructVar{
		Value: &GrantsState{
			SecurableType: securableType,
			FullName:      "",
			EmbeddedSlice: *grantsPtr,
		},
		Refs: map[string]string{
			"full_name": "${" + baseNode + ".id}",
		},
	}, nil
}

type ResourceGrants struct {
	client *databricks.WorkspaceClient
}

func (*ResourceGrants) New(client *databricks.WorkspaceClient) *ResourceGrants {
	return &ResourceGrants{client: client}
}

func (*ResourceGrants) PrepareState(state *GrantsState) *GrantsState {
	return state
}

func grantKey(x catalog.PrivilegeAssignment) (string, string) {
	return "principal", x.Principal
}

func (*ResourceGrants) KeyedSlices() map[string]any {
	// Empty key because EmbeddedSlice appears at the root path of
	// GrantsState (no "grants" prefix in struct walker paths).
	return map[string]any{
		"": grantKey,
	}
}

func (r *ResourceGrants) DoRead(ctx context.Context, id string) (*GrantsState, error) {
	securableType, fullName, err := parseGrantsID(id)
	if err != nil {
		return nil, err
	}

	assignments, err := r.listGrants(ctx, securableType, fullName)
	if err != nil {
		return nil, err
	}

	return &GrantsState{
		SecurableType: securableType,
		FullName:      fullName,
		EmbeddedSlice: assignments,
	}, nil
}

func (r *ResourceGrants) DoCreate(ctx context.Context, state *GrantsState) (string, *GrantsState, error) {
	_, err := r.DoUpdate(ctx, "", state, nil)
	if err != nil {
		// Grants Update is idempotent (additive PATCH), so retrying on transient errors is safe.
		return "", nil, retrySafe(err)
	}

	return state.SecurableType + "/" + state.FullName, nil, nil
}

func (r *ResourceGrants) DoUpdate(ctx context.Context, _ string, state *GrantsState, entry *PlanEntry) (*GrantsState, error) {
	if state.FullName == "" {
		return nil, errors.New("internal error: grants full_name must be resolved before deployment")
	}
	_, err := r.client.Grants.Update(ctx, catalog.UpdatePermissions{
		SecurableType:             state.SecurableType,
		FullName:                  state.FullName,
		Changes:                   buildGrantChanges(state.EmbeddedSlice, remoteAssignments(entry)),
		OmitPermissionsInResponse: false,
		ForceSendFields:           nil,
	})
	return nil, err
}

func (r *ResourceGrants) DoDelete(ctx context.Context, id string, _ *GrantsState) error {
	// Similar to permissions, we do nothing there.
	// We could delete all grants there, but it would be confusing to explain wrt permissions.
	return nil
}

// buildGrantChanges computes the per-principal add/remove diff between the
// desired assignments and the remote assignments, mirroring the Terraform
// provider's diffPermissions. For each principal Add is (desired - remote) and
// Remove is (remote - desired). Computing an exact diff (rather than sending a
// blanket "remove ALL_PRIVILEGES" to wipe the principal) is what lets a
// principal granted ALL_PRIVILEGES converge when the backend also reports
// concrete privileges for it: those extra privileges land in Remove instead of
// being left in place forever. A privilege can never be in both Add and Remove,
// so this also avoids the "Duplicate privileges to add and delete" API error.
func buildGrantChanges(desiredAssignments, remoteAssignments []catalog.PrivilegeAssignment) []catalog.PermissionsChange {
	desired := privilegesByPrincipal(desiredAssignments)
	remote := privilegesByPrincipal(remoteAssignments)

	principals := make([]string, 0, len(desired)+len(remote))
	for p := range desired {
		principals = append(principals, p)
	}
	for p := range remote {
		if _, ok := desired[p]; !ok {
			principals = append(principals, p)
		}
	}
	slices.Sort(principals)

	changes := make([]catalog.PermissionsChange, 0, len(principals))
	for _, principal := range principals {
		add := setDifference(desired[principal], remote[principal])
		remove := setDifference(remote[principal], desired[principal])
		if len(add) == 0 && len(remove) == 0 {
			continue
		}
		changes = append(changes, catalog.PermissionsChange{
			Principal:       principal,
			Add:             add,
			Remove:          remove,
			ForceSendFields: nil,
		})
	}
	return changes
}

// remoteAssignments extracts the remote privilege assignments from the plan
// entry. It returns nil when the entry or its remote state is absent, which
// happens on create and when deploying from a serialized plan (the JSON-loaded
// state is not a *GrantsState). In those cases nothing is revoked.
func remoteAssignments(entry *PlanEntry) []catalog.PrivilegeAssignment {
	if entry == nil {
		return nil
	}
	remote, ok := entry.RemoteState.(*GrantsState)
	if !ok || remote == nil {
		return nil
	}
	return remote.EmbeddedSlice
}

func privilegesByPrincipal(assignments []catalog.PrivilegeAssignment) map[string]map[catalog.Privilege]struct{} {
	result := make(map[string]map[catalog.Privilege]struct{}, len(assignments))
	for _, a := range assignments {
		if a.Principal == "" {
			continue
		}
		privs := result[a.Principal]
		if privs == nil {
			privs = make(map[catalog.Privilege]struct{}, len(a.Privileges))
			result[a.Principal] = privs
		}
		for _, p := range a.Privileges {
			privs[p] = struct{}{}
		}
	}
	return result
}

// setDifference returns the sorted privileges present in a but not in b.
func setDifference(a, b map[catalog.Privilege]struct{}) []catalog.Privilege {
	var result []catalog.Privilege
	for p := range a {
		if _, ok := b[p]; !ok {
			result = append(result, p)
		}
	}
	slices.Sort(result)
	return result
}

func (r *ResourceGrants) listGrants(ctx context.Context, securableType, fullName string) ([]catalog.PrivilegeAssignment, error) {
	var assignments []catalog.PrivilegeAssignment
	pageToken := ""
	for {
		resp, err := r.client.Grants.Get(ctx, catalog.GetGrantRequest{
			FullName:        fullName,
			MaxResults:      0,
			PageToken:       pageToken,
			Principal:       "",
			SecurableType:   securableType,
			ForceSendFields: nil,
		})
		if err != nil {
			return nil, err
		}
		for _, assignment := range resp.PrivilegeAssignments {
			if assignment.Principal == "" {
				continue
			}
			assignments = append(assignments, catalog.PrivilegeAssignment{
				Principal:       assignment.Principal,
				Privileges:      assignment.Privileges,
				ForceSendFields: nil,
			})
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	// The backend does not guarantee a stable privilege order across reads, so
	// normalize the same way as the config side to avoid false drift.
	normalizeAssignments(assignments)
	return assignments, nil
}

// normalizePrivilege matches the backend's normalization: privileges are
// uppercased and spaces are converted to underscores (e.g. "use schema" ->
// "USE_SCHEMA"). Mirrors the Terraform provider's permissions.NormalizePrivilege.
func normalizePrivilege(p catalog.Privilege) catalog.Privilege {
	return catalog.Privilege(strings.ToUpper(strings.ReplaceAll(string(p), " ", "_")))
}

// normalizeAssignments normalizes and sorts each assignment's privileges in
// place so that config and remote state compare equal regardless of the
// case or order the backend returns.
func normalizeAssignments(assignments []catalog.PrivilegeAssignment) {
	for i := range assignments {
		for j := range assignments[i].Privileges {
			assignments[i].Privileges[j] = normalizePrivilege(assignments[i].Privileges[j])
		}
		slices.Sort(assignments[i].Privileges)
	}
}

func extractGrantResourceType(node string) (string, error) {
	rest, ok := strings.CutPrefix(node, "resources.")
	if !ok {
		return "", fmt.Errorf("cannot extract resource type from %q", node)
	}
	parts := strings.Split(rest, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("cannot extract resource type from %q", node)
	}
	return parts[0], nil
}

func parseGrantsID(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid grants id: %q", id)
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid grants id: %q", id)
	}
	return parts[0], parts[1], nil
}
