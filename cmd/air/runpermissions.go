package aircmd

import (
	"context"
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
	// The AI Runtime backend creates the MLflow experiment around submit time, so
	// we give it a brief window to become resolvable; capping the wait keeps a
	// not-yet-created experiment from noticeably delaying submit.
	mlflowPermissionTimeout = 5 * time.Second
	// mlflowPermissionPollInterval is how often get-by-name is retried while
	// waiting for the experiment to appear.
	mlflowPermissionPollInterval = 500 * time.Millisecond
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

// resolveExperimentID polls get-by-name until the experiment is resolvable or
// ctx expires. The AI Runtime backend creates the experiment around submit time,
// so a brief wait covers the window where it isn't visible yet.
func resolveExperimentID(ctx context.Context, w *databricks.WorkspaceClient, experimentName string) (string, error) {
	ticker := time.NewTicker(mlflowPermissionPollInterval)
	defer ticker.Stop()
	for {
		resp, err := w.Experiments.GetByName(ctx, ml.GetByNameRequest{ExperimentName: experimentName})
		if err == nil && resp.Experiment != nil {
			return resp.Experiment.ExperimentId, nil
		}

		select {
		case <-ctx.Done():
			if err != nil {
				return "", fmt.Errorf("failed to resolve experiment %q: %w", experimentName, err)
			}
			return "", fmt.Errorf("experiment %q not found", experimentName)
		case <-ticker.C:
		}
	}
}

// grantExperimentPermissions resolves the run's MLflow experiment and adds the
// configured ACLs to it. Unlike jobs, whose permissions are set remotely, the
// experiment is created by the AI Runtime backend so its ACLs must be set here
// from the client.
func grantExperimentPermissions(ctx context.Context, w *databricks.WorkspaceClient, experimentName string, permissions []permission) error {
	experimentACL := make([]ml.ExperimentAccessControlRequest, 0, len(permissions))
	for _, p := range permissions {
		if acl, ok := experimentAccessControl(p); ok {
			experimentACL = append(experimentACL, acl)
		}
	}
	if len(experimentACL) == 0 {
		return nil
	}

	experimentID, err := resolveExperimentID(ctx, w, experimentName)
	if err != nil {
		return err
	}

	_, err = w.Experiments.UpdatePermissions(ctx, ml.ExperimentPermissionsRequest{
		ExperimentId:      experimentID,
		AccessControlList: experimentACL,
	})
	if err != nil {
		return fmt.Errorf("failed to grant experiment permissions: %w", err)
	}
	return nil
}

// submittedExperimentName returns the full MLflow experiment name resolved onto
// the run's AI Runtime task, or "" when the run has no experiment. Unlike
// experimentName in format.go it keeps the leading /Users/<user>/ prefix, which
// get-by-name requires.
func submittedExperimentName(run *jobs.Run) string {
	if len(run.Tasks) == 0 {
		return ""
	}
	task := run.Tasks[0].GenAiComputeTask
	if task == nil {
		return ""
	}
	return task.MlflowExperimentName
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

	experimentName := submittedExperimentName(run)
	if experimentName == "" {
		return
	}
	// Best-effort and time-bounded: the experiment may not be resolvable yet, and
	// a missing experiment must never fail an otherwise-successful submit.
	expCtx, cancel := context.WithTimeout(ctx, mlflowPermissionTimeout)
	defer cancel()
	if err := grantExperimentPermissions(expCtx, w, experimentName, permissions); err != nil {
		log.Warnf(ctx, "failed to grant experiment permissions: %v", err)
		log.Warnf(ctx, "job was created successfully, but experiment permissions could not be granted")
	}
}
