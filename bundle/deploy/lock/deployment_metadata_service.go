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
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/tmpdms"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/google/uuid"
)

const defaultHeartbeatInterval = 30 * time.Second

type metadataServiceLock struct {
	b           *bundle.Bundle
	versionType tmpdms.VersionType

	svc       *tmpdms.DeploymentMetadataAPI
	versionID string

	stopHeartbeat func()
	reporter      *asyncReporter
}

func newMetadataServiceLock(b *bundle.Bundle, versionType tmpdms.VersionType) *metadataServiceLock {
	return &metadataServiceLock{b: b, versionType: versionType}
}

func (l *metadataServiceLock) Acquire(ctx context.Context) error {
	if l.b.Config.Bundle.Deployment.Lock.Force {
		return errors.New("force lock is not supported with the deployment metadata service")
	}

	svc, err := tmpdms.NewDeploymentMetadataAPI(l.b.WorkspaceClient())
	if err != nil {
		return fmt.Errorf("failed to create metadata service client: %w", err)
	}
	l.svc = svc

	deploymentID, versionID, err := acquireLock(ctx, l.b, svc, l.versionType)
	if err != nil {
		return err
	}

	l.b.DeploymentID = deploymentID
	l.versionID = versionID
	l.stopHeartbeat = startHeartbeat(ctx, svc, deploymentID, versionID)
	l.reporter = newAsyncReporter(ctx, makeSyncReporter(svc, deploymentID, versionID))
	l.b.DeploymentBundle.OperationReporter = l.reporter.Reporter()
	return nil
}

func (l *metadataServiceLock) Release(ctx context.Context, status DeploymentStatus) error {
	if l.reporter != nil {
		l.reporter.Close()
	}
	if l.stopHeartbeat != nil {
		l.stopHeartbeat()
	}

	reason := tmpdms.VersionCompleteSuccess
	if status == DeploymentFailure {
		reason = tmpdms.VersionCompleteFailure
	}

	_, completeErr := l.svc.CompleteVersion(ctx, tmpdms.CompleteVersionRequest{
		DeploymentID:     l.b.DeploymentID,
		VersionID:        l.versionID,
		Name:             fmt.Sprintf("deployments/%s/versions/%s", l.b.DeploymentID, l.versionID),
		CompletionReason: reason,
	})
	if completeErr != nil {
		return completeErr
	}
	log.Infof(ctx, "Released deployment lock: deployment=%s version=%s reason=%s", l.b.DeploymentID, l.versionID, reason)

	// For destroy operations, delete the deployment record after
	// successfully releasing the lock.
	if status == DeploymentSuccess && l.versionType == tmpdms.VersionTypeDestroy {
		_, deleteErr := l.svc.DeleteDeployment(ctx, tmpdms.DeleteDeploymentRequest{
			DeploymentID: l.b.DeploymentID,
		})
		if deleteErr != nil {
			return fmt.Errorf("failed to delete deployment: %w", deleteErr)
		}
	}

	return nil
}

