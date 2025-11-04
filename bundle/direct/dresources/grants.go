package dresources

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

var grantResourceToSecurableType = map[string]string{
	"schemas":           "schema",
	"volumes":           "volume",
	"registered_models": "registered-model",
}

type GrantAssignment struct {
	Principal  string              `json:"principal"`
	Privileges []catalog.Privilege `json:"privileges,omitempty"`
}

type GrantsState struct {
	SecurableType string            `json:"securable_type"`
	FullName      string            `json:"full_name"`
	Grants        []GrantAssignment `json:"grants,omitempty"`
}

// PrepareGrantsInputConfig converts the grants slice in bundle configuration into a StructVar the planner can resolve.
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

	rv := reflect.ValueOf(inputConfig)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return nil, fmt.Errorf("inputConfig must be a pointer to a slice, got: %T", inputConfig)
	}

	sliceValue := rv.Elem()
	grants := make([]GrantAssignment, 0, sliceValue.Len())
	for i := range sliceValue.Len() {
		elem := sliceValue.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		if elem.Kind() != reflect.Struct {
			return nil, fmt.Errorf("grant element must be a struct, got %s", elem.Kind())
		}

		principalField := elem.FieldByName("Principal")
		if !principalField.IsValid() {
			return nil, errors.New("grant element missing Principal field")
		}
		principal := principalField.String()

		privilegesField := elem.FieldByName("Privileges")
		if !privilegesField.IsValid() {
			return nil, errors.New("grant element missing Privileges field")
		}
		if privilegesField.Kind() != reflect.Slice {
			return nil, errors.New("grant Privileges field must be a slice")
		}

		privileges := make([]catalog.Privilege, 0, privilegesField.Len())
		for j := range privilegesField.Len() {
			item := privilegesField.Index(j)
			if item.Kind() != reflect.String {
				return nil, fmt.Errorf("privilege must be a string, got %s", item.Kind())
			}
			privileges = append(privileges, catalog.Privilege(item.String()))
		}

		//normalizePrivileges(privileges)
		grants = append(grants, GrantAssignment{
			Principal:  principal,
			Privileges: privileges,
		})
	}

	normalizeGrantAssignments(grants)

	return &structvar.StructVar{
		Config: &GrantsState{
			SecurableType: securableType,
			FullName:      "",
			Grants:        grants,
		},
		Refs: map[string]string{
			"full_name": "${" + baseNode + ".id}",
		},
	}, nil
}

func extractGrantResourceType(node string) (string, error) {
	// XXX use CutPrefix
	idx := strings.Index(node, "resources.")
	if idx == -1 {
		return "", fmt.Errorf("cannot extract resource type from %q", node)
	}
	rest := node[idx+len("resources."):]
	parts := strings.Split(rest, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("cannot extract resource type from %q", node)
	}
	return parts[0], nil
}

func normalizePrivileges(privileges []catalog.Privilege) {
	sort.Slice(privileges, func(i, j int) bool {
		return privileges[i] < privileges[j]
	})
}

func normalizeGrantAssignments(assignments []GrantAssignment) {
	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].Principal == assignments[j].Principal {
			if len(assignments[i].Privileges) != len(assignments[j].Privileges) {
				return len(assignments[i].Privileges) < len(assignments[j].Privileges)
			}
			for k := range assignments[i].Privileges {
				if assignments[i].Privileges[k] == assignments[j].Privileges[k] {
					continue
				}
				return assignments[i].Privileges[k] < assignments[j].Privileges[k]
			}
			return false
		}
		return assignments[i].Principal < assignments[j].Principal
	})
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

func (r *ResourceGrants) DoRefresh(ctx context.Context, id string) (*GrantsState, error) {
	securableType, fullName, err := parseGrantsID(id)
	if err != nil {
		return nil, err
	}

	assignments, err := r.listGrants(ctx, securableType, fullName)
	if err != nil {
		return nil, err
	}

	normalizeGrantAssignments(assignments)

	return &GrantsState{
		SecurableType: securableType,
		FullName:      fullName,
		Grants:        assignments,
	}, nil
}

