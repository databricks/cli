package aircmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// permissionExperimentName returns the full MLflow experiment path for a run.
func permissionExperimentName(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig) (string, error) {
	if cfg.MLflowExperimentDirectory != nil {
		return strings.TrimRight(*cfg.MLflowExperimentDirectory, "/") + "/" + cfg.ExperimentName, nil
	}

	email, err := currentUserEmail(ctx, w)
	if err != nil {
		return "", err
	}
	return "/Users/" + email + "/" + cfg.ExperimentName, nil
}

// getOrCreateMLflowExperiment resolves the experiment ID, creating it when absent.
func getOrCreateMLflowExperiment(ctx context.Context, w *databricks.WorkspaceClient, name, artifactLocation string) (string, error) {
	existing, err := w.Experiments.GetByName(ctx, ml.GetByNameRequest{ExperimentName: name})
	if err == nil && existing.Experiment != nil && existing.Experiment.ExperimentId != "" {
		return existing.Experiment.ExperimentId, nil
	}
	if err != nil && !errors.Is(err, apierr.ErrNotFound) {
		return "", fmt.Errorf("failed to get MLflow experiment %q: %w", name, err)
	}

	created, err := w.Experiments.CreateExperiment(ctx, ml.CreateExperiment{
		Name:             name,
		ArtifactLocation: artifactLocation,
	})
	if err == nil {
		return created.ExperimentId, nil
	}
	if !errors.Is(err, apierr.ErrAlreadyExists) && !errors.Is(err, apierr.ErrResourceAlreadyExists) {
		return "", fmt.Errorf("failed to create MLflow experiment %q: %w", name, err)
	}

	existing, err = w.Experiments.GetByName(ctx, ml.GetByNameRequest{ExperimentName: name})
	if err != nil {
		return "", fmt.Errorf("failed to get concurrently created MLflow experiment %q: %w", name, err)
	}
	if existing.Experiment == nil || existing.Experiment.ExperimentId == "" {
		return "", fmt.Errorf("MLflow experiment %q exists but has no experiment ID", name)
	}
	return existing.Experiment.ExperimentId, nil
}

// experimentPermissionLevel maps a Jobs permission level to its MLflow equivalent.
func experimentPermissionLevel(level string) (iam.PermissionLevel, error) {
	switch iam.PermissionLevel(level) {
	case iam.PermissionLevelCanView:
		return iam.PermissionLevelCanRead, nil
	case iam.PermissionLevelCanManageRun:
		return iam.PermissionLevelCanEdit, nil
	case iam.PermissionLevelCanManage, iam.PermissionLevelIsOwner:
		return iam.PermissionLevelCanManage, nil
	default:
		return "", fmt.Errorf("unsupported AIR permission level %q", level)
	}
}

// permissionAccessControl builds an ACL entry for a validated permission grant.
func permissionAccessControl(p permission, level iam.PermissionLevel) iam.AccessControlRequest {
	acl := iam.AccessControlRequest{PermissionLevel: level}
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

// grantWorkloadPermissions adds the configured ACLs to a job and its experiment.
func grantWorkloadPermissions(ctx context.Context, w *databricks.WorkspaceClient, jobID, experimentID string, permissions []permission) error {
	if len(permissions) == 0 {
		return nil
	}

	jobACL := make([]iam.AccessControlRequest, 0, len(permissions))
	experimentACL := make([]iam.AccessControlRequest, 0, len(permissions))
	for _, p := range permissions {
		experimentLevel, err := experimentPermissionLevel(p.Level)
		if err != nil {
			return err
		}
		jobACL = append(jobACL, permissionAccessControl(p, iam.PermissionLevel(p.Level)))
		experimentACL = append(experimentACL, permissionAccessControl(p, experimentLevel))
	}

	_, err := w.Permissions.Update(ctx, iam.UpdateObjectPermissions{
		RequestObjectType: "jobs",
		RequestObjectId:   jobID,
		AccessControlList: jobACL,
	})
	if err != nil {
		return fmt.Errorf("failed to grant job permissions: %w", err)
	}

	_, err = w.Permissions.Update(ctx, iam.UpdateObjectPermissions{
		RequestObjectType: "experiments",
		RequestObjectId:   experimentID,
		AccessControlList: experimentACL,
	})
	if err != nil {
		return fmt.Errorf("failed to grant MLflow experiment permissions: %w", err)
	}
	return nil
}

// preparePermissionExperiment resolves the experiment before the workload is submitted.
func preparePermissionExperiment(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig) string {
	if len(cfg.Permissions) == 0 {
		return ""
	}

	name, err := permissionExperimentName(ctx, w, cfg)
	if err != nil {
		log.Warnf(ctx, "unable to resolve MLflow experiment name; skipping permission grants: %v", err)
		return ""
	}
	artifactLocation := ""
	if cfg.MLflowArtifactLocation != nil {
		artifactLocation = *cfg.MLflowArtifactLocation
	}
	experimentID, err := getOrCreateMLflowExperiment(ctx, w, name, artifactLocation)
	if err != nil {
		log.Warnf(ctx, "unable to get or create MLflow experiment; skipping permission grants: %v", err)
		return ""
	}
	return experimentID
}

// applySubmittedPermissions resolves the submitted job and adds its configured ACLs.
func applySubmittedPermissions(ctx context.Context, w *databricks.WorkspaceClient, runID int64, experimentID string, permissions []permission) {
	if len(permissions) == 0 || experimentID == "" {
		return
	}

	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err == nil {
		err = grantWorkloadPermissions(ctx, w, strconv.FormatInt(run.JobId, 10), experimentID, permissions)
	}
	if err != nil {
		log.Warnf(ctx, "failed to grant permissions on workload: %v", err)
		log.Warnf(ctx, "job was created successfully, but permissions could not be granted")
	}
}