// acquireLock implements the lock acquisition protocol using the deployment
// metadata service: resolve deployment ID, ensure deployment, create version.
func acquireLock(ctx context.Context, b *bundle.Bundle, svc *tmpdms.DeploymentMetadataAPI, versionType tmpdms.VersionType) (deploymentID, versionID string, err error) {
	var isNew bool
	deploymentID, isNew, err = resolveDeploymentID(ctx, b)
	if err != nil {
		return "", "", err
	}

	if isNew {
		// Fresh deployment: create the record and start at version 1.
		_, createErr := svc.CreateDeployment(ctx, tmpdms.CreateDeploymentRequest{
			DeploymentID: deploymentID,
			Deployment: &tmpdms.Deployment{
				DisplayName: b.Config.Bundle.Name,
				TargetName:  b.Config.Bundle.Target,
			},
		})
		if createErr != nil {
			return "", "", fmt.Errorf("failed to create deployment: %w", createErr)
		}
		// Write the deployment ID to workspace only after the server-side
		// record is created. This avoids leaving a dangling ID if creation fails.
		if err := writeDeploymentID(ctx, b, deploymentID); err != nil {
			return "", "", err
		}
		versionID = "1"
	} else {
		// Existing deployment: get the last version ID to determine the next one.
		dep, getErr := svc.GetDeployment(ctx, tmpdms.GetDeploymentRequest{
			DeploymentID: deploymentID,
		})
		if getErr != nil {
			return "", "", fmt.Errorf("failed to get deployment: %w", getErr)
		}
		lastVersion, parseErr := strconv.ParseInt(dep.LastVersionID, 10, 64)
		if parseErr != nil {
			return "", "", fmt.Errorf("failed to parse last_version_id %q: %w", dep.LastVersionID, parseErr)
		}
		versionID = strconv.FormatInt(lastVersion+1, 10)
	}

	// Create a version to acquire the deployment lock.
	version, versionErr := svc.CreateVersion(ctx, tmpdms.CreateVersionRequest{
		DeploymentID: deploymentID,
		Parent:       "deployments/" + deploymentID,
		VersionID:    versionID,
		Version: &tmpdms.Version{
			CliVersion:  build.GetInfo().Version,
			VersionType: versionType,
			TargetName:  b.Config.Bundle.Target,
		},
	})
	if versionErr != nil {
		return "", "", fmt.Errorf("failed to acquire deployment lock: %w", versionErr)
	}

	log.Infof(ctx, "Acquired deployment lock: deployment=%s version=%s", deploymentID, version.VersionID)
	return deploymentID, versionID, nil
}

// resolveDeploymentID reads the deployment ID from managed_service.json in the
// workspace state directory. If the file doesn't exist or has no deployment ID,
// a new UUID is generated. The boolean return indicates whether this is a fresh
// deployment (true) or an existing one (false). For fresh deployments, the
// caller is responsible for writing the deployment ID to workspace after the
// server-side deployment record is created successfully.
func resolveDeploymentID(ctx context.Context, b *bundle.Bundle) (string, bool, error) {
	f, err := deploy.StateFiler(b)
	if err != nil {
		return "", false, fmt.Errorf("failed to create state filer: %w", err)
	}

	// Try reading existing deployment ID from managed_service.json.
	reader, readErr := f.Read(ctx, statemgmt.ManagedServiceFileName)
	if readErr == nil {
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			return "", false, fmt.Errorf("failed to read %s content: %w", statemgmt.ManagedServiceFileName, err)
		}
		var sj statemgmt.ManagedServiceJSON
		if err := json.Unmarshal(data, &sj); err != nil {
			return "", false, fmt.Errorf("failed to parse %s: %w", statemgmt.ManagedServiceFileName, err)
		}
		if sj.DeploymentID != "" {
			return sj.DeploymentID, false, nil
		}
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return "", false, fmt.Errorf("failed to read %s: %w", statemgmt.ManagedServiceFileName, readErr)
	}

	// Fresh deployment: generate a new ID but don't write yet.
	return uuid.New().String(), true, nil
}

// writeDeploymentID writes the deployment ID to managed_service.json in the
// workspace state directory. This should be called after the server-side
// deployment record is created successfully.
func writeDeploymentID(ctx context.Context, b *bundle.Bundle, deploymentID string) error {
	f, err := deploy.StateFiler(b)
	if err != nil {
		return fmt.Errorf("failed to create state filer: %w", err)
	}
	sj := statemgmt.ManagedServiceJSON{DeploymentID: deploymentID}
	data, err := json.Marshal(sj)
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", statemgmt.ManagedServiceFileName, err)
	}
	err = f.Write(ctx, statemgmt.ManagedServiceFileName, bytes.NewReader(data), filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", statemgmt.ManagedServiceFileName, err)
	}
	return nil
}

