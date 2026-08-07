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
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
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

// createVersionRequest is the CreateVersion request body.
//
// The CLI builds the body itself instead of using bundledeployments.Version
// because the generated struct has no previous_version_id field, which the
// service requires as its concurrency check. Without it every deploy after the
// first is rejected.
type createVersionRequest struct {
	CliVersion  string      `json:"cli_version"`
	VersionType VersionType `json:"version_type"`
	TargetName  string      `json:"target_name,omitempty"`
	// DisplayName names the deployment in the UI. The service copies it onto the
	// deployment's workspace node, which is where GetDeployment reads it from, so
	// a version that omits it leaves the deployment unnamed.
	DisplayName string `json:"display_name,omitempty"`
	// PreviousVersionId is the deployment's most recent version, unset for a
	// deployment's first version.
	PreviousVersionId string `json:"previous_version_id,omitempty"`
	// DeploymentMode is the bundle target's mode, unset when the target sets none.
	DeploymentMode bundledeployments.DeploymentMode `json:"deployment_mode,omitempty"`
	// GitInfo and WorkspaceInfo record where the deployed source came from and
	// where it landed. The service denormalizes both onto the deployment.
	GitInfo       *bundledeployments.GitInfo       `json:"git_info,omitempty"`
	WorkspaceInfo *bundledeployments.WorkspaceInfo `json:"workspace_info,omitempty"`
}

// versionCreator creates a version under a deployment. It exists because the
// generated client cannot express the request body (see createVersionRequest).
type versionCreator interface {
	CreateVersion(ctx context.Context, deploymentID, versionID string, body createVersionRequest) (*bundledeployments.Version, error)
}

// apiVersionCreator creates versions through the workspace API client.
type apiVersionCreator struct {
	client *client.DatabricksClient
}

// NewAPIVersionCreator returns a versionCreator that posts to the DMS API.
func NewAPIVersionCreator(c *client.DatabricksClient) versionCreator {
	return &apiVersionCreator{client: c}
}

func (a *apiVersionCreator) CreateVersion(ctx context.Context, deploymentID, versionID string, body createVersionRequest) (*bundledeployments.Version, error) {
	var version bundledeployments.Version
	path := fmt.Sprintf("/api/2.0/bundle/deployments/%s/versions", deploymentID)
	err := a.client.Do(ctx, http.MethodPost, path,
		auth.WorkspaceIDHeaders(a.client.Config),
		map[string]any{"version_id": versionID},
		body, &version)
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// Recorder records a single deploy/destroy as a version with DMS.
//
// The server assigns the deployment ID on the first deploy, i.e. when the ID
// resolved from the workspace is empty (see ResolveDeploymentID). Later deploys
// resolve the same ID and reuse the record; a destroy deletes the record and its
// node, so the next deploy starts over from empty.
type Recorder struct {
	svc          bundledeployments.BundleDeploymentsInterface
	versions     versionCreator
	deploymentID string
	statePath    string
	versionType  VersionType
	metadata     Metadata

	// populated by CreateVersion
	versionNum    int64
	stopHeartbeat context.CancelFunc

	// completed makes CompleteVersion idempotent, so a caller that completes the
	// version early can still defer it unconditionally.
	completed bool
}

// RecorderOptions are the dependencies and deployment identity a Recorder needs.
type RecorderOptions struct {
	// Service handles every DMS call except CreateVersion.
	Service bundledeployments.BundleDeploymentsInterface
	// Versions handles CreateVersion; see versionCreator.
	Versions versionCreator
	// DeploymentID is the ID resolved from the deployment's workspace node, or
	// empty if this bundle has not recorded a deployment yet (the server assigns
	// one during CreateVersion).
	DeploymentID string
	// StatePath is the bundle's remote state directory, under which DMS registers
	// the deployment node.
	StatePath   string
	VersionType VersionType
	// Metadata is what the version records about the deploy; see Metadata.
	Metadata Metadata
}

// Metadata is what a version records about the bundle it deployed, the source it
// came from, and where it landed. The service denormalizes these onto the
// deployment, so they describe the deployment as of its most recent version.
type Metadata struct {
	// DisplayName is the bundle's name, which the deployment is listed under.
	DisplayName string
	// TargetName is the bundle target that was deployed.
	TargetName string
	// Mode is the bundle target's mode, empty when the target sets none.
	Mode      bundledeployments.DeploymentMode
	Git       *bundledeployments.GitInfo
	Workspace *bundledeployments.WorkspaceInfo
}

// NewRecorder returns a Recorder for the deployment described by opts.
func NewRecorder(opts RecorderOptions) *Recorder {
	return &Recorder{
		svc:          opts.Service,
		versions:     opts.Versions,
		deploymentID: opts.DeploymentID,
		statePath:    opts.StatePath,
		versionType:  opts.VersionType,
		metadata:     opts.Metadata,
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
	if r == nil || r.versionNum == 0 || r.completed {
		return nil
	}
	r.completed = true

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
	// The version this one supersedes, sent as the concurrency check. Empty for a
	// deployment's first version.
	var previousVersionID string
	if r.deploymentID != "" {
		// The ID came from a BUNDLE_DEPLOYMENT node that get-status returned, and by
		// design the service has a deployment for every such node, so a not-found
		// here means that invariant is broken rather than anything the user did.
		dep, getErr := r.svc.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
			Name: "deployments/" + r.deploymentID,
		})
		switch {
		case errors.Is(getErr, apierr.ErrNotFound), errors.Is(getErr, apierr.ErrResourceDoesNotExist):
			return "", fmt.Errorf("internal error: no deployment found for the file with object id %s: %w", r.deploymentID, getErr)
		case getErr != nil:
			return "", fmt.Errorf("failed to get deployment: %w", getErr)
		case dep.LastVersionId == "":
			// The record exists but carries no version: a deploy whose first version was
			// rejected still leaves the record behind. Retry at version 1.
			versionID = "1"
		default:
			lastVersion, parseErr := strconv.ParseInt(dep.LastVersionId, 10, 64)
			if parseErr != nil {
				return "", fmt.Errorf("failed to parse last_version_id %q: %w", dep.LastVersionId, parseErr)
			}
			versionID = strconv.FormatInt(lastVersion+1, 10)
			previousVersionID = dep.LastVersionId
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
				TargetName:        r.metadata.TargetName,
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

	// The server rejects the call unless versionID is numerically greater than
	// last_version_id and previous_version_id matches it, so a deploy racing
	// another is rejected rather than overwriting it.
	version, versionErr := r.versions.CreateVersion(ctx, r.deploymentID, versionID, createVersionRequest{
		CliVersion:        build.GetInfo().Version,
		VersionType:       r.versionType,
		TargetName:        r.metadata.TargetName,
		DisplayName:       r.metadata.DisplayName,
		PreviousVersionId: previousVersionID,
		DeploymentMode:    r.metadata.Mode,
		GitInfo:           r.metadata.Git,
		WorkspaceInfo:     r.metadata.Workspace,
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
