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
	"catalogs":           "catalog",
	"schemas":            "schema",
	"external_locations": "external_location",
	"volumes":            "volume",
	"registered_models":  "function",
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

	// Backend sorts privileges, so we sort here as well.
	for i := range *grantsPtr {
		sortPriviliges((*grantsPtr)[i].Privileges)
	}

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
	err := r.applyGrants(ctx, state)
	if err != nil {
		return "", nil, err
	}

	return makeGrantsID(state.SecurableType, state.FullName), nil, nil
}

func (r *ResourceGrants) DoUpdate(ctx context.Context, _ string, state *GrantsState, _ Changes) (*GrantsState, error) {
	return nil, r.applyGrants(ctx, state)
}

func (r *ResourceGrants) DoDelete(ctx context.Context, id string) error {
	// Similar to permissions, we do nothing there.
	// We could delete all grants there, but it would be confusing to explain wrt permissions.
	return nil
}

func (r *ResourceGrants) applyGrants(ctx context.Context, state *GrantsState) error {
	if state.FullName == "" {
		return errors.New("internal error: grants full_name must be resolved before deployment")
	}

	var changes []catalog.PermissionsChange
	for _, ga := range state.EmbeddedSlice {
		change := catalog.PermissionsChange{
			Principal: ga.Principal,
			Add:       ga.Privileges,
		}
		// Remove all other privileges unless ALL_PRIVILEGES is being granted
		// (it would conflict with appearing in both Add and Remove).
		if !slices.Contains(ga.Privileges, catalog.PrivilegeAllPrivileges) {
			change.Remove = []catalog.Privilege{catalog.PrivilegeAllPrivileges}
		}
		changes = append(changes, change)
	}

	_, err := r.client.Grants.Update(ctx, catalog.UpdatePermissions{
		SecurableType: state.SecurableType,
		FullName:      state.FullName,
		Changes:       changes,
	})
	return err
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
			privs := make([]catalog.Privilege, len(assignment.Privileges))
			copy(privs, assignment.Privileges)
			assignments = append(assignments, catalog.PrivilegeAssignment{
				Principal:       assignment.Principal,
				Privileges:      privs,
				ForceSendFields: nil,
			})
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return assignments, nil
}

func sortPriviliges(privileges []catalog.Privilege) {
	slices.Sort(privileges)
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

func makeGrantsID(securableType, fullName string) string {
	return securableType + "/" + fullName
}
