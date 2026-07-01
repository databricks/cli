// Package dms records bundle deployments as versions with the Deployment
// Metadata Service (DMS).
//
// It is intentionally independent of the deployment lock: a Recorder does not
// acquire or hold any lock. Callers serialize concurrent deployments (today via
// the workspace-filesystem lock). The DMS deployment is identified by the
// bundle's state lineage and each version by the state serial; both are read
// from the plan (the single source of truth) and passed to CreateVersion /
// CompleteVersion, so the recorder never derives them itself.
package dms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// The server expires a version's lease if it does not receive a heartbeat
// within a 2-minute TTL; we heartbeat well inside that window.
const defaultHeartbeatInterval = 30 * time.Second

// VersionType identifies the kind of deployment a version records.
type VersionType = bundledeployments.VersionType

const (
	VersionTypeDeploy  VersionType = bundledeployments.VersionTypeVersionTypeDeploy
	VersionTypeDestroy VersionType = bundledeployments.VersionTypeVersionTypeDestroy
)

// Recorder records a single deploy/destroy as a version with DMS. The lineage
// and serial that identify the deployment and version are passed in from the
// plan, not stored here.
type Recorder struct {
	svc         bundledeployments.BundleDeploymentsInterface
	targetName  string
	versionType VersionType

	// stopHeartbeat stops the heartbeat goroutine started by CreateVersion. It
	// is nil until CreateVersion runs, which is how CompleteVersion knows whether
	// there is anything to finalize.
	stopHeartbeat context.CancelFunc
}

// NewRecorder returns a Recorder for the given deployment target.
func NewRecorder(svc bundledeployments.BundleDeploymentsInterface, targetName string, versionType VersionType) *Recorder {
	return &Recorder{
		svc:         svc,
		targetName:  targetName,
		versionType: versionType,
	}
}

// CreateVersion registers a new version with DMS, claiming it for the duration
// of the deployment. The lineage identifies the deployment and serial is the
// plan's state serial; both come from the plan. A nil Recorder is a no-op, so
// callers can leave it nil when recording is disabled.
func (r *Recorder) CreateVersion(ctx context.Context, lineage string, serial int) error {
	if r == nil {
		return nil
	}

	if err := r.ensureDeployment(ctx, lineage); err != nil {
		return err
	}

	id := strconv.Itoa(serial)
	// TODO: once the SDK exposes previous_version_id (universe #2061768), set it
	// to the serial the plan was computed against so the server can reject a
	// deployment built on a stale plan (serializability). It is required unless
	// the deployment is net new, in which case it is left unset.
	version, err := r.svc.CreateVersion(ctx, bundledeployments.CreateVersionRequest{
		Parent:    "deployments/" + lineage,
		VersionId: id,
		Version: bundledeployments.Version{
			CliVersion:  build.GetInfo().Version,
			VersionType: r.versionType,
			TargetName:  r.targetName,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment version: %w", err)
	}

	log.Infof(ctx, "Created deployment version: deployment=%s version=%s", lineage, version.VersionId)
	r.stopHeartbeat = startHeartbeat(ctx, r.svc, lineage, id)
	return nil
}

// ensureDeployment creates the DMS deployment record if it does not exist yet.
// GetDeployment is the only thing read from the backend: it tells us whether
// this lineage has been recorded before (recording may have been enabled after
// the first deploy). The version ID itself comes from the plan, not from the
// deployment's last_version_id.
func (r *Recorder) ensureDeployment(ctx context.Context, lineage string) error {
	_, err := r.svc.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
		Name: "deployments/" + lineage,
	})
	if errors.Is(err, apierr.ErrNotFound) {
		_, err = r.svc.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{
			DeploymentId: lineage,
			Deployment: bundledeployments.Deployment{
				TargetName: r.targetName,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}
	return nil
}

// CompleteVersion finalizes the version created by CreateVersion. The lineage
// and serial must be the same ones passed to CreateVersion (both come from the
// same plan). A nil Recorder, or one whose CreateVersion never ran, is a no-op.
func (r *Recorder) CompleteVersion(ctx context.Context, lineage string, serial int, success bool) error {
	if r == nil || r.stopHeartbeat == nil {
		return nil
	}

	r.stopHeartbeat()

	id := strconv.Itoa(serial)
	versionName := fmt.Sprintf("deployments/%s/versions/%s", lineage, id)

	reason := bundledeployments.VersionCompleteVersionCompleteSuccess
	if !success {
		reason = bundledeployments.VersionCompleteVersionCompleteFailure
	}

	_, err := r.svc.CompleteVersion(ctx, bundledeployments.CompleteVersionRequest{
		Name:             versionName,
		CompletionReason: reason,
	})
	if err != nil {
		return err
	}
	log.Infof(ctx, "Completed deployment version: deployment=%s version=%s reason=%s", lineage, id, reason)

	// For destroy operations, delete the deployment record after the version
	// completes successfully.
	if success && r.versionType == VersionTypeDestroy {
		err = r.svc.DeleteDeployment(ctx, bundledeployments.DeleteDeploymentRequest{
			Name: "deployments/" + lineage,
		})
		if err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	}

	return nil
}

// startHeartbeat starts a background goroutine that sends heartbeats to keep
// the deployment version's lease alive. Returns a cancel function to stop it.
func startHeartbeat(ctx context.Context, svc bundledeployments.BundleDeploymentsInterface, lineage, versionID string) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	versionName := fmt.Sprintf("deployments/%s/versions/%s", lineage, versionID)

	go func() {
		ticker := time.NewTicker(defaultHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := svc.Heartbeat(ctx, bundledeployments.HeartbeatRequest{Name: versionName})
				if err != nil {
					// A 409 ABORTED is expected if the version was completed
					// between the ticker firing and the heartbeat.
					if isAbortedErr(err) {
						log.Debugf(ctx, "Heartbeat stopped: version already completed")
						return
					}
					log.Warnf(ctx, "Failed to send deployment heartbeat: %v", err)
				} else {
					log.Debugf(ctx, "Deployment heartbeat sent: deployment=%s version=%s", lineage, versionID)
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
