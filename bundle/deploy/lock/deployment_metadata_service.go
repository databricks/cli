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
	"github.com/databricks/cli/bundle/direct"
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
	l.b.DeploymentBundle.OperationReporter = makeOperationReporter(svc, deploymentID, versionID)
	return nil
}

func (l *metadataServiceLock) Release(ctx context.Context, status DeploymentStatus) error {
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

	// Only create the deployment record for fresh deployments.
	if isNew {
		_, createErr := svc.CreateDeployment(ctx, tmpdms.CreateDeploymentRequest{
			DeploymentID: deploymentID,
			Deployment: &tmpdms.Deployment{
				TargetName: b.Config.Bundle.Target,
			},
		})
		if createErr != nil {
			return "", "", fmt.Errorf("failed to create deployment: %w", createErr)
		}
	}

	// Get the deployment to determine the next version ID.
	dep, getErr := svc.GetDeployment(ctx, tmpdms.GetDeploymentRequest{
		DeploymentID: deploymentID,
	})
	if getErr != nil {
		return "", "", fmt.Errorf("failed to get deployment: %w", getErr)
	}

	if dep.LastVersionID == "" {
		versionID = "1"
	} else {
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

// resolveDeploymentID reads the deployment ID from resources.json in the
// workspace state directory. If the file doesn't exist or has no deployment ID,
// a new UUID is generated and written. The boolean return indicates whether
// this is a fresh deployment (true) or an existing one (false).
func resolveDeploymentID(ctx context.Context, b *bundle.Bundle) (string, bool, error) {
	f, err := deploy.StateFiler(b)
	if err != nil {
		return "", false, fmt.Errorf("failed to create state filer: %w", err)
	}

	// Try reading existing deployment ID from resources.json.
	reader, readErr := f.Read(ctx, "resources.json")
	if readErr == nil {
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			return "", false, fmt.Errorf("failed to read resources.json content: %w", err)
		}
		var rj statemgmt.ResourcesJSON
		if err := json.Unmarshal(data, &rj); err != nil {
			return "", false, fmt.Errorf("failed to parse resources.json: %w", err)
		}
		if rj.DeploymentID != "" {
			return rj.DeploymentID, false, nil
		}
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return "", false, fmt.Errorf("failed to read resources.json: %w", readErr)
	}

	// Fresh deployment: generate a new ID and write resources.json.
	deploymentID := uuid.New().String()
	rj := statemgmt.ResourcesJSON{DeploymentID: deploymentID}
	data, err := json.Marshal(rj)
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal resources.json: %w", err)
	}
	err = f.Write(ctx, "resources.json", bytes.NewReader(data), filer.CreateParentDirectories, filer.OverwriteIfExists)
	if err != nil {
		return "", false, fmt.Errorf("failed to write resources.json: %w", err)
	}
	return deploymentID, true, nil
}

// makeOperationReporter returns an OperationReporter that reports each resource
// operation (success or failure) to the deployment metadata service.
func makeOperationReporter(svc *tmpdms.DeploymentMetadataAPI, deploymentID, versionID string) direct.OperationReporter {
	return func(
		ctx context.Context,
		resourceKey string,
		resourceID string,
		action deployplan.ActionType,
		operationErr error,
		state json.RawMessage,
	) error {
		// The internal state DB uses "resources.jobs.foo" keys but the API
		// expects "jobs.foo" — strip the "resources." prefix.
		apiKey := strings.TrimPrefix(resourceKey, "resources.")
		actionType, err := planActionToOperationAction(action)
		if err != nil {
			return fmt.Errorf("mapping action for resource %s: %w", resourceKey, err)
		}
		if actionType == "" {
			return nil
		}

		status := tmpdms.OperationStatusSucceeded
		var errorMessage string
		if operationErr != nil {
			status = tmpdms.OperationStatusFailed
			errorMessage = operationErr.Error()
		}

		op := &tmpdms.Operation{
			ResourceKey:  apiKey,
			ResourceID:   resourceID,
			Status:       status,
			ActionType:   actionType,
			ErrorMessage: errorMessage,
		}
		if len(state) > 0 {
			op.State = state
		}

		_, err = svc.CreateOperation(ctx, tmpdms.CreateOperationRequest{
			DeploymentID: deploymentID,
			VersionID:    versionID,
			Parent:       fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID),
			ResourceKey:  apiKey,
			Operation:    op,
		})
		if err != nil {
			return fmt.Errorf("reporting operation for resource %s: %w", resourceKey, err)
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
