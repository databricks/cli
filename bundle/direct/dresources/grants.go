package dresources

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/structs/registry"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registry.Register[catalog.PrivilegeAssignment]("principal")
}

var grantResourceToSecurableType = map[string]string{
	"catalogs":                "catalog",
	"schemas":                 "schema",
	"external_locations":      "external_location",
	"volumes":                 "volume",
	"registered_models":       "function",
	"secrets":                 "secret",
	"vector_search_indexes":   "table",
	"model_services":          "model_service",
	"mcp_services":            "mcp_service",
	"model_provider_services": "model_provider_service",
}

type GrantsState struct {
	SecurableType string                        `json:"securable_type"`
	FullName      string                        `json:"full_name"`
	EmbeddedSlice []catalog.PrivilegeAssignment `json:"__embed__,omitempty"`
}

type ResourceGrants struct {
	client *databricks.WorkspaceClient

	// securableType is the UC securable type of the parent resource, e.g. "schema".
	securableType string
}

func (*ResourceGrants) New(client *databricks.WorkspaceClient) *ResourceGrants {
	return &ResourceGrants{client: client, securableType: ""}
}

func (r *ResourceGrants) Configure(resourceType string) error {
	parentType, ok := strings.CutSuffix(resourceType, ".grants")
	if !ok {
		return fmt.Errorf("internal error: resource type %q does not end with .grants", resourceType)
	}

	r.securableType, ok = grantResourceToSecurableType[parentType]
	if !ok {
		return fmt.Errorf("unsupported grants resource type: %s", parentType)
	}

	return nil
}

func (r *ResourceGrants) PrepareInputConfig(inputConfig *[]catalog.PrivilegeAssignment, resourceKey string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(resourceKey, ".grants")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .grants", resourceKey)
	}

	// Normalize the same way as DoRead (sort, collapse ALL_PRIVILEGES) so the
	// config and the value read back compare equal.
	normalizeAssignments(*inputConfig)

	return &structvar.StructVar{
		Value: &GrantsState{
			SecurableType: r.securableType,
			FullName:      "",
			EmbeddedSlice: *inputConfig,
		},
		Refs: map[string]string{
			"full_name": "${" + baseNode + ".id}",
		},
	}, nil
}

func (*ResourceGrants) PrepareState(state *GrantsState) *GrantsState {
	return state
}

// IsEmptyState reports an empty grants list as no resource at all: nothing to grant, and
// Terraform records no databricks_grants resource for it either, so migrated bundles have
// no state entry.
func (*ResourceGrants) IsEmptyState(state *GrantsState) bool {
	return len(state.EmbeddedSlice) == 0
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
		Changes:                   buildGrantChanges(state.EmbeddedSlice, remoteGrantPrivileges(entry)),
		OmitPermissionsInResponse: false,
		ForceSendFields:           nil,
	})
	return nil, err
}

// ResourceGrants intentionally implements no DoDelete: removing grants from the
// bundle does nothing to the backend. We could revoke all grants here, but it would
// be confusing to explain wrt permissions. Deleting the resource is a state-only
// cleanup (see PlanEntry.StateOnly).

func buildGrantChanges(desiredAssignments []catalog.PrivilegeAssignment, remote map[string][]catalog.Privilege) []catalog.PermissionsChange {
	changes := make([]catalog.PermissionsChange, 0, len(desiredAssignments)+len(remote))
	desiredPrincipals := make(map[string]struct{}, len(desiredAssignments))
	for _, ga := range desiredAssignments {
		desiredPrincipals[ga.Principal] = struct{}{}
		changes = append(changes, catalog.PermissionsChange{
			Principal:       ga.Principal,
			Add:             ga.Privileges,
			Remove:          grantRemovals(ga.Privileges, remote[ga.Principal]),
			ForceSendFields: nil,
		})
	}

	// Principals present remotely but no longer desired: revoke everything.
	var removedPrincipals []string
	for principal := range remote {
		if _, ok := desiredPrincipals[principal]; !ok {
			removedPrincipals = append(removedPrincipals, principal)
		}
	}
	slices.Sort(removedPrincipals)
	for _, principal := range removedPrincipals {
		changes = append(changes, catalog.PermissionsChange{
			Principal:       principal,
			Add:             nil,
			Remove:          grantRemovals(nil, remote[principal]),
			ForceSendFields: nil,
		})
	}
	return changes
}