// makeSyncReporter returns the synchronous "send one event to DMS" function
// consumed by asyncReporter's sender goroutine. Skip-actions short-circuit to
// nil; mapping errors and API errors are returned to the caller (which logs
// and continues — see asyncReporter).
func makeSyncReporter(svc *tmpdms.DeploymentMetadataAPI, deploymentID, versionID string) func(context.Context, operationEvent) error {
	return func(ctx context.Context, ev operationEvent) error {
		// The internal state DB uses "resources.jobs.foo" keys but the API
		// expects "jobs.foo" — strip the "resources." prefix.
		apiKey := strings.TrimPrefix(ev.resourceKey, "resources.")
		actionType, err := planActionToOperationAction(ev.action)
		if err != nil {
			return fmt.Errorf("mapping action for resource %s: %w", ev.resourceKey, err)
		}
		if actionType == "" {
			return nil
		}

		status := tmpdms.OperationStatusSucceeded
		var errorMessage string
		if ev.operationErr != nil {
			status = tmpdms.OperationStatusFailed
			errorMessage = ev.operationErr.Error()
		}

		op := &tmpdms.Operation{
			ResourceKey:  apiKey,
			ResourceID:   ev.resourceID,
			Status:       status,
			ActionType:   actionType,
			ErrorMessage: errorMessage,
		}
		if len(ev.state) > 0 {
			op.State = ev.state
		}

		_, err = svc.CreateOperation(ctx, tmpdms.CreateOperationRequest{
			DeploymentID: deploymentID,
			VersionID:    versionID,
			Parent:       fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID),
			ResourceKey:  apiKey,
			Operation:    op,
		})
		if err != nil {
			return fmt.Errorf("reporting operation for resource %s: %w", ev.resourceKey, err)
		}
		return nil
	}
}


// planActionToOperationAction maps a deploy plan action to a metadata service
// operation action type. No-op actions like Skip return ("", nil) and should
// be ignored.
func planActionToOperationAction(action deployplan.ActionType) (tmpdms.OperationActionType, error) {
	switch action {
	case deployplan.Skip:
		return "", nil
	case deployplan.Create:
		return tmpdms.OperationActionTypeCreate, nil
	case deployplan.Update:
		return tmpdms.OperationActionTypeUpdate, nil
	case deployplan.UpdateWithID:
		return tmpdms.OperationActionTypeUpdateWithID, nil
	case deployplan.Delete:
		return tmpdms.OperationActionTypeDelete, nil
	case deployplan.Recreate:
		return tmpdms.OperationActionTypeRecreate, nil
	case deployplan.Resize:
		return tmpdms.OperationActionTypeResize, nil
	default:
		return "", fmt.Errorf("unsupported operation action type: %s", action)
	}
}

// startHeartbeat starts a background goroutine that sends heartbeats to keep
// the deployment lock alive. Returns a cancel function to stop the heartbeat.
func startHeartbeat(ctx context.Context, svc *tmpdms.DeploymentMetadataAPI, deploymentID, versionID string) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(defaultHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := svc.Heartbeat(ctx, tmpdms.HeartbeatRequest{
					DeploymentID: deploymentID,
					VersionID:    versionID,
				})
				if err != nil {
					// A 409 ABORTED is expected if the version was completed
					// between the ticker firing and the heartbeat request.
					if isAborted(err) {
						log.Debugf(ctx, "Heartbeat stopped: version already completed")
						return
					}
					log.Warnf(ctx, "Failed to send deployment heartbeat: %v", err)
				} else {
					log.Debugf(ctx, "Deployment heartbeat sent for deployment=%s version=%s", deploymentID, versionID)
				}
			}
		}
	}()

	return cancel
}

// isAborted checks if an error indicates the operation was aborted (HTTP 409 with ABORTED error code).
func isAborted(err error) bool {
	var apiErr *apierr.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict && apiErr.ErrorCode == "ABORTED" {
		return true
	}
	return false
}
