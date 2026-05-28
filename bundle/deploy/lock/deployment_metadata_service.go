package lock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/google/uuid"
)

// defaultHeartbeatInterval is how often the background heartbeat goroutine
// renews the DMS-side lock lease while a deployment is in progress.
const defaultHeartbeatInterval = 30 * time.Second

// metadataServiceLock implements DeploymentLock against the bundle deployment
// metadata service (DMS). The lock is acquired by creating a new Version
// under the deployment; a background goroutine renews the lock lease via
// Heartbeat calls; the lock is released by CompleteVersion.
type metadataServiceLock struct {
	b           *bundle.Bundle
	versionType sdkbundle.VersionType

	svc          sdkbundle.BundleInterface
	deploymentID string
	versionID    string

	stopHeartbeat context.CancelFunc
}

func newMetadataServiceLock(b *bundle.Bundle, goal Goal) (*metadataServiceLock, error) {
	versionType, err := goalToVersionType(goal)
	if err != nil {
		return nil, err
	}
	return &metadataServiceLock{b: b, versionType: versionType}, nil
}

// goalToVersionType maps a deployment Goal onto the DMS VersionType enum.
// Bind and Unbind are not yet supported under DMS — they will gain dedicated
// DMS operations in a later change.
func goalToVersionType(goal Goal) (sdkbundle.VersionType, error) {
	switch goal {
	case GoalDeploy:
		return sdkbundle.VersionTypeVersionTypeDeploy, nil
	case GoalDestroy:
		return sdkbundle.VersionTypeVersionTypeDestroy, nil
	case GoalBind, GoalUnbind:
		return "", fmt.Errorf("%s is not supported with the deployment metadata service", goal)
	default:
		return "", fmt.Errorf("unknown deployment goal: %s", goal)
	}
}

func (l *metadataServiceLock) Acquire(ctx context.Context) error {
	if l.b.Config.Bundle.Deployment.Lock.Force {
		return errors.New("force lock is not supported with the deployment metadata service")
	}

	l.svc = l.b.WorkspaceClient(ctx).Bundle

	deploymentID, versionID, err := acquireLock(ctx, l.b, l.svc, l.versionType)
	if err != nil {
		return err
	}

	l.deploymentID = deploymentID
	l.versionID = versionID
	// Publish the deployment ID on the bundle so downstream code (e.g.
	// statemgmt.LoadStateFromDMS) can address the right server-side record.
	l.b.DeploymentID = deploymentID
	l.stopHeartbeat = startHeartbeat(ctx, l.svc, deploymentID, versionID)

	log.Infof(ctx, "Acquired deployment lock: deployment=%s version=%s", deploymentID, versionID)
	return nil
}

func (l *metadataServiceLock) Release(ctx context.Context, status DeploymentStatus) error {
	// Stop the heartbeat first so its in-flight request doesn't race with
	// CompleteVersion below.
	if l.stopHeartbeat != nil {
		l.stopHeartbeat()
	}

	// If Acquire failed before reaching CreateVersion there is nothing to release.
	if l.svc == nil || l.deploymentID == "" || l.versionID == "" {
		return nil
	}

	reason := sdkbundle.VersionCompleteVersionCompleteSuccess
	if status == DeploymentFailure {
		reason = sdkbundle.VersionCompleteVersionCompleteFailure
	}

	versionName := fmt.Sprintf("deployments/%s/versions/%s", l.deploymentID, l.versionID)
	if _, err := l.svc.CompleteVersion(ctx, sdkbundle.CompleteVersionRequest{
		Name:             versionName,
		CompletionReason: reason,
	}); err != nil {
		return err
	}
	log.Infof(ctx, "Released deployment lock: deployment=%s version=%s reason=%s",
		l.deploymentID, l.versionID, reason)

	// On successful destroy, delete the deployment record. Surface failures
	// to the caller — they are deploy-correctness issues, not best-effort
	// cleanup.
	if status == DeploymentSuccess && l.versionType == sdkbundle.VersionTypeVersionTypeDestroy {
		if err := l.svc.DeleteDeployment(ctx, sdkbundle.DeleteDeploymentRequest{
			Name: "deployments/" + l.deploymentID,
		}); err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	}
	return nil
}