// grantRemovals returns the privileges to revoke for a principal so the backend
// converges to desired. When ALL_PRIVILEGES stays desired it must not appear in
// Remove (the backend rejects the same privilege in Add and Remove), so only the
// no-longer-wanted excluded privileges are revoked. Otherwise ALL_PRIVILEGES is
// removed to clear everything it implies, plus the excluded privileges by name
// because ALL_PRIVILEGES does not imply them and its removal would leave them behind.
func grantRemovals(desired, remotePrivileges []catalog.Privilege) []catalog.Privilege {
	remove := revokedExcludedPrivileges(desired, remotePrivileges)
	if slices.Contains(desired, catalog.PrivilegeAllPrivileges) {
		return remove
	}
	return append([]catalog.Privilege{catalog.PrivilegeAllPrivileges}, remove...)
}

// revokedExcludedPrivileges returns the privileges not implied by ALL_PRIVILEGES
// (see allPrivilegesExcludes) that the principal holds remotely but no longer
// desires, so they can be revoked by name.
func revokedExcludedPrivileges(desired, remotePrivileges []catalog.Privilege) []catalog.Privilege {
	var remove []catalog.Privilege
	for _, p := range allPrivilegesExcludes {
		if slices.Contains(remotePrivileges, p) && !slices.Contains(desired, p) {
			remove = append(remove, p)
		}
	}
	return remove
}

// remoteGrantPrivileges maps each principal in the plan's remote state to the
// privileges it currently holds, or nil when there is no remote state (e.g. on
// create). The privileges are normalized the same way as the desired assignments.
func remoteGrantPrivileges(entry *PlanEntry) map[string][]catalog.Privilege {
	if entry == nil {
		return nil
	}
	remote, ok := entry.RemoteState.(*GrantsState)
	if !ok || remote == nil {
		return nil
	}
	result := make(map[string][]catalog.Privilege, len(remote.EmbeddedSlice))
	for _, a := range remote.EmbeddedSlice {
		if a.Principal != "" {
			result[a.Principal] = a.Privileges
		}
	}
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
	// Normalize the same way as the config side (sort, collapse ALL_PRIVILEGES)
	// so the two compare equal and we don't report false drift.
	normalizeAssignments(assignments)
	return assignments, nil
}

// allPrivilegesExcludes are the privileges that ALL_PRIVILEGES does not imply, so
// they are granted independently and must survive the collapse in
// normalizeAssignments. UC excludes exactly these four from ALL_PRIVILEGES to
// avoid accidental data exfiltration or privilege escalation; see
// https://docs.databricks.com/aws/en/data-governance/unity-catalog/manage-privileges/privileges
var allPrivilegesExcludes = []catalog.Privilege{
	catalog.PrivilegeExternalUseLocation,
	catalog.PrivilegeExternalUseSchema,
	catalog.PrivilegeManage,
	catalog.PrivilegeReadMetadata,
}

// normalizeAssignments sorts each assignment's privileges (the backend sorts
// them, so we match that) and collapses a principal holding ALL_PRIVILEGES down
// to ALL_PRIVILEGES plus any privilege ALL_PRIVILEGES does not imply. The collapse
// is applied to both the config and read sides, so config granting ALL_PRIVILEGES
// matches a backend that reports ALL_PRIVILEGES plus the concrete privileges it
// implies, instead of reporting a perpetual update. The excluded privileges are
// kept so they are not silently dropped from both the request and the drift check.
func normalizeAssignments(assignments []catalog.PrivilegeAssignment) {
	for i := range assignments {
		privileges := assignments[i].Privileges
		if slices.Contains(privileges, catalog.PrivilegeAllPrivileges) {
			collapsed := []catalog.Privilege{catalog.PrivilegeAllPrivileges}
			for _, p := range allPrivilegesExcludes {
				if slices.Contains(privileges, p) {
					collapsed = append(collapsed, p)
				}
			}
			slices.Sort(collapsed)
			assignments[i].Privileges = collapsed
			continue
		}
		slices.Sort(privileges)
	}
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
