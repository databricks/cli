package dms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
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

	// DeploymentID is resolved from the deployment's workspace node when the state is opened,
	// and empty until the first recorded deploy - Start creates the deployment then.
	DeploymentID string

	// LastVersionID is the deployment's most recent version, read with it, and empty when the
	// deployment has none. The version this run creates is the next one after it.
	LastVersionID string

	// StatePath is the bundle's remote state directory, under which DMS registers
	// the deployment node.
	StatePath   string
	VersionType VersionType

	// Metadata is what this run says about the bundle; see Metadata.
	Metadata Metadata

	// Deployment is what the service already holds, read with the last version, and nil before
	// the first recorded deploy. Metadata that matches it is not written again.
	Deployment *bundledeployments.Deployment
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

// deploymentFields are the deployment's own metadata, in the order a mask lists them. Git is not
// among them: the service derives the deployment's from the version that carried it.
var deploymentFields = []string{"display_name", "target_name", "deployment_mode", "workspace_info"}

// sameWorkspaceInfo compares the paths alone. The SDK records which fields a response carried in
// ForceSendFields, so a record read back never deep-equals one built here, and comparing the
// structs whole would report every run as a change.
func sameWorkspaceInfo(want, current *bundledeployments.WorkspaceInfo) bool {
	if want == nil || current == nil {
		return want == nil && current == nil
	}

	a, b := *want, *current
	a.ForceSendFields, b.ForceSendFields = nil, nil
	return reflect.DeepEqual(a, b)
}

// deployment renders the metadata the deployment owns.
func (m Metadata) deployment() bundledeployments.Deployment {
	return bundledeployments.Deployment{
		DisplayName:    m.DisplayName,
		TargetName:     m.TargetName,
		DeploymentMode: m.Mode,
		WorkspaceInfo:  m.Workspace,
	}
}

// staleFields returns the mask that brings current up to m, empty when the deployment already
// says what this run would say.
func (m Metadata) staleFields(current *bundledeployments.Deployment) string {
	if current == nil {
		return strings.Join(deploymentFields, ",")
	}

	want := m.deployment()
	var stale []string
	if want.DisplayName != current.DisplayName {
		stale = append(stale, "display_name")
	}
	if want.TargetName != current.TargetName {
		stale = append(stale, "target_name")
	}
	if want.DeploymentMode != current.DeploymentMode {
		stale = append(stale, "deployment_mode")
	}
	if !sameWorkspaceInfo(want.WorkspaceInfo, current.WorkspaceInfo) {
		stale = append(stale, "workspace_info")
	}
	return strings.Join(stale, ",")
}

// NewBufferedClient returns a client for the deployment described by opts, with the version it
// will create settled: the plan is stamped with that number before the version exists. The buffer
// does not run until Start creates it.
func NewBufferedClient(opts Options) (*BufferedClient, error) {
	versionNum := int64(1)
	if opts.LastVersionID != "" {
		last, err := strconv.ParseInt(opts.LastVersionID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_version_id %q: %w", opts.LastVersionID, err)
		}
		versionNum = last + 1
	}

	return &BufferedClient{
		client:            opts.Client,
		deploymentID:      opts.DeploymentID,
		statePath:         opts.StatePath,
		versionType:       opts.VersionType,
		metadata:          opts.Metadata,
		current:           opts.Deployment,
		versionNum:        versionNum,
		previousVersionID: opts.LastVersionID,
	}, nil
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

	// current is the deployment as the service holds it, nil when there is none yet.
	current *bundledeployments.Deployment

	// The version this run will create, known before it exists so the plan can be stamped with
	// it, and the one it must follow.
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
	sequenceIDs map[string]string

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

// EnsureDeployment creates the deployment if the bundle has none, so a deploy can stamp its id
// onto the resources the plan is computed from. Start creates it too, but that is after the plan.
func (c *BufferedClient) EnsureDeployment(ctx context.Context) error {
	if c == nil {
		return nil
	}
	return c.ensureDeployment(ctx)
}

// ensureDeployment creates the deployment on a first deploy, or writes the metadata this run
// changed. initial_parent_path is required on create, and the node the service makes under it is
// what the next run resolves the id from.
func (c *BufferedClient) ensureDeployment(ctx context.Context) error {
	switch mask := c.metadata.staleFields(c.current); {
	case c.deploymentID == "":
		id, err := c.client.CreateDeployment(ctx, c.statePath, c.metadata)
		if err != nil {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
		c.deploymentID = id
	case mask != "":
		if err := c.client.UpdateDeployment(ctx, c.deploymentID, c.metadata, mask); err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}
	default:
		return nil
	}

	// The service now holds what this run says, so the second call - Start makes one after the
	// deploy's own - has nothing left to write.
	written := c.metadata.deployment()
	c.current = &written
	return nil
}

// Start creates the version, staging an operation for each resource, and starts the buffer that
// fills them in. The staged set is fixed here: the service has no call to add one later, so a
// resource left out can never be recorded.
func (c *BufferedClient) Start(ctx context.Context, staged []StagedOperation) error {
	if c == nil {
		return nil
	}

	if err := c.ensureDeployment(ctx); err != nil {
		return err
	}

	// The server rejects this unless the version number exceeds last_version_id and
	// previous_version_id matches it, which is what makes claiming the number up front
	// safe: a deploy that took it in the meantime is reported, not overwritten.
	version, err := c.client.CreateVersion(ctx, c.deploymentID, c.versionNum, CreateVersionRequest{
		CliVersion:        build.GetInfo().Version,
		VersionType:       c.versionType,
		PreviousVersionId: c.previousVersionID,
		Operations:        staged,
		GitInfo:           c.metadata.Git,
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
	c.sequenceIDs = make(map[string]string)
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
		if err := c.write(ctx, resourceKey, update); err != nil {
			c.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
		}
	}
}

// write sends one update, at the sequence id the resource is at.
func (c *BufferedClient) write(ctx context.Context, key string, update OperationUpdate) error {
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