func (r *ResourceGrants) DoCreate(ctx context.Context, state *GrantsState) (string, error) {
	err := r.applyGrants(ctx, state)
	if err != nil {
		return "", err
	}

	return makeGrantsID(state.SecurableType, state.FullName), nil
}

func (r *ResourceGrants) DoUpdate(ctx context.Context, _ string, state *GrantsState) error {
	return r.applyGrants(ctx, state)
}

func (r *ResourceGrants) DoDelete(ctx context.Context, _ string) error {
	// Leaving grants untouched on delete. Users can manage them manually afterwards.
	return nil
}

func (r *ResourceGrants) applyGrants(ctx context.Context, state *GrantsState) error {
	if state.FullName == "" {
		return errors.New("internal error: grants full_name must be resolved before deployment")
	}

	for i := range state.Grants {
		normalizePrivileges(state.Grants[i].Privileges)
	}
	normalizeGrantAssignments(state.Grants)

	assignments, err := r.listGrants(ctx, state.SecurableType, state.FullName)
	if err != nil {
		return err
	}

	current := grantsToMap(assignments)
	desired := grantsToMap(state.Grants)

	changes := diffGrants(desired, current)
	if len(changes) == 0 {
		return nil
	}

	_, err = r.client.Grants.Update(ctx, catalog.UpdatePermissions{
		SecurableType: state.SecurableType,
		FullName:      state.FullName,
		Changes:       changes,
	})
	return err
}

func (r *ResourceGrants) listGrants(ctx context.Context, securableType, fullName string) ([]GrantAssignment, error) {
	var assignments []GrantAssignment
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
			//normalizePrivileges(privs)
			assignments = append(assignments, GrantAssignment{
				Principal:  assignment.Principal,
				Privileges: privs,
			})
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	normalizeGrantAssignments(assignments)
	return assignments, nil
}

func diffGrants(desired, current map[string]map[catalog.Privilege]struct{}) []catalog.PermissionsChange {
	var changes []catalog.PermissionsChange

	desiredPrincipals := make([]string, 0, len(desired))
	for principal := range desired {
		desiredPrincipals = append(desiredPrincipals, principal)
	}
	sort.Strings(desiredPrincipals)
	for _, principal := range desiredPrincipals {
		desiredPrivs := desired[principal]
		currentPrivs := current[principal]
		add := privilegesDiff(desiredPrivs, currentPrivs)
		remove := privilegesDiff(currentPrivs, desiredPrivs)
		if len(add) == 0 && len(remove) == 0 {
			continue
		}
		changes = append(changes, catalog.PermissionsChange{
			Add:             add,
			Principal:       principal,
			Remove:          remove,
			ForceSendFields: nil,
		})
	}

	var removals []catalog.PermissionsChange
	currentPrincipals := make([]string, 0, len(current))
	for principal := range current {
		currentPrincipals = append(currentPrincipals, principal)
	}
	sort.Strings(currentPrincipals)
	for _, principal := range currentPrincipals {
		currentPrivs := current[principal]
		if _, ok := desired[principal]; ok {
			continue
		}
		remove := privilegesDiff(currentPrivs, nil)
		if len(remove) == 0 {
			continue
		}
		removals = append(removals, catalog.PermissionsChange{
			Add:             nil,
			Principal:       principal,
			Remove:          remove,
			ForceSendFields: nil,
		})
	}

	changes = append(changes, removals...)
	return changes
}

func privilegesDiff(a, b map[catalog.Privilege]struct{}) []catalog.Privilege {
	if len(a) == 0 {
		return nil
	}

	var diff []catalog.Privilege
	for privilege := range a {
		if b != nil {
			if _, ok := b[privilege]; ok {
				continue
			}
		}
		diff = append(diff, privilege)
	}
	sort.Slice(diff, func(i, j int) bool { return diff[i] < diff[j] })
	return diff
}

func grantsToMap(grants []GrantAssignment) map[string]map[catalog.Privilege]struct{} {
	result := make(map[string]map[catalog.Privilege]struct{}, len(grants))
	for _, grant := range grants {
		privs := make(map[catalog.Privilege]struct{}, len(grant.Privileges))
		for _, privilege := range grant.Privileges {
			privs[privilege] = struct{}{}
		}
		result[grant.Principal] = privs
	}
	return result
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
