package dms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

// Recorder records a single deploy/destroy as a version with DMS.
//
// The server assigns the deployment ID on the first deploy, i.e. when the ID
// resolved from the workspace is empty (see ResolveDeploymentID). Later deploys
// resolve the same ID and reuse the record; a destroy deletes the record and its
// node, so the next deploy starts over from empty.
type Recorder struct {
	svc          bundledeployments.BundleDeploymentsInterface
	deploymentID string
	statePath    string
	targetName   string
	versionType  VersionType
	gitInfo      *bundledeployments.GitInfo

	// populated by CreateVersion
	versionNum    int64
	stopHeartbeat context.CancelFunc
}

// NewRecorder returns a Recorder for the given deployment. deploymentID is the
// ID resolved from the deployment's workspace node, or empty if this bundle has
// not yet recorded a deployment (the server assigns one during CreateVersion).
// statePath is the bundle's remote state directory, under which DMS registers
// the deployment node. gitInfo is the source's git origin/branch/commit recorded
// on each version, or nil when the bundle is not in a git repository.
func NewRecorder(svc bundledeployments.BundleDeploymentsInterface, deploymentID, statePath, targetName string, versionType VersionType, gitInfo *bundledeployments.GitInfo) *Recorder {
	return &Recorder{
		svc:          svc,
		deploymentID: deploymentID,
		statePath:    statePath,
		targetName:   targetName,
		versionType:  versionType,
		gitInfo:      gitInfo,
	}
}

// DeploymentID returns the DMS deployment ID this recorder is bound to. It is
// empty until CreateVersion has created the deployment record (on a first
// deploy) and non-empty afterwards, so callers can parent operations under it.
func (r *Recorder) DeploymentID() string {
	if r == nil {
		return ""
	}
	return r.deploymentID
}

// Version returns the version number claimed by CreateVersion. It is zero until
// CreateVersion has run; callers use it to parent operations under the version.
func (r *Recorder) Version() int64 {
	if r == nil {
		return 0
	}
	return r.versionNum
}

// CreateVersion registers a new version with DMS, claiming it for the duration
// of the deployment. A nil Recorder is a no-op, so callers can leave it nil
// when recording is disabled.
func (r *Recorder) CreateVersion(ctx context.Context) error {
	if r == nil {
		return nil
	}

	versionID, err := r.createDeploymentVersion(ctx)
	if err != nil {
		return err
	}

	versionNum, err := strconv.ParseInt(versionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse version ID %q: %w", versionID, err)
	}
	r.versionNum = versionNum
	r.stopHeartbeat = startHeartbeat(ctx, r.svc, r.deploymentID, versionID)
	return nil
}

// CompleteVersion finalizes the version created by CreateVersion. A nil
// Recorder, or one whose CreateVersion never ran or failed, is a no-op: there is
// no version on the server to complete. Callers defer it unconditionally, so this
// is the check that keeps a cancelled or failed deploy from completing a version
// that was never created.
func (r *Recorder) CompleteVersion(ctx context.Context, success bool) error {
	if r == nil || r.versionNum == 0 {
		return nil
	}

	r.stopHeartbeat()

	versionIDStr := strconv.FormatInt(r.versionNum, 10)
	versionName := fmt.Sprintf("deployments/%s/versions/%s", r.deploymentID, versionIDStr)

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
	log.Infof(ctx, "Completed deployment version: deployment=%s version=%s reason=%s", r.deploymentID, versionIDStr, reason)

	// For destroy operations, delete the deployment record after the version
	// completes successfully.
	if success && r.versionType == VersionTypeDestroy {
		err = r.svc.DeleteDeployment(ctx, bundledeployments.DeleteDeploymentRequest{
			Name: "deployments/" + r.deploymentID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	}

	return nil
}

// createDeploymentVersion ensures the deployment record exists, then creates a
// new version under it. With no deployment ID it creates the deployment and lets
// the server assign the ID; otherwise it reads the existing deployment to
// compute the next version number.
func (r *Recorder) createDeploymentVersion(ctx context.Context) (versionID string, err error) {
	// previousVersionID is the deployment's current most-recent version, which the
	// server requires for optimistic concurrency (see CreateVersion below). It
	// stays empty for the deployment's first version, which has no predecessor.
	var previousVersionID string
	if r.deploymentID != "" {
		// A resolved node names the deployment, but its record is created by the
		// first version, so there may be none yet: a deploy that registered the
		// deployment and then failed before recording a version. Start at version 1
		// under the ID the node already names, rather than creating a second
		// deployment, which would collide on the same node path.
		dep, getErr := r.svc.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
			Name: "deployments/" + r.deploymentID,
		})
		switch {
		case getErr == nil:
			lastVersion, parseErr := strconv.ParseInt(dep.LastVersionId, 10, 64)
			if parseErr != nil {
				return "", fmt.Errorf("failed to parse last_version_id %q: %w", dep.LastVersionId, parseErr)
			}
			previousVersionID = dep.LastVersionId
			versionID = strconv.FormatInt(lastVersion+1, 10)
		case errors.Is(getErr, apierr.ErrNotFound), errors.Is(getErr, apierr.ErrResourceDoesNotExist):
			versionID = "1"
		default:
			return "", fmt.Errorf("failed to get deployment: %w", getErr)
		}
	} else {
		// First deploy: create the deployment so the server assigns an ID.
		//
		// initial_parent_path is required. The service creates the deployment node
		// under it, and that node's ID is the deployment ID ResolveDeploymentID reads
		// back later. The folder already exists by now: the deployment lock lives in
		// the same directory.
		dep, createErr := r.svc.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{
			Deployment: bundledeployments.Deployment{
				InitialParentPath: r.statePath,
				TargetName:        r.targetName,
			},
		})
		if createErr != nil {
			return "", fmt.Errorf("failed to create deployment: %w", createErr)
		}
		id, idErr := deploymentIDFromName(dep.Name)
		if idErr != nil {
			return "", idErr
		}
		r.deploymentID = id
		versionID = "1"
	}

	// The server validates that previous_version_id matches the deployment's
	// current most-recent version and returns INVALID_PARAMETER_VALUE otherwise
	// (e.g. a concurrent deploy already recorded a newer version). It is empty for
	// the first version, which has no predecessor.
	version, versionErr := r.svc.CreateVersion(ctx, bundledeployments.CreateVersionRequest{
		Parent:    "deployments/" + r.deploymentID,
		VersionId: versionID,
		Version: bundledeployments.Version{
			CliVersion:        build.GetInfo().Version,
			VersionType:       r.versionType,
			TargetName:        r.targetName,
			PreviousVersionId: previousVersionID,
			GitInfo:           r.gitInfo,
		},
	})
	if versionErr != nil {
		return "", fmt.Errorf("failed to create deployment version: %w", versionErr)
	}

	log.Infof(ctx, "Created deployment version: deployment=%s version=%s", r.deploymentID, version.VersionId)
	return versionID, nil
}

// deploymentIDFromName extracts the deployment ID from a DMS resource name of
// the form "deployments/{deployment_id}".
func deploymentIDFromName(name string) (string, error) {
	id, ok := strings.CutPrefix(name, "deployments/")
	if !ok || id == "" {
		return "", fmt.Errorf("unexpected deployment name %q from deployment metadata service", name)
	}
	return id, nil
}

// startHeartbeat starts a background goroutine that sends heartbeats to keep
// the deployment version's lease alive. Returns a cancel function to stop it.
func startHeartbeat(ctx context.Context, svc bundledeployments.BundleDeploymentsInterface, deploymentID, versionID string) context.CancelFunc {
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
