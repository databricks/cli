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
	"registered_models": "function",
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

		// Backend sorts privileges, so we sort here as well.
		sortPriviliges(privileges)

		grants = append(grants, GrantAssignment{
			Principal:  principal,
			Privileges: privileges,
		})
	}

	return &structvar.StructVar{
		Value: &GrantsState{
			SecurableType: securableType,
			FullName:      "",
			Grants:        grants,
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
		Grants:        assignments,
	}, nil
}

func (r *ResourceGrants) DoCreate(ctx context.Context, state *GrantsState) (string, *GrantsState, error) {
	err := r.applyGrants(ctx, state)
	if err != nil {
		return "", nil, err
	}

	return makeGrantsID(state.SecurableType, state.FullName), nil, nil
}

func (r *ResourceGrants) DoUpdate(ctx context.Context, _ string, state *GrantsState) (*GrantsState, error) {
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

	// For each principal in the config, add their grants and remove everything else
	for _, grantAssignment := range state.Grants {
		changes = append(changes, catalog.PermissionsChange{
			Principal:       grantAssignment.Principal,
			Add:             grantAssignment.Privileges,
			Remove:          []catalog.Privilege{catalog.PrivilegeAllPrivileges},
			ForceSendFields: nil,
		})
	}

	_, err := r.client.Grants.Update(ctx, catalog.UpdatePermissions{
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
	return assignments, nil
}

func sortPriviliges(privileges []catalog.Privilege) {
	sort.Slice(privileges, func(i, j int) bool {
		return privileges[i] < privileges[j]
	})
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
