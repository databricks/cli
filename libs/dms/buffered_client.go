package dms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
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

// Options are the dependencies and deployment identity a BufferedClient needs.
type Options struct {
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

// NewBufferedClient returns a client for the deployment described by opts. The buffer does not
// run until Start creates the version there is something to record under.
func NewBufferedClient(opts Options) *BufferedClient {
	return &BufferedClient{
		client:       opts.Client,
		deploymentID: opts.DeploymentID,
		statePath:    opts.StatePath,
		versionType:  opts.VersionType,
		metadata:     opts.Metadata,
	}
}

// BufferedClient is everything one deploy or destroy records with DMS. Operations are buffered
// so a state write never waits on a round trip, and every method is a no-op on a nil client,
// which is what a bundle that does not record deployment history gets.
type BufferedClient struct {
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

	// The buffer, live between Start and Close. queue holds bundle state keys, and pending the
	// newest update per key, so a second write for a resource replaces the first.
	queue     chan string
	done      chan struct{}
	stopQueue func()

	// sequenceIDs holds the token the last update for a resource returned. A resource absent
	// from it has only what staging left, so its first update sends that. Unguarded: run is the
	// only goroutine that writes, one update at a time.
	sequenceIDs map[ResourceKey]string

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the newest update per resource key, absent once the writer takes it.
	pending map[string]OperationUpdate

	err error
}

// DeploymentID is the deployment being recorded under. Empty until Prepare, which creates
// the deployment on a first deploy.
func (c *BufferedClient) DeploymentID() string {
	if c == nil {
		return ""
	}
	return c.deploymentID
}

// Version is the version number Prepare claimed, and zero before it runs.
func (c *BufferedClient) Version() int64 {
	if c == nil {
		return 0
	}
	return c.versionNum
}

// Prepare settles the deployment and the version number this run will create, without
// creating it. Both are needed before the plan, which the version number is stamped onto.
func (c *BufferedClient) Prepare(ctx context.Context) error {
	if c == nil {
		return nil
	}

	versionID, err := c.resolveNextVersion(ctx)
	if err != nil {
		return err
	}

	versionNum, err := strconv.ParseInt(versionID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse version ID %q: %w", versionID, err)
	}
	c.versionNum = versionNum
	return nil
}

// Start creates the version, staging an operation for each resource, and starts the buffer that
// fills them in. The staged set is fixed here: the service has no call to add one later, so a
// resource left out can never be recorded.
func (c *BufferedClient) Start(ctx context.Context, staged []StagedOperation) error {
	if c == nil {
		return nil
	}

	// A deploy calls Prepare itself, because the version number is stamped onto every job and
	// pipeline before the plan is computed. A destroy creates a version too, but stamps
	// nothing, so it has no reason to settle the deployment any earlier than here.
	if c.versionNum == 0 {
		if err := c.Prepare(ctx); err != nil {
			return err
		}
	}

	// The server rejects this unless the version number exceeds last_version_id and
	// previous_version_id matches it, which is what makes claiming the number up front
	// safe: a deploy that took it in the meantime is reported, not overwritten.
	version, err := c.client.CreateVersion(ctx, c.deploymentID, c.versionNum, CreateVersionRequest{
		CliVersion:        build.GetInfo().Version,
		VersionType:       c.versionType,
		TargetName:        c.metadata.TargetName,
		DisplayName:       c.metadata.DisplayName,
		PreviousVersionId: c.previousVersionID,
		DeploymentMode:    c.metadata.Mode,
		Operations:        staged,
		GitInfo:           c.metadata.Git,
		WorkspaceInfo:     c.metadata.Workspace,
	})
	if err != nil {
		// The service caps how many operations one version may stage, so a bundle past the
		// cap cannot be recorded at all. Say so rather than passing the raw API error on.
		if isResourceExhaustedErr(err) {
			return fmt.Errorf("this bundle deploys %d resources, more than the deployment metadata service records in one version: %w", len(staged), err)
		}
		// A 409 Conflict means another deploy claimed this version number in between
		// Prepare and here.
		if isAbortedErr(err) {
			return fmt.Errorf("another deploy already claimed version %d of this deployment, try again: %w", c.versionNum, err)
		}
		return fmt.Errorf("failed to create deployment version: %w", err)
	}

	c.versionCreated = true
	c.stopHeartbeat = startHeartbeat(ctx, c.client, c.deploymentID, c.versionNum)
	log.Infof(ctx, "Created deployment version: deployment=%s version=%s", c.deploymentID, version.VersionId)

	// The version exists, so there is something to record under: start the buffer.
	c.queue = make(chan string, bufferedOperations)
	c.done = make(chan struct{})
	c.pending = make(map[string]OperationUpdate)
	c.sequenceIDs = make(map[ResourceKey]string)
	c.stopQueue = sync.OnceFunc(func() { close(c.queue) })
	go c.run(ctx)

	return nil
}

// Close drains what is buffered, then completes the version. A no-op before Start, and safe twice.
func (c *BufferedClient) Close(ctx context.Context, success bool) error {
	if c == nil {
		return nil
	}

	// Drain first; its error is left to whoever drained, which is apply, and not returned twice.
	if c.Drain() != nil {
		success = false
	}

	reason := bundledeployments.VersionCompleteVersionCompleteSuccess
	if !success {
		reason = bundledeployments.VersionCompleteVersionCompleteFailure
	}

	if !c.versionCreated || c.completed {
		return nil
	}
	c.completed = true

	c.stopHeartbeat()

	if err := c.client.CompleteVersion(ctx, c.deploymentID, c.versionNum, reason); err != nil {
		return err
	}
	log.Infof(ctx, "Completed deployment version: deployment=%s version=%d reason=%s", c.deploymentID, c.versionNum, reason)

	// For destroy operations, delete the deployment record after the version
	// completes successfully.
	if reason == bundledeployments.VersionCompleteVersionCompleteSuccess && c.versionType == VersionTypeDestroy {
		if err := c.client.DeleteDeployment(ctx, c.deploymentID); err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	}

	return nil
}

// resolveNextVersion creates the deployment if this is the first deploy, and returns
// the version ID to create under it.
func (c *BufferedClient) resolveNextVersion(ctx context.Context) (versionID string, err error) {
	if c.deploymentID != "" {
		// The ID came from a BUNDLE_DEPLOYMENT node that get-status returned, and by
		// design the service has a deployment for every such node, so a not-found
		// here means that invariant is broken rather than anything the user did.
		dep, getErr := c.client.GetDeployment(ctx, c.deploymentID)
		switch {
		case errors.Is(getErr, apierr.ErrNotFound), errors.Is(getErr, apierr.ErrResourceDoesNotExist):
			return "", fmt.Errorf("internal error: no deployment found for the file with object id %s: %w", c.deploymentID, getErr)
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
			c.previousVersionID = dep.LastVersionId
		}
	} else {
		// First deploy: create the deployment so the server assigns an ID.
		// initial_parent_path is required - the node the service creates under it is
		// what ResolveDeploymentID reads back later.
		id, createErr := c.client.CreateDeployment(ctx, c.statePath, c.metadata.TargetName)
		if createErr != nil {
			return "", fmt.Errorf("failed to create deployment: %w", createErr)
		}
		c.deploymentID = id
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
					// A 409 Conflict is expected if the version was completed
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

// isAbortedErr reports whether err is an HTTP 409 Conflict from the DMS API, whose error
// code is ABORTED.
func isAbortedErr(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.StatusCode == http.StatusConflict && apiErr.ErrorCode == "ABORTED"
}

// bufferedOperations caps how far ahead of the service a deploy may get; DMS is what the next
// plan reads.
const bufferedOperations = 10

// stagedSequenceID is what CreateVersion leaves on a staged operation, so a resource's first
// update sends it as the precondition.
const stagedSequenceID = "0"

// RecordOperation records a state write. state is the serialized envelope, nil for a delete.
// An earlier failure does not stop it: keep recording, best effort.
func (c *BufferedClient) RecordOperation(ctx context.Context, resourceKey string, inProgress bool, resourceID string, state json.RawMessage) {
	if c == nil {
		return
	}

	update, err := NewStateUpdate(resourceID, state, inProgress)
	if err != nil {
		c.setErr(fmt.Errorf("recording operation for %s: %w", resourceKey, err))
		return
	}

	c.record(resourceKey, update)
}

// RecordFailure records that a resource did not apply, so the history says why rather
// than leaving the resource out.
func (c *BufferedClient) RecordFailure(resourceKey, resourceID string, cause error) {
	if c == nil {
		return
	}

	c.record(resourceKey, NewFailureUpdate(resourceID, cause))
}

// record makes update the one waiting for resourceKey, waiting itself while the queue is
// full. Recording after Close panics, so every caller must return before Close.
func (c *BufferedClient) record(resourceKey string, update OperationUpdate) {
	// Nothing to record under yet. A wiring bug rather than a disabled client, which is nil, so
	// say so instead of dropping the operation.
	if c.queue == nil {
		c.setErr(fmt.Errorf("recorded an operation for %s before the deployment version was created", resourceKey))
		return
	}

	c.mu.Lock()
	waiting, queued := c.pending[resourceKey]
	if queued {
		update = waiting.Merge(update)
	}
	c.pending[resourceKey] = update
	c.mu.Unlock()

	// Already queued: the writer reads the map when it gets to the key, so it picks up
	// what was just stored. No second slot, and no waiting.
	if !queued {
		c.queue <- resourceKey
	}
}

// take claims the update waiting for resourceKey.
func (c *BufferedClient) take(resourceKey string) (OperationUpdate, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	update, ok := c.pending[resourceKey]
	delete(c.pending, resourceKey)
	return update, ok
}

func (c *BufferedClient) run(ctx context.Context) {
	defer close(c.done)

	for resourceKey := range c.queue {
		update, ok := c.take(resourceKey)
		if !ok {
			// Unreachable: a key is queued only when nothing was waiting for it. Guard so
			// a stray key could never write a zero-valued update.
			continue
		}

		// Keep going after a failure, so one bad write does not drop everything behind it.
		if err := c.write(ctx, KeyFromState(resourceKey), update); err != nil {
			c.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
		}
	}
}

// write sends one update, at the sequence id the resource is at.
func (c *BufferedClient) write(ctx context.Context, key ResourceKey, update OperationUpdate) error {
	sequenceID, written := c.sequenceIDs[key]
	if !written {
		sequenceID = stagedSequenceID
	}

	next, err := c.client.UpdateOperation(ctx, c.deploymentID, c.versionNum, key, sequenceID, update)
	if err != nil {
		return err
	}

	// The next write for this resource echoes the sequence id this one earned.
	c.sequenceIDs[key] = next

	return nil
}

// Drain waits for everything buffered to reach the service and returns the first write error,
// which fails the deploy: DMS is the source of truth for what exists. Safe to call twice.
func (c *BufferedClient) Drain() error {
	if c == nil {
		return nil
	}
	if c.queue == nil {
		return c.Err()
	}

	c.stopQueue()
	<-c.done
	return c.Err()
}

// setErr keeps the first error; one failure is enough to fail the deploy.
func (c *BufferedClient) setErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err == nil {
		c.err = err
	}
}

// Err returns the first recording error, or nil.
func (c *BufferedClient) Err() error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.err
}
