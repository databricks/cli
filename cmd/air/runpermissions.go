package aircmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

const (
	// mlflowPermissionTimeout bounds the best-effort experiment permission grant.
	// The run's MLflow experiment id comes from runs/get-output, which may lag the
	// submit by a moment; this caps how long we wait for it and keeps a
	// not-yet-populated output from stalling submit.
	mlflowPermissionTimeout = 2 * time.Second
	// mlflowPermissionPollInterval is how often runs/get-output is retried while
	// waiting for the experiment id to appear.
	mlflowPermissionPollInterval = 100 * time.Millisecond
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

// experimentPermissionLevel maps a job permission level onto the nearest MLflow
// experiment permission level. Experiments only support CAN_READ/CAN_EDIT/
// CAN_MANAGE, so the job-oriented levels fold onto those. ok is false when the
// level has no experiment equivalent and the grant should be skipped.
func experimentPermissionLevel(level string) (ml.ExperimentPermissionLevel, bool) {
	switch iam.PermissionLevel(level) {
	case iam.PermissionLevelCanView:
		return ml.ExperimentPermissionLevelCanRead, true
	case iam.PermissionLevelCanManageRun:
		return ml.ExperimentPermissionLevelCanEdit, true
	case iam.PermissionLevelCanManage, iam.PermissionLevelIsOwner:
		return ml.ExperimentPermissionLevelCanManage, true
	default:
		return "", false
	}
}

// experimentAccessControl builds an experiment ACL entry from a validated grant.
// ok is false when the grant's level has no experiment equivalent.
func experimentAccessControl(p permission) (ml.ExperimentAccessControlRequest, bool) {
	level, ok := experimentPermissionLevel(p.Level)
	if !ok {
		return ml.ExperimentAccessControlRequest{}, false
	}
	acl := ml.ExperimentAccessControlRequest{PermissionLevel: level}
	switch {
	case p.UserName != nil:
		acl.UserName = *p.UserName
	case p.GroupName != nil:
		acl.GroupName = *p.GroupName
	case p.ServicePrincipalName != nil:
		acl.ServicePrincipalName = *p.ServicePrincipalName
	}
	return acl, true
}

// resolveSubmittedExperimentID polls runs/get-output for the run's MLflow
// experiment id until it appears or ctx expires. The id is the run's own
// experiment, so granting on it can never touch a different user's experiment —
// unlike resolving a bare experiment name, which would have to guess the owning
// directory (and has no default for a service principal). Returns "" if the id
// doesn't become available in time.
func resolveSubmittedExperimentID(ctx context.Context, w *databricks.WorkspaceClient, run *jobs.Run) string {
	ticker := time.NewTicker(mlflowPermissionPollInterval)
	defer ticker.Stop()
	for {
		if out := aiRuntimeTaskOutput(ctx, w, run); out != nil && out.MlflowExperimentId != "" {
			return out.MlflowExperimentId
		}

		select {
		case <-ctx.Done():
			return ""
		case <-ticker.C:
		}
	}
}

// grantExperimentPermissions resolves the run's MLflow experiment and adds the
// configured ACLs to it. Unlike jobs, whose permissions are set remotely, the
// experiment is created by the AI Runtime backend so its ACLs must be set here
// from the client.
func grantExperimentPermissions(ctx context.Context, w *databricks.WorkspaceClient, run *jobs.Run, permissions []permission) error {
	experimentACL := make([]ml.ExperimentAccessControlRequest, 0, len(permissions))
	for _, p := range permissions {
		if acl, ok := experimentAccessControl(p); ok {
			experimentACL = append(experimentACL, acl)
		}
	}
	if len(experimentACL) == 0 {
		return nil
	}

	// Bound only the resolution: a not-yet-populated output must never stall the
	// grant indefinitely, but the UpdatePermissions call below runs on the
	// caller's context.
	resolveCtx, cancel := context.WithTimeout(ctx, mlflowPermissionTimeout)
	defer cancel()
	experimentID := resolveSubmittedExperimentID(resolveCtx, w, run)
	if experimentID == "" {
		return errors.New("could not resolve the run's MLflow experiment")
	}

	_, err := w.Experiments.UpdatePermissions(ctx, ml.ExperimentPermissionsRequest{
		ExperimentId:      experimentID,
		AccessControlList: experimentACL,
	})
	if err != nil {
		return fmt.Errorf("failed to grant experiment permissions: %w", err)
	}
	return nil
}

// grantSubmittedPermissions applies the run's configured ACLs after the submit
// result has been shown, wrapping the bounded best-effort grant in a spinner so
// it never silently delays the success line. showProgress gates the spinner
// (text mode only).
func grantSubmittedPermissions(ctx context.Context, w *databricks.WorkspaceClient, runID int64, permissions []permission, showProgress bool) {
	if len(permissions) == 0 {
		return
	}
	_ = withSpinner(ctx, showProgress, "Granting permissions…", func() error {
		applySubmittedPermissions(ctx, w, runID, permissions)
		return nil
	})
}

// applySubmittedPermissions resolves the submitted run and grants its configured
// ACLs on both the job (remotely) and the MLflow experiment (client-side).
func applySubmittedPermissions(ctx context.Context, w *databricks.WorkspaceClient, runID int64, permissions []permission) {
	if len(permissions) == 0 {
		return
	}

	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		log.Warnf(ctx, "failed to grant permissions on workload: %v", err)
		log.Warnf(ctx, "job was created successfully, but permissions could not be granted")
		return
	}

	if err := grantJobPermissions(ctx, w, strconv.FormatInt(run.JobId, 10), permissions); err != nil {
		log.Warnf(ctx, "failed to grant permissions on workload: %v", err)
		log.Warnf(ctx, "job was created successfully, but permissions could not be granted")
	}

	// Best-effort: a missing experiment must never fail an otherwise-successful
	// submit.
	if err := grantExperimentPermissions(ctx, w, run, permissions); err != nil {
		log.Warnf(ctx, "failed to grant experiment permissions: %v", err)
		log.Warnf(ctx, "job was created successfully, but experiment permissions could not be granted")
	}
}
