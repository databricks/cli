package lock

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// The server expires a version's lease if it does not receive a heartbeat
// within a 2-minute TTL; we heartbeat well inside that window.
const defaultHeartbeatInterval = 30 * time.Second

// DeploymentStatus indicates whether the deployment operation succeeded or failed.
type DeploymentStatus int

const (
	DeploymentSuccess DeploymentStatus = iota
	DeploymentFailure
)

// DeploymentVersionRecorder records each deploy/destroy as a version with the
// Deployment Metadata Service (DMS). It runs alongside the workspace-filesystem
// lock (lock.Acquire / lock.Release): a version is created after the lock is
// acquired and completed before it is released.
//
// Recording is gated by experimental.record_deployment_history; when disabled,
// CreateVersion and CompleteVersion are no-ops. The deployment ID is the state
// lineage (resources.json), so a bundle deployment maps one-to-one to a DMS
// deployment record.
type DeploymentVersionRecorder struct {
	b       *bundle.Bundle
	goal    Goal
	enabled bool

	// validateVersion is set when applying a pre-computed plan: the plan's
	// version_id must match the deployment's current version, otherwise the plan
	// is stale and is rejected.
	validateVersion bool
	expectedVersion string

	// populated by CreateVersion
	svc           sdkbundle.BundleInterface
	deploymentID  string
	versionNum    int64
	stopHeartbeat context.CancelFunc
}

// NewDeploymentVersionRecorder returns a recorder for the given goal. The
// returned recorder is a no-op unless experimental.record_deployment_history
// is set. When plan is non-nil (applying a pre-computed plan), the plan's
// version_id is validated against the live deployment version at lock time.
func NewDeploymentVersionRecorder(b *bundle.Bundle, goal Goal, plan *deployplan.Plan) *DeploymentVersionRecorder {
	enabled := b.Config.Experimental != nil && b.Config.Experimental.RecordDeploymentHistory
	r := &DeploymentVersionRecorder{b: b, goal: goal, enabled: enabled}
	if plan != nil {
		r.validateVersion = true
		r.expectedVersion = plan.VersionId
	}
	return r
}

// CreateVersion registers a new deployment version with DMS, claiming it for the
// duration of the deployment. No-op when recording is disabled.
func (r *DeploymentVersionRecorder) CreateVersion(ctx context.Context) error {
	if !r.enabled {
		return nil
	}

	versionType, ok := goalToVersionType(r.goal)
	if !ok {
		return fmt.Errorf("%s is not supported with the deployment metadata service", r.goal)
	}

	svc := r.b.WorkspaceClient(ctx).Bundle

	// The deployment ID is the state lineage. GetOrInitLineage generates one on
	// the first deploy and stores it so the deploy persists the same value.
	r.deploymentID = r.b.DeploymentBundle.StateDB.GetOrInitLineage()

	versionID, err := createDeploymentVersion(ctx, r.b, svc, r.deploymentID, versionType, r.expectedVersion, r.validateVersion)
	if err != nil {
		return err
	}

	versionNum, err := strconv.ParseInt(versionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse version ID %q: %w", versionID, err)
	}

	// Set svc only after the version is created so CompleteVersion is a no-op
	// when creation failed (e.g. a stale plan was rejected): there is no version
	// to complete.
	r.svc = svc
	r.versionNum = versionNum
	r.stopHeartbeat = startHeartbeat(ctx, r.svc, r.deploymentID, versionID)
	return nil
}

