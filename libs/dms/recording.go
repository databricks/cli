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

// Recording is what one deploy or destroy records with DMS: a version, the operations it
// stages, and their outcomes. A disabled recording is a no-op throughout, so callers do not
// branch on whether recording is on.
type Recording interface {
	// Prepare settles the deployment and the version number this run will create, without
	// creating it. Both are needed before the plan, which the version number is stamped onto.
	Prepare(ctx context.Context) error

	// DeploymentID is the deployment being recorded under. Empty until Prepare, which
	// creates the deployment on a first deploy.
	DeploymentID() string

	// Version is the version number Prepare claimed, and zero before it runs.
	Version() int64

	// Start creates the version, staging an operation for each resource, and returns the sink
	// that fills them in - nil when nothing is recorded, which is what leaves the state DB
	// without one. The staged set is fixed here: the service has no call to add one later, so
	// a resource left out can never be recorded.
	Start(ctx context.Context, staged []StagedOperation) (*OperationSink, error)

	// Finish completes the version. It is a no-op before Start, which is what lets a caller
	// defer it and still not complete a version a cancelled deploy never created, and it is
	// safe to call twice.
	Finish(ctx context.Context, success bool) error
}

// RecordingOptions are the dependencies and deployment identity a Recording needs.
type RecordingOptions struct {
	Client *Client

	// DeploymentID is resolved from the deployment's workspace node, empty until the
	// first recorded deploy (Prepare creates the deployment then).
	DeploymentID string

	// StatePath is the bundle's remote state directory, under which DMS registers
	// the deployment node.
	StatePath   string
	VersionType VersionType

	// Metadata is what the version records about the deploy; see Metadata.
	Metadata Metadata
}

// Metadata is what a version records about the bundle, its source and where it
// landed. The service copies these onto the deployment, so they describe it as of
// its most recent version.
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

// NewRecording returns a Recording for the deployment described by opts.
func NewRecording(opts RecordingOptions) Recording {
	return &recording{
		client:       opts.Client,
		deploymentID: opts.DeploymentID,
		statePath:    opts.StatePath,
		versionType:  opts.VersionType,
		metadata:     opts.Metadata,
	}
}

// Disabled returns a Recording that records nothing, for a bundle that does not record
// deployment history.
func Disabled() Recording {
	return disabled{}
}

// disabled records nothing. Its Prepare leaves no deployment and no version, and its Start no
// sink, so a caller that stamps a version or installs the sink finds nothing to install.
type disabled struct{}

func (disabled) Prepare(context.Context) error      { return nil }
func (disabled) DeploymentID() string               { return "" }
func (disabled) Version() int64                     { return 0 }
func (disabled) Finish(context.Context, bool) error { return nil }

func (disabled) Start(context.Context, []StagedOperation) (*OperationSink, error) {
	return nil, nil
}

// recording records with the service.
type recording struct {
	client       *Client
	deploymentID string
	statePath    string
	versionType  VersionType
	metadata     Metadata

	// populated by Prepare: the version number this deploy intends to create. It is known
	// before the version exists so it can be stamped onto the resources the plan is
	// computed from.
	versionNum        int64
	previousVersionID string

	// populated by Start, once the version actually exists. A deploy the user declines
	// never gets here, so there is nothing to complete or heartbeat.
	versionCreated bool
	stopHeartbeat  context.CancelFunc

	// completed makes Finish idempotent, so a caller that completes the version early can
	// still defer it unconditionally.
	completed bool
}

func (r *recording) DeploymentID() string { return r.deploymentID }

func (r *recording) Version() int64 { return r.versionNum }

// Prepare implements Recording.
func (r *recording) Prepare(ctx context.Context) error {
	versionID, err := r.resolveNextVersion(ctx)
	if err != nil {
		return err
	}

	versionNum, err := strconv.ParseInt(versionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse version ID %q: %w", versionID, err)
	}
	r.versionNum = versionNum
	return nil
}