// acquireLock implements the lock acquisition protocol:
//  1. Resolve the deployment ID from managed_service.json (or generate a new one).
//  2. CreateDeployment for fresh IDs; GetDeployment otherwise to learn the
//     next version number.
//  3. CreateVersion to acquire the lock.
func acquireLock(ctx context.Context, b *bundle.Bundle, svc sdkbundle.BundleInterface, versionType sdkbundle.VersionType) (deploymentID, versionID string, err error) {
	deploymentID, isNew, err := resolveDeploymentID(ctx, b)
	if err != nil {
		return "", "", err
	}

	if isNew {
		// Fresh deployment: create the record at version 1.
		_, createErr := svc.CreateDeployment(ctx, sdkbundle.CreateDeploymentRequest{
			DeploymentId: deploymentID,
			Deployment: sdkbundle.Deployment{
				TargetName: b.Config.Bundle.Target,
			},
		})
		if createErr != nil {
			return "", "", fmt.Errorf("failed to create deployment: %w", createErr)
		}
		// Persist the deployment ID only after the server-side record exists,
		// so a failed CreateDeployment doesn't leave a dangling ID on disk.
		if err := writeDeploymentID(ctx, b, deploymentID); err != nil {
			return "", "", err
		}
		versionID = "1"
	} else {
		// Existing deployment: ask the server for the last version ID.
		dep, getErr := svc.GetDeployment(ctx, sdkbundle.GetDeploymentRequest{
			Name: "deployments/" + deploymentID,
		})
		if getErr != nil {
			return "", "", fmt.Errorf("failed to get deployment: %w", getErr)
		}
		next, parseErr := nextVersionID(dep.LastVersionId)
		if parseErr != nil {
			return "", "", parseErr
		}
		versionID = next
	}

	if _, err := svc.CreateVersion(ctx, sdkbundle.CreateVersionRequest{
		Parent:    "deployments/" + deploymentID,
		VersionId: versionID,
		Version: sdkbundle.Version{
			CliVersion:  build.GetInfo().Version,
			VersionType: versionType,
			TargetName:  b.Config.Bundle.Target,
		},
	}); err != nil {
		return "", "", fmt.Errorf("failed to acquire deployment lock: %w", err)
	}

	return deploymentID, versionID, nil
}

// nextVersionID returns the next monotonic version ID following lastVersionID.
// An empty lastVersionID means "no prior versions" so the next ID is "1".
func nextVersionID(lastVersionID string) (string, error) {
	if lastVersionID == "" {
		return "1", nil
	}
	n, err := strconv.ParseInt(lastVersionID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse last_version_id %q: %w", lastVersionID, err)
	}
	return strconv.FormatInt(n+1, 10), nil
}

// resolveDeploymentID returns the deployment ID for this bundle. If
// managed_service.json exists in the workspace state directory and contains a
// deployment ID, it is reused. Otherwise a new UUID is generated and the
// caller must write it to disk after CreateDeployment succeeds.
func resolveDeploymentID(ctx context.Context, b *bundle.Bundle) (string, bool, error) {
	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		return "", false, fmt.Errorf("failed to create state filer: %w", err)
	}

	reader, readErr := f.Read(ctx, statemgmt.ManagedServiceFileName)
	if readErr == nil {
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			return "", false, fmt.Errorf("failed to read %s: %w", statemgmt.ManagedServiceFileName, err)
		}
		var sj statemgmt.ManagedServiceJSON
		if err := json.Unmarshal(data, &sj); err != nil {
			return "", false, fmt.Errorf("failed to parse %s: %w", statemgmt.ManagedServiceFileName, err)
		}
		if sj.DeploymentID != "" {
			return sj.DeploymentID, false, nil
		}
		// File exists but has no deployment_id — treat as fresh.
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return "", false, fmt.Errorf("failed to read %s: %w", statemgmt.ManagedServiceFileName, readErr)
	}

	return uuid.New().String(), true, nil
}

func writeDeploymentID(ctx context.Context, b *bundle.Bundle, deploymentID string) error {
	f, err := deploy.StateFiler(ctx, b)
	if err != nil {
		return fmt.Errorf("failed to create state filer: %w", err)
	}
	data, err := json.Marshal(statemgmt.ManagedServiceJSON{DeploymentID: deploymentID})
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", statemgmt.ManagedServiceFileName, err)
	}
	if err := f.Write(ctx, statemgmt.ManagedServiceFileName, bytes.NewReader(data),
		filer.CreateParentDirectories, filer.OverwriteIfExists); err != nil {
		return fmt.Errorf("failed to write %s: %w", statemgmt.ManagedServiceFileName, err)
	}
	return nil
}

// startHeartbeat spawns a goroutine that renews the DMS lock lease at
// defaultHeartbeatInterval. The returned cancel func stops the goroutine.
// Heartbeat errors that indicate the version was already completed (HTTP 409
// ABORTED) are treated as benign termination; all other errors are logged
// and the goroutine continues so a transient network blip doesn't tear down
// the deploy.
func startHeartbeat(parent context.Context, svc sdkbundle.BundleInterface, deploymentID, versionID string) context.CancelFunc {
	ctx, cancel := context.WithCancel(parent)
	versionName := fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID)

	go func() {
		ticker := time.NewTicker(defaultHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := svc.Heartbeat(ctx, sdkbundle.HeartbeatRequest{Name: versionName}); err != nil {
					if isAborted(err) {
						log.Debugf(ctx, "Heartbeat stopped: version already completed")
						return
					}
					log.Warnf(ctx, "Failed to send deployment heartbeat: %v", err)
					continue
				}
				log.Debugf(ctx, "Deployment heartbeat sent for deployment=%s version=%s",
					deploymentID, versionID)
			}
		}
	}()

	return cancel
}

// isAborted reports whether err is the DMS-specific "409 ABORTED" response
// the server emits when the heartbeat target version is no longer active.
func isAborted(err error) bool {
	var apiErr *apierr.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict && apiErr.ErrorCode == "ABORTED" {
		return true
	}
	return false
}
