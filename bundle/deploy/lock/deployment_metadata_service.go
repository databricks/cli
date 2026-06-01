package lock

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	clibundle "github.com/databricks/cli/bundle"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

const defaultHeartbeatInterval = 30 * time.Second

// metadataServiceLock records each deployment as a versioned operation with
// the Deployment Metadata Service. It is called additively alongside the
// workspace-filesystem lock inside DeploymentLock.Acquire / Release.
//
// The deployment ID is the state lineage (read from resources.json), so a
// bundle deployment maps one-to-one to a DMS deployment record.
type metadataServiceLock struct {
	b    *clibundle.Bundle
	goal Goal

	// populated by createVersion
	svc           sdkbundle.BundleInterface
	deploymentID  string
	versionNum    int64
	stopHeartbeat context.CancelFunc
}

func newMetadataServiceLock(b *clibundle.Bundle, goal Goal) *metadataServiceLock {
	return &metadataServiceLock{b: b, goal: goal}
}

func (l *metadataServiceLock) createVersion(ctx context.Context) error {
	versionType, ok := goalToVersionType(l.goal)
	if !ok {
		return fmt.Errorf("%s is not supported with the deployment metadata service", l.goal)
	}

	l.svc = l.b.WorkspaceClient(ctx).Bundle

	// The deployment ID is the state lineage. GetOrInitLineage generates one on
	// the first deploy and stores it so the deploy persists the same value.
	l.deploymentID = l.b.DeploymentBundle.StateDB.GetOrInitLineage()

	versionID, err := acquireLock(ctx, l.b, l.svc, l.deploymentID, versionType)
	if err != nil {
		return err
	}

	versionNum, err := strconv.ParseInt(versionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse version ID %q: %w", versionID, err)
	}
	l.versionNum = versionNum
	l.stopHeartbeat = startHeartbeat(ctx, l.svc, l.deploymentID, versionID)
	return nil
}

func (l *metadataServiceLock) completeVersion(ctx context.Context, status DeploymentStatus) error {
	if l.stopHeartbeat != nil {
		l.stopHeartbeat()
	}

	versionIDStr := strconv.FormatInt(l.versionNum, 10)
	versionName := fmt.Sprintf("deployments/%s/versions/%s", l.deploymentID, versionIDStr)

	reason := sdkbundle.VersionCompleteVersionCompleteSuccess
	if status == DeploymentFailure {
		reason = sdkbundle.VersionCompleteVersionCompleteFailure
	}

	_, err := l.svc.CompleteVersion(ctx, sdkbundle.CompleteVersionRequest{
		Name:             versionName,
		CompletionReason: reason,
	})
	if err != nil {
		return err
	}
	log.Infof(ctx, "Released deployment lock: deployment=%s version=%s reason=%s", l.deploymentID, versionIDStr, reason)

	// For destroy operations, delete the deployment record after successfully
	// releasing the lock.
	if status == DeploymentSuccess && l.goal == GoalDestroy {
		err = l.svc.DeleteDeployment(ctx, sdkbundle.DeleteDeploymentRequest{
			Name: "deployments/" + l.deploymentID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	}

	return nil
}

// acquireLock ensures the deployment record exists, then creates a version to
// claim the lock. The deployment ID is the state lineage: we GetDeployment
// first and only CreateDeployment when it does not exist yet.
func acquireLock(ctx context.Context, b *clibundle.Bundle, svc sdkbundle.BundleInterface, deploymentID string, versionType sdkbundle.VersionType) (versionID string, err error) {
	dep, getErr := svc.GetDeployment(ctx, sdkbundle.GetDeploymentRequest{
		Name: "deployments/" + deploymentID,
	})
	switch {
	case errors.Is(getErr, apierr.ErrNotFound):
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
		// Existing deployment: increment the last version to get the next one.
		lastVersion, parseErr := strconv.ParseInt(dep.LastVersionId, 10, 64)
		if parseErr != nil {
			return "", fmt.Errorf("failed to parse last_version_id %q: %w", dep.LastVersionId, parseErr)
		}
		versionID = strconv.FormatInt(lastVersion+1, 10)
	}

	// CreateVersion acquires the deployment lock. The server validates that
	// versionID equals last_version_id + 1 and returns ABORTED otherwise (e.g.
	// a concurrent deploy already claimed it).
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
		return "", fmt.Errorf("failed to acquire deployment lock: %w", versionErr)
	}

	log.Infof(ctx, "Acquired deployment lock: deployment=%s version=%s", deploymentID, version.VersionId)
	return versionID, nil
}

// startHeartbeat starts a background goroutine that sends heartbeats to keep
// the deployment lock alive. Returns a cancel function to stop it.
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
	var apiErr *apierr.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict && apiErr.ErrorCode == "ABORTED"
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