// Start implements Recording.
func (r *recording) Start(ctx context.Context, staged []StagedOperation) (*OperationSink, error) {
	// A deploy calls Prepare itself, because the version number is stamped onto every job and
	// pipeline before the plan is computed. A destroy creates a version too, but stamps
	// nothing, so it has no reason to settle the deployment any earlier than here.
	if r.versionNum == 0 {
		if err := r.Prepare(ctx); err != nil {
			return nil, err
		}
	}

	// The server rejects this unless the version number exceeds last_version_id and
	// previous_version_id matches it, which is what makes claiming the number up front
	// safe: a deploy that took it in the meantime is reported, not overwritten.
	version, err := r.client.CreateVersion(ctx, r.deploymentID, r.versionNum, CreateVersionRequest{
		CliVersion:        build.GetInfo().Version,
		VersionType:       r.versionType,
		TargetName:        r.metadata.TargetName,
		DisplayName:       r.metadata.DisplayName,
		PreviousVersionId: r.previousVersionID,
		DeploymentMode:    r.metadata.Mode,
		Operations:        staged,
		GitInfo:           r.metadata.Git,
		WorkspaceInfo:     r.metadata.Workspace,
	})
	if err != nil {
		// The service caps how many operations one version may stage, so a bundle past the
		// cap cannot be recorded at all. Say so rather than passing the raw API error on.
		if isResourceExhaustedErr(err) {
			return nil, fmt.Errorf("this bundle deploys %d resources, more than the deployment metadata service records in one version: %w", len(staged), err)
		}
		// A 409 ABORTED means another deploy claimed this version number in between
		// Prepare and here.
		if isAbortedErr(err) {
			return nil, fmt.Errorf("another deploy already claimed version %d of this deployment, try again: %w", r.versionNum, err)
		}
		return nil, fmt.Errorf("failed to create deployment version: %w", err)
	}

	r.versionCreated = true
	r.stopHeartbeat = startHeartbeat(ctx, r.client, r.deploymentID, r.versionNum)
	log.Infof(ctx, "Created deployment version: deployment=%s version=%s", r.deploymentID, version.VersionId)

	return newOperationSink(ctx, r.client, r.deploymentID, r.versionNum), nil
}

// Finish implements Recording.
func (r *recording) Finish(ctx context.Context, success bool) error {
	reason := bundledeployments.VersionCompleteVersionCompleteSuccess
	if !success {
		reason = bundledeployments.VersionCompleteVersionCompleteFailure
	}

	if !r.versionCreated || r.completed {
		return nil
	}
	r.completed = true

	r.stopHeartbeat()

	if err := r.client.CompleteVersion(ctx, r.deploymentID, r.versionNum, reason); err != nil {
		return err
	}
	log.Infof(ctx, "Completed deployment version: deployment=%s version=%d reason=%s", r.deploymentID, r.versionNum, reason)

	// For destroy operations, delete the deployment record after the version
	// completes successfully.
	if reason == bundledeployments.VersionCompleteVersionCompleteSuccess && r.versionType == VersionTypeDestroy {
		if err := r.client.DeleteDeployment(ctx, r.deploymentID); err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	}

	return nil
}

// resolveNextVersion creates the deployment if this is the first deploy, and returns
// the version ID to create under it.
func (r *recording) resolveNextVersion(ctx context.Context) (versionID string, err error) {
	if r.deploymentID != "" {
		// The ID came from a BUNDLE_DEPLOYMENT node that get-status returned, and by
		// design the service has a deployment for every such node, so a not-found
		// here means that invariant is broken rather than anything the user did.
		dep, getErr := r.client.GetDeployment(ctx, r.deploymentID)
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
			r.previousVersionID = dep.LastVersionId
		}
	} else {
		// First deploy: create the deployment so the server assigns an ID.
		// initial_parent_path is required - the node the service creates under it is
		// what ResolveDeploymentID reads back later.
		id, createErr := r.client.CreateDeployment(ctx, r.statePath, r.metadata.TargetName)
		if createErr != nil {
			return "", fmt.Errorf("failed to create deployment: %w", createErr)
		}
		r.deploymentID = id
		versionID = "1"
	}

	return versionID, nil
}

// startHeartbeat starts a background goroutine that sends heartbeats to keep
// the deployment version's lease alive. Returns a cancel function to stop it.
func startHeartbeat(ctx context.Context, client *Client, deploymentID string, version int64) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(defaultHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := client.Heartbeat(ctx, deploymentID, version)
				if err != nil {
					// A 409 ABORTED is expected if the version was completed
					// between the ticker firing and the heartbeat.
					if isAbortedErr(err) {
						log.Debugf(ctx, "Heartbeat stopped: version already completed")
						return
					}
					log.Warnf(ctx, "Failed to send deployment heartbeat: %v", err)
				} else {
					log.Debugf(ctx, "Deployment heartbeat sent: deployment=%s version=%d", deploymentID, version)
				}
			}
		}
	}()

	return cancel
}

// isResourceExhaustedErr reports whether the service refused the call for exceeding a quota.
func isResourceExhaustedErr(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.ErrorCode == "RESOURCE_EXHAUSTED"
}

// isAbortedErr reports whether err is an HTTP 409 ABORTED from the DMS API.
func isAbortedErr(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.StatusCode == http.StatusConflict && apiErr.ErrorCode == "ABORTED"
}
