package tfpermissions

import (
	"context"
	"errors"
	"strings"

	"github.com/databricks/cli/libs/tfpermissions/entity"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// NewPermissionsAPI creates PermissionsAPI instance from WorkspaceClient
func NewPermissionsAPI(w *databricks.WorkspaceClient) PermissionsAPI {
	return PermissionsAPI{
		client: w,
	}
}

// PermissionsAPI exposes general permission related methods
type PermissionsAPI struct {
	client *databricks.WorkspaceClient
}

// safePutWithOwner is a workaround for the limitation where warehouse without owners cannot have IS_OWNER set
func (a PermissionsAPI) safePutWithOwner(ctx context.Context, objectID string, objectACL []iam.AccessControlRequest, mapping resourcePermissions, ownerOpt string) error {
	idParts := strings.Split(objectID, "/")
	id := idParts[len(idParts)-1]
	// fmt.Fprintf(os.Stderr, "objectID=%q id=%q\n", objectID, id)
	withOwner := mapping.addOwnerPermissionIfNeeded(objectACL, ownerOpt)
	_, err := a.client.Permissions.Set(ctx, iam.SetObjectPermissions{
		RequestObjectId:   id,
		RequestObjectType: mapping.requestObjectType,
		AccessControlList: withOwner,
	})
	if err != nil {
		if strings.Contains(err.Error(), "with no existing owner must provide a new owner") {
			_, err = a.client.Permissions.Set(ctx, iam.SetObjectPermissions{
				RequestObjectId:   id,
				RequestObjectType: mapping.requestObjectType,
				AccessControlList: objectACL,
			})
		}
		return err
	}
	return nil
}

func (a PermissionsAPI) getCurrentUser(ctx context.Context) (string, error) {
	me, err := a.client.CurrentUser.Me(ctx)
	if err != nil {
		return "", err
	}
	return me.UserName, nil
}

// Update updates object permissions. Technically, it's using method named SetOrDelete, but here we do more
func (a PermissionsAPI) Update(ctx context.Context, objectID string, entity entity.PermissionsEntity, mapping resourcePermissions) error {
	currentUser, err := a.getCurrentUser(ctx)
	if err != nil {
		return err
	}
	// this logic was moved from CustomizeDiff because of undeterministic auth behavior
	// in the corner-case scenarios.
	// see https://github.com/databricks/terraform-provider-databricks/issues/2052
	err = mapping.validate(ctx, entity, currentUser)
	if err != nil {
		return err
	}
	prepared, err := mapping.prepareForUpdate(objectID, entity, currentUser)
	if err != nil {
		return err
	}
	return a.safePutWithOwner(ctx, objectID, prepared.AccessControlList, mapping, currentUser)
}

// Delete gracefully removes permissions of non-admin users. After this operation, the object is managed
// by the current user and admin group. If the resource has IS_OWNER permissions, they are reset to the
// object creator, if it can be determined.
func (a PermissionsAPI) Delete(ctx context.Context, objectID string, mapping resourcePermissions) error {
	objectACL, err := a.readRaw(ctx, objectID, mapping)
	if err != nil {
		return err
	}
	accl, err := mapping.prepareForDelete(objectACL, func() (string, error) {
		return a.getCurrentUser(ctx)
	})
	if err != nil {
		return err
	}
	resourceStatus, err := mapping.getObjectStatus(ctx, a.client, objectID)
	if err != nil {
		return err
	}
	// Do not bother resetting permissions for deleted resources
	if !resourceStatus.exists {
		return nil
	}
	return a.safePutWithOwner(ctx, objectID, accl, mapping, resourceStatus.creator)
}

func (a PermissionsAPI) ReadRaw(ctx context.Context, objectID string, mapping resourcePermissions) (*iam.ObjectPermissions, error) {
	return a.readRaw(ctx, objectID, mapping)
}

func (a PermissionsAPI) readRaw(ctx context.Context, objectID string, mapping resourcePermissions) (*iam.ObjectPermissions, error) {
	idParts := strings.Split(objectID, "/")
	id := idParts[len(idParts)-1]

	// TODO: This a temporary measure to implement retry on 504 until this is
	// supported natively in the Go SDK.
	// permissions, err := common.RetryOn504(ctx, func(ctx context.Context) (*iam.ObjectPermissions, error) {
	permissions, err := a.client.Permissions.Get(ctx, iam.GetPermissionRequest{
		RequestObjectId:   id,
		RequestObjectType: mapping.requestObjectType,
	})

	var apiErr *apierr.APIError
	// https://github.com/databricks/terraform-provider-databricks/issues/1227
	// platform propagates INVALID_STATE error for auto-purged clusters in
	// the permissions api. this adds "a logical fix" also here, not to introduce
	// cross-package dependency on "clusters".
	if errors.As(err, &apiErr) && strings.Contains(apiErr.Message, "Cannot access cluster") && apiErr.StatusCode == 400 {
		apiErr.StatusCode = 404
		apiErr.ErrorCode = "RESOURCE_DOES_NOT_EXIST"
		err = apiErr
	}
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// Read gets all relevant permissions for the object, including inherited ones
func (a PermissionsAPI) Read(ctx context.Context, objectID string, mapping resourcePermissions, existing entity.PermissionsEntity, me string) (entity.PermissionsEntity, error) {
	permissions, err := a.readRaw(ctx, objectID, mapping)
	if err != nil {
		return entity.PermissionsEntity{}, err
	}
	return mapping.prepareResponse(objectID, permissions, existing, me)
}