// CompleteVersion finalizes the version created by CreateVersion. No-op when
// recording is disabled or no version was created.
func (r *DeploymentVersionRecorder) CompleteVersion(ctx context.Context, status DeploymentStatus) error {
	if !r.enabled || r.svc == nil {
		return nil
	}

	if r.stopHeartbeat != nil {
		r.stopHeartbeat()
	}

	versionIDStr := strconv.FormatInt(r.versionNum, 10)
	versionName := fmt.Sprintf("deployments/%s/versions/%s", r.deploymentID, versionIDStr)

	reason := sdkbundle.VersionCompleteVersionCompleteSuccess
	if status == DeploymentFailure {
		reason = sdkbundle.VersionCompleteVersionCompleteFailure
	}

	_, err := r.svc.CompleteVersion(ctx, sdkbundle.CompleteVersionRequest{
		Name:             versionName,
		CompletionReason: reason,
	})
	if err != nil {
		return err
	}
	log.Infof(ctx, "Completed deployment version: deployment=%s version=%s reason=%s", r.deploymentID, versionIDStr, reason)

	// For destroy operations, delete the deployment record after the version
	// completes successfully.
	if status == DeploymentSuccess && r.goal == GoalDestroy {
		err = r.svc.DeleteDeployment(ctx, sdkbundle.DeleteDeploymentRequest{
			Name: "deployments/" + r.deploymentID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	}

	return nil
}

// createDeploymentVersion ensures the deployment record exists, then creates a
// new version. The deployment ID is the state lineage: we GetDeployment first
// and only CreateDeployment when it does not exist yet.
//
// When validateVersion is set (applying a pre-computed plan), the deployment's
// current version must equal expectedVersion — the version the plan was
// generated against. Otherwise the deployment moved since the plan was created
// and the plan is rejected as stale.
func createDeploymentVersion(ctx context.Context, b *bundle.Bundle, svc sdkbundle.BundleInterface, deploymentID string, versionType sdkbundle.VersionType, expectedVersion string, validateVersion bool) (versionID string, err error) {
	dep, getErr := svc.GetDeployment(ctx, sdkbundle.GetDeploymentRequest{
		Name: "deployments/" + deploymentID,
	})
	switch {
	case errors.Is(getErr, apierr.ErrNotFound):
		if validateVersion && expectedVersion != "" {
			return "", outdatedPlanErr(expectedVersion, "")
		}
		// Fresh deployment: create the record and start at version 1.
		_, createErr := svc.CreateDeployment(ctx, sdkbundle.CreateDeploymentRequest{
			DeploymentId: deploymentID,
			Deployment: sdkbundle.Deployment{
				TargetName: b.Config.Bundle.Target,
			},
		})
		if createErr != nil {
			return "", fmt.Errorf("failed to create deployment: %w", createErr)
		}
		versionID = "1"
	case getErr != nil:
		return "", fmt.Errorf("failed to get deployment: %w", getErr)
	default:
		if validateVersion && dep.LastVersionId != expectedVersion {
			return "", outdatedPlanErr(expectedVersion, dep.LastVersionId)
		}
		// Existing deployment: increment the last version to get the next one.
		lastVersion, parseErr := strconv.ParseInt(dep.LastVersionId, 10, 64)
		if parseErr != nil {
			return "", fmt.Errorf("failed to parse last_version_id %q: %w", dep.LastVersionId, parseErr)
		}
		versionID = strconv.FormatInt(lastVersion+1, 10)
	}

	// The server validates that versionID equals last_version_id + 1 and returns
	// ABORTED otherwise (e.g. a concurrent deploy already created this version).
	version, versionErr := svc.CreateVersion(ctx, sdkbundle.CreateVersionRequest{
		Parent:    "deployments/" + deploymentID,
		VersionId: versionID,
		Version: sdkbundle.Version{
			CliVersion:  build.GetInfo().Version,
			VersionType: versionType,
			TargetName:  b.Config.Bundle.Target,
		},
	})
	if versionErr != nil {
		return "", fmt.Errorf("failed to create deployment version: %w", versionErr)
	}

	log.Infof(ctx, "Created deployment version: deployment=%s version=%s", deploymentID, version.VersionId)
	return versionID, nil
}

// outdatedPlanErr returns the error reported when a pre-computed plan was
// generated against a deployment version that no longer matches the live one.
func outdatedPlanErr(expectedVersion, currentVersion string) error {
	if currentVersion == "" {
		return fmt.Errorf("plan is outdated: it was generated against deployment version %s, but the deployment no longer exists. Please run 'bundle plan' again", expectedVersion)
	}
	return fmt.Errorf("plan is outdated: it was generated against deployment version %s, but the current version is %s. Please run 'bundle plan' again", expectedVersion, currentVersion)
}

// startHeartbeat starts a background goroutine that sends heartbeats to keep
// the deployment version's lease alive. Returns a cancel function to stop it.
func startHeartbeat(ctx context.Context, svc sdkbundle.BundleInterface, deploymentID, versionID string) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	versionName := fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID)

	go func() {
		ticker := time.NewTicker(defaultHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := svc.Heartbeat(ctx, sdkbundle.HeartbeatRequest{Name: versionName})
				if err != nil {
					// A 409 ABORTED is expected if the version was completed
					// between the ticker firing and the heartbeat.
					if isAbortedErr(err) {
						log.Debugf(ctx, "Heartbeat stopped: version already completed")
						return
					}
					log.Warnf(ctx, "Failed to send deployment heartbeat: %v", err)
				} else {
					log.Debugf(ctx, "Deployment heartbeat sent: deployment=%s version=%s", deploymentID, versionID)
				}
			}
		}
	}()

	return cancel
}

// isAbortedErr reports whether err is an HTTP 409 ABORTED from the DMS API.
func isAbortedErr(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.StatusCode == http.StatusConflict && apiErr.ErrorCode == "ABORTED"
}

// goalToVersionType maps a deployment goal to a DMS VersionType.
// Returns false for goals not supported by the metadata service (bind/unbind).
func goalToVersionType(goal Goal) (sdkbundle.VersionType, bool) {
	switch goal {
	case GoalDeploy:
		return sdkbundle.VersionTypeVersionTypeDeploy, true
	case GoalDestroy:
		return sdkbundle.VersionTypeVersionTypeDestroy, true
	default:
		return "", false
	}
}
