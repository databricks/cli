package aircmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// permissionAccessControl builds an ACL entry for a validated permission grant.
func permissionAccessControl(p permission) iam.AccessControlRequest {
	acl := iam.AccessControlRequest{PermissionLevel: iam.PermissionLevel(p.Level)}
	switch {
	case p.UserName != nil:
		acl.UserName = *p.UserName
	case p.GroupName != nil:
		acl.GroupName = *p.GroupName
	case p.ServicePrincipalName != nil:
		acl.ServicePrincipalName = *p.ServicePrincipalName
	}
	return acl
}

// grantJobPermissions adds the configured ACLs to a job.
func grantJobPermissions(ctx context.Context, w *databricks.WorkspaceClient, jobID string, permissions []permission) error {
	if len(permissions) == 0 {
		return nil
	}

	jobACL := make([]iam.AccessControlRequest, 0, len(permissions))
	for _, p := range permissions {
		jobACL = append(jobACL, permissionAccessControl(p))
	}

	_, err := w.Permissions.Update(ctx, iam.UpdateObjectPermissions{
		RequestObjectType: "jobs",
		RequestObjectId:   jobID,
		AccessControlList: jobACL,
	})
	if err != nil {
		return fmt.Errorf("failed to grant job permissions: %w", err)
	}
	return nil
}

// applySubmittedPermissions resolves the submitted job and adds its configured ACLs.
func applySubmittedPermissions(ctx context.Context, w *databricks.WorkspaceClient, runID int64, permissions []permission) {
	if len(permissions) == 0 {
		return
	}

	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err == nil {
		err = grantJobPermissions(ctx, w, strconv.FormatInt(run.JobId, 10), permissions)
	}
	if err != nil {
		log.Warnf(ctx, "failed to grant permissions on workload: %v", err)
		log.Warnf(ctx, "job was created successfully, but permissions could not be granted")
	}
}
