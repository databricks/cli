package dstate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

const (
	// currentStateVersion is the schema version written by this CLI and the version
	// legacy states are migrated up to on load.
	currentStateVersion             = 3
	initialBufferSize               = 64 * 1024
	maxWalEntrySize                 = 10 * 1024 * 1024
	walSuffix                       = ".wal"
	historyDrainTimeoutPerOperation = 5 * time.Second
)

// FeatureDeploymentHistory marks a state whose resources are also recorded with the
// deployment metadata service. Both stores are kept in step, so the marker is what tells a
// reader the two already agree. A CLI that does not know the name refuses the state rather
// than deploying over a deployment it would leave the service out of step with.
//
// The marker is sticky: once a deployment is recorded, the service holds resources that a
// CLI which is not recording must not touch. So turning recording off does not clear it, and
// deploying such a state without recording is refused.
const FeatureDeploymentHistory = "deployment_history"

// FeatureDeploymentHistoryBestEffort requires writers to attempt DMS history recording.
// Resource state stays in resources.json; recording errors only warn.
const FeatureDeploymentHistoryBestEffort = "deployment_history_best_effort"

// featuresDocURL explains the state-features mechanism in the error shown when this CLI refuses a
// state that depends on a feature it does not recognize.
const featuresDocURL = "https://docs.databricks.com/aws/en/dev-tools/bundles/state-features#state-features"

// recognizedFeatures is the set of state feature flags this CLI understands. A state depending on
// any feature not listed here is refused (see checkStateFeatures), so a newer CLI's feature is not
// silently clobbered by this one.
var recognizedFeatures = map[string]struct{}{
	FeatureDeploymentHistory:           {},
	FeatureDeploymentHistoryBestEffort: {},
}

// assertNoUnsupportedFeatures refuses a state that depends on a feature this CLI does not recognize.
func assertNoUnsupportedFeatures(features map[string]struct{}) error {
	var unsupported []string
	for feature := range features {
		if _, ok := recognizedFeatures[feature]; !ok {
			unsupported = append(unsupported, feature)
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	slices.Sort(unsupported)
	return fmt.Errorf("the deployment state requires features this CLI does not support: %s; upgrade to the latest CLI version and see %s for more information", strings.Join(unsupported, ", "), featuresDocURL)
}

// errStaleWAL is returned when the WAL serial is behind the expected serial.
// The caller should delete the stale WAL and proceed normally.
var errStaleWAL = errors.New("stale WAL")

// ErrUnsettingRecording is returned by Open when an operation that does not support deployment
// history (such as unbind) reaches a recorded state. Callers present an operation-appropriate
// message via errors.Is.
var ErrUnsettingRecording = errors.New("this operation is not supported for a deployment that records deployment history")

type DeploymentState struct {
	Path    string
	Data    Database
	mu      sync.Mutex
	walFile *os.File

	// mode controls where resource state is read and written, whether DMS
	// history is recorded, and whether DMS errors fail the deployment.
	mode Mode

	// Maps resource key to ID. Unlike Data.State, this is up to date during writes (deploys).
	stateIDs map[string]string

	// operationBuffer records each state write with DMS. Nil unless the bundle records deployment
	// history, in which case InitializeOperationBuffer installs it once the version exists.
	operationBuffer *dms.OperationBuffer

	historyDrainTimeout time.Duration

	// versionCompleted makes CompleteVersion a no-op after the first call, so a deferred safety-net
	// completion after an explicit one does nothing.
	versionCompleted bool

	// DeploymentID is the recorded deployment's id. Set at Open, or by the first deploy that creates
	// the deployment, which Open cannot know about.
	DeploymentID string

	// VersionID is the service-owned counter for recorded deployments. A copy is stored in
	// resources.json as Serial to ensure outdated resource state is not read.
	// Open always takes the authoritative value from the service.
	VersionID int
}

// OpenDmsArgs identifies the recorded deployment Open reads from. The zero value means the
// bundle does not record deployment history, or no deployment exists for it yet.
type OpenDmsArgs struct {
	// DeploymentID is the deployment's server-minted id.
	DeploymentID string

	// LastVersionID is the most recent version the service has recorded, zero when it has none.
	// Open takes the state's version from it: the service owns the number.
	//
	// Equivalent to serial for a file based deployment that uses resources.json to track state,
	LastVersionID int
}

type Header struct {
	StateVersion int `json:"state_version"`

	// CLIVersion is the version of the CLI that last wrote this state. It is
	// refreshed from the WAL header on every deploy that commits changes, so it
	// tracks the most recent writer rather than the CLI that created the state.
	CLIVersion string `json:"cli_version"`

	Lineage string `json:"lineage"`

	// Serial counts state writes. For recorded deployments it is a copy of the current DMS version
	// to ensure outdated resource state is not read; the service remains the source of truth.
	Serial int `json:"serial,omitempty"`

	// Features maps each feature flag this state depends on to a (currently empty)
	// value. It is read to detect a state that depends on features this CLI lacks and
	// refuse it (see migrateState). It is a map so a future CLI can attach per-feature
	// data without reshaping the state. Empty/omitted for states that use no features.
	Features map[string]struct{} `json:"features,omitempty"`
}

type Database struct {
	Header

	// Maps resource key to ResourceEntry which includes ID + full serialized state.
	// This is not updated during write/deploy, those writes go to WAL instead.
	// The State is then reconstructed from WAL.
	State map[string]ResourceEntry `json:"state"`
}

type ResourceEntry struct {
	ID        string                      `json:"__id__"`
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

type WALEntry struct {
	Key   string         `json:"k"`
	Value *ResourceEntry `json:"v,omitempty"` // nil means delete
}

// InitializeOperationBuffer opens the operation buffer that records each subsequent state write with
// DMS under deploymentID and versionID, so what the service holds mirrors the WAL. It uses the client
// Open built, so it is called only when the deployment records history, and once the version exists
// (after approval) - which is why it is not an Open option. It also records the id a first deploy
// just created, which Open could not know, so CompleteVersion later has it.
func (db *DeploymentState) InitializeOperationBuffer(ctx context.Context, deploymentID string, versionID, operationCount int) {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()
	options := dms.BufferOptions{
		BestEffort: db.mode == WritingFileWithHistory || db.mode == WritingFileWithHistoryEnforced,
	}
	db.operationBuffer = dms.StartOperationBuffer(ctx, deploymentID, versionID, options)
	db.historyDrainTimeout = time.Duration(operationCount) * historyDrainTimeoutPerOperation
	db.DeploymentID = deploymentID
	db.VersionID = versionID
}

// getOperationBuffer returns the buffer for the open version, nil when no version is open.
func (db *DeploymentState) getOperationBuffer() *dms.OperationBuffer {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.operationBuffer
}

// RecordingError reports whether recording state writes to the service has failed, so the apply
// stops touching resources once the service is no longer keeping up. File-backed history modes
// always return nil because their resource state remains recoverable from the WAL.
func (db *DeploymentState) RecordingError() error {
	if !db.IsDeploymentMetadataService() {
		return nil
	}
	buf := db.getOperationBuffer()
	if buf == nil {
		return nil
	}
	return buf.Err()
}

// CompleteVersion marks the recorded version done, reporting whether it completed here. A no-op
// returning false when no version was created (the bundle does not record history, or a deploy was
// declined) or the version was already completed, so a deferred safety-net call after an explicit
// one does nothing. Finalize has already drained the buffered operations. It reads only fields that
// survive Finalize's reset and asserts nothing, so it is safe to call after the state is closed.
func (db *DeploymentState) CompleteVersion(ctx context.Context, success bool) (bool, error) {
	if !db.RecordsHistory() {
		return false, nil
	}

	db.mu.Lock()
	buf := db.operationBuffer
	if buf == nil || db.versionCompleted {
		db.mu.Unlock()
		return false, nil
	}
	db.versionCompleted = true
	deploymentID, versionID := db.DeploymentID, db.VersionID
	bestEffort := db.mode == WritingFileWithHistory || db.mode == WritingFileWithHistoryEnforced
	db.mu.Unlock()

	// A recording failure fails the version even when the caller counted the deploy a success: the
	// service does not then hold everything the WAL does. Finalize already drained and surfaced it;
	// this reads the drained buffer's error so a destroy whose uploads failed keeps its record.
	if buf.Err() != nil {
		success = false
	}

	reason := bundledeployments.VersionCompleteVersionCompleteSuccess
	if !success {
		reason = bundledeployments.VersionCompleteVersionCompleteFailure
	}
	w := cmdctx.WorkspaceClient(ctx)
	_, err := w.BundleDeployments.CompleteVersion(ctx, bundledeployments.CompleteVersionRequest{
		Name:             dms.VersionName(deploymentID, versionID),
		CompletionReason: reason,
	})
	if err != nil {
		if bestEffort {
			log.Warnf(ctx, "Failed to complete deployment history version: %s", err)
			return false, nil
		}
		return false, err
	}
	log.Infof(ctx, "Completed deployment version: deployment=%s version=%d reason=%s", deploymentID, versionID, reason)
	// Report the success value the version completed with, not merely that it completed: a destroy
	// deletes the deployment record only for a version that completed successfully.
	return success, nil
}

// RecordFailure records that a resource did not apply, so the history says why rather than
// leaving the resource out. resourceID is the id it had before the failure.
func (db *DeploymentState) RecordFailure(resourceKey, resourceID string, cause error) {
	if !db.RecordsHistory() {
		return
	}

	// The service refuses a failure that leaves a live resource described by nothing, so re-state
	// what it still has. An empty resourceID means there is nothing left: a create that never
	// landed, or a recreate whose delete already went through.
	var recorded json.RawMessage
	if entry, ok := db.GetResourceEntry(resourceKey); resourceID != "" && ok && len(entry.State) > 0 {
		var err error
		recorded, err = json.Marshal(RecordedState{State: entry.State, DependsOn: entry.DependsOn})
		if err != nil {
			// Nothing the caller can act on, so record the failure without the state.
			recorded = nil
		}
	}

	// A failure is normally recorded only while a version is open, so the buffer is set; nil means
	// no version was created, and there is nothing to record against.
	if buf := db.getOperationBuffer(); buf != nil {
		buf.RecordFailure(resourceKey, resourceID, recorded, cause)
	}
}

func NewDatabase(lineage string, serial int) Database {
	return Database{
		StateVersion: currentStateVersion,
		CLIVersion:   build.GetInfo().Version,
		Lineage:      lineage,
		Serial:       serial,
		State:        make(map[string]ResourceEntry),
	}
}

// SaveState records the resource's state after an operation was applied to it.
func (db *DeploymentState) SaveState(ctx context.Context, key, newID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}

	jsonMessage, err := json.Marshal(state)
	if err != nil {
		return err
	}
	entry := ResourceEntry{
		ID:        newID,
		State:     json.RawMessage(jsonMessage),
		DependsOn: dependsOn,
	}

	switch db.mode {
	case WritingDMS:
		db.Data.State[key] = entry
		db.stateIDs[key] = newID
		if buf := db.operationBuffer; buf != nil {
			recorded, err := json.Marshal(RecordedState{State: entry.State, DependsOn: dependsOn})
			if err != nil {
				return err
			}
			buf.RecordOperation(ctx, key, false, newID, recorded)
		}

	case WritingFile:
		if err := appendJSONLine(db.walFile, WALEntry{Key: key, Value: &entry}); err != nil {
			return err
		}
		db.stateIDs[key] = newID

	case WritingFileWithHistory, WritingFileWithHistoryEnforced:
		// File-backed modes record history only after the WAL append succeeds.
		if err := appendJSONLine(db.walFile, WALEntry{Key: key, Value: &entry}); err != nil {
			return err
		}
		db.stateIDs[key] = newID
		if buf := db.operationBuffer; buf != nil {
			recorded, err := json.Marshal(RecordedState{State: entry.State, DependsOn: dependsOn})
			if err != nil {
				return err
			}
			buf.RecordOperation(ctx, key, false, newID, recorded)
		}

	default:
		return fmt.Errorf("cannot save state in mode %d", db.mode)
	}
	return nil
}

// DeleteState drops the resource's state entry: the resource is gone. inProgress records the
// operation as unfinished, which is what the first half of a recreate wants - an interrupted
// deploy must not leave the resource described as finished.
func (db *DeploymentState) DeleteState(ctx context.Context, key string, inProgress bool) error {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return nil
	}
	// Read before the delete below: DMS needs the id to say which resource went away.
	deletedID := db.stateIDs[key]

	switch db.mode {
	case WritingDMS:
		delete(db.Data.State, key)
		if buf := db.operationBuffer; buf != nil {
			buf.RecordOperation(ctx, key, inProgress, deletedID, nil)
		}
		delete(db.stateIDs, key)

	case WritingFile:
		if err := appendJSONLine(db.walFile, WALEntry{Key: key}); err != nil {
			return err
		}
		delete(db.stateIDs, key)

	case WritingFileWithHistory, WritingFileWithHistoryEnforced:
		if err := appendJSONLine(db.walFile, WALEntry{Key: key}); err != nil {
			return err
		}
		if buf := db.operationBuffer; buf != nil {
			buf.RecordOperation(ctx, key, inProgress, deletedID, nil)
		}
		delete(db.stateIDs, key)

	default:
		return fmt.Errorf("cannot delete state in mode %d", db.mode)
	}
	return nil
}

func (db *DeploymentState) GetResourceEntry(key string) (ResourceEntry, bool) {
	// Note, if opened for write, you get the state that you had at the beginning of deploy, not most recent one
	db.AssertOpenedForReadOrWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return ResourceEntry{}, false
	}

	result, ok := db.Data.State[key]
	return result, ok
}

// GetResourceID returns the ID of the resource for the given key, or an empty string if not found.
func (db *DeploymentState) GetResourceID(key string) string {
	db.AssertOpenedForReadOrWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.stateIDs[key]
}

// StateCLIVersion returns the CLI version that last wrote the state, or an empty
// string if the state does not record one (a fresh state that this CLI has not
// written yet). It is the version stored in the on-disk header, not the running
// build's version.
func (db *DeploymentState) StateCLIVersion() string {
	db.AssertOpenedForReadOrWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.Data.CLIVersion
}

// StateFeatures returns the feature flags this state depends on, the signal a plan
// carries so deploy --plan can reject a plan built for a different feature set.
func (db *DeploymentState) StateFeatures() map[string]struct{} {
	db.mu.Lock()
	defer db.mu.Unlock()
	// Copied so callers cannot modify the state's set of features.
	return maps.Clone(db.Data.Features)
}

// GetSerial reports the counter a saved plan is validated against: the service's version for a
// deployment that records history, and the state serial otherwise. When the deployment uses the
// deployment metadata service, the serial from resources.json is not used for plan validation.
func (db *DeploymentState) GetSerial() int {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.mode == ReadingDMS || db.mode == WritingDMS {
		return db.VersionID
	}
	return db.Data.Serial
}

// IsDeploymentMetadataService reports whether this deployment's resource state lives in the
// deployment metadata service rather than the state file. The state's features are the source of
// truth; valid only after Open.
func (db *DeploymentState) IsDeploymentMetadataService() bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.mode == ReadingDMS || db.mode == WritingDMS
}

// RecordsHistory reports whether this mode records resource operations with DMS.
func (db *DeploymentState) RecordsHistory() bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	switch db.mode {
	case ReadingDMS, WritingDMS, WritingFileWithHistory, WritingFileWithHistoryEnforced:
		return true
	default:
		return false
	}
}

// GetOrInitLineage returns the deployment lineage, generating and storing a new
// one if the state does not have one yet. It is the single place the lineage is
// initialized, shared so the direct deployment engine (when it writes state, via
// Open/UpgradeToWrite) and DMS (when it records a deployment) always agree on the
// value.
//
// DMS needs this before the engine writes state: it records the version at lock
// time, which is before the engine assigns the lineage at plan-apply time.
// Seeding db.Data.Lineage here means the subsequent write reuses the same value
// instead of minting a different one.
//
// It does not take db.mu: Open and UpgradeToWrite already hold it, and the DMS
// recorder calls it during deploy setup, before any concurrent state writes.
func (db *DeploymentState) GetOrInitLineage() string {
	if db.Data.Lineage == "" {
		db.Data.Lineage = uuid.New().String()
	}
	return db.Data.Lineage
}

// If true, then Open reads the WAL and merges it in the state. If false, and WAL is present, Open returns an error.
type WithRecovery bool

// Open loads resource state using mode, recovering a leftover WAL when requested.
// DMS reads and history writes require a workspace client on ctx.
func (db *DeploymentState) Open(ctx context.Context, path string, mode Mode, withRecovery WithRecovery, dmsDeployment OpenDmsArgs) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		panic(fmt.Sprintf("state already opened: %v, cannot open %v", db.Path, path))
	}

	err := db.unlockedOpen(ctx, path, mode, withRecovery, dmsDeployment)
	if err != nil {
		// A failed open must leave the receiver closed so it can be retried.
		db.reset()
		db.mode = ReadingFile
		db.operationBuffer = nil
	}
	return err
}

// reset returns the receiver to the not-opened state. Callers must hold db.mu.
func (db *DeploymentState) reset() {
	if db.walFile != nil {
		db.walFile.Close()
		db.walFile = nil
	}
	db.Path = ""
	db.Data = Database{}
	db.stateIDs = nil
	db.historyDrainTimeout = 0
}

func (db *DeploymentState) unlockedOpen(ctx context.Context, path string, mode Mode, withRecovery WithRecovery, dmsDeployment OpenDmsArgs) error {
	if err := mode.validate(); err != nil {
		return err
	}
	db.Path = path
	db.mode = ReadingFile
	if mode == ReadingDMS || mode == WritingDMS {
		db.mode = ReadingDMS
	}
	db.operationBuffer = nil
	db.versionCompleted = false
	db.historyDrainTimeout = 0
	db.DeploymentID = dmsDeployment.DeploymentID
	db.VersionID = dmsDeployment.LastVersionID

	// The state file is the source of truth for whether this deployment records history: read it
	// first.
	data, err := os.ReadFile(db.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			db.Data = NewDatabase("", 0)
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &db.Data); err != nil {
			return err
		}
	}

	if err := migrateState(&db.Data); err != nil {
		return fmt.Errorf("migrating state %s: %w", path, err)
	}

	db.stateIDs = make(map[string]string)
	for key, entry := range db.Data.State {
		db.stateIDs[key] = entry.ID
	}

	recording := mode == ReadingDMS || mode == WritingDMS

	walPath := db.Path + walSuffix
	_, err = os.Stat(walPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// no WAL, nothing to do
	case err != nil:
		return fmt.Errorf("failed to stat WAL file %s: %w", walPath, err)
	default: // WAL exists
		switch {
		case recording:
			// A recorded deployment writes no WAL, so finding one means this state was written by a
			// deployment that did not record history. Refuse rather than discard it: the file is the
			// only record of that deploy's writes, and recording is refused below anyway.
			return fmt.Errorf("unexpected WAL file found at %s: this deployment records deployment history, which does not write one", walPath)
		case bool(withRecovery):
			if err := db.replayWAL(ctx); err != nil {
				return fmt.Errorf("reading state from %s: %w", path, err)
			}
		default:
			return fmt.Errorf("unexpected WAL file found at %s", walPath)
		}
	}

	// Reconcile the config against the state. A brand-new deployment is where recording begins, so
	// the config bootstraps the marker there. On an existing deployment the state is authoritative:
	// the config can neither start recording one that was not (its resources would be created a
	// second time) nor stop recording one that is (the service still holds it).
	_, recorded := db.Data.Features[FeatureDeploymentHistory]
	if recording && recorded && db.Data.Serial > dmsDeployment.LastVersionID {
		return fmt.Errorf("inconsistent deployment state: resource state serial (%d) is ahead of deployment history version (%d)", db.Data.Serial, dmsDeployment.LastVersionID)
	}
	_, bestEffort := db.Data.Features[FeatureDeploymentHistoryBestEffort]
	if bestEffort && (recorded || recording) {
		return errors.New("deployment_history_best_effort file state cannot be opened as DMS-authoritative state")
	}
	switch {
	case recording && !recorded && len(db.Data.State) == 0:
		if db.Data.Features == nil {
			db.Data.Features = make(map[string]struct{}, 1)
		}
		db.Data.Features[FeatureDeploymentHistory] = struct{}{}
		recorded = true
	case recording && !recorded:
		return errors.New(`enabling experimental.deployment_history for an existing deployment is not supported

Run "databricks bundle destroy" first, then deploy again with deployment history enabled`)
	case !recording && recorded:
		return ErrUnsettingRecording
	}

	if recorded {
		// The service is the source of truth for a recorded deployment; the file is a tombstone
		// carrying only the marker, and applyDMSState loads the resources the service holds.
		db.DeploymentID = dmsDeployment.DeploymentID

		if dmsDeployment.DeploymentID != "" {
			w := cmdctx.WorkspaceClient(ctx)
			recorded, err := retry.With(ctx, retry.IsTransientRead, func() ([]bundledeployments.Resource, error) {
				return w.BundleDeployments.ListResourcesAll(ctx, bundledeployments.ListResourcesRequest{
					Parent: dms.DeploymentName(dmsDeployment.DeploymentID),
				})
			})
			if err != nil {
				return fmt.Errorf("listing resources from the deployment history service: %w", err)
			}
			resources := make([]dms.Resource, len(recorded))
			for i, res := range recorded {
				// DMS keys drop the "resources." prefix; add it back for local state.
				resources[i] = dms.Resource{
					Key:   dms.StatePrefix + res.ResourceKey,
					ID:    res.ResourceId,
					State: res.State,
				}
			}
			if err := db.applyDMSState(resources); err != nil {
				return err
			}
		}

		// The service owns the version number; the state file stores a copy.
		// No deployment yet means no versions, hence zero.
		db.VersionID = dmsDeployment.LastVersionID
	}

	if mode.isWritable() {
		return db.unlockedUpgradeToWrite(mode)
	}

	return nil
}

// OpenWithData initializes the state from an in-memory database without reading from disk.
// The state is opened in read mode; call UpgradeToWrite to transition to write mode.
func (db *DeploymentState) OpenWithData(path string, data Database) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		panic(fmt.Sprintf("state already opened: %v, cannot open %v", db.Path, path))
	}

	db.Path = path
	db.mode = ReadingFile
	db.operationBuffer = nil
	db.versionCompleted = false
	db.historyDrainTimeout = 0
	db.DeploymentID = ""
	db.VersionID = 0
	db.Data = data
	db.stateIDs = make(map[string]string)
	for key, entry := range data.State {
		db.stateIDs[key] = entry.ID
	}
}

func (db *DeploymentState) replayWAL(ctx context.Context) error {
	walPath := db.Path + walSuffix
	hasEntries, err := db.mergeWalIntoState(ctx)
	if err != nil {
		if errors.Is(err, errStaleWAL) {
			log.Debugf(ctx, "Deleting stale WAL file %s", walPath)
			_ = os.Remove(walPath)
			return nil
		}
		return fmt.Errorf("WAL recovery failed: %w", err)
	}
	if hasEntries {
		if err := db.unlockedSave(); err != nil {
			return err
		}
	}
	if err := os.Remove(walPath); err != nil {
		return fmt.Errorf("failed to remove WAL file %s: %w", walPath, err)
	}
	return nil
}

func (db *DeploymentState) mergeWalIntoState(ctx context.Context) (bool, error) {
	if db.walFile != nil {
		panic("internal error: walFile must be closed")
	}

	walPath := db.Path + walSuffix
	walFile, err := os.Open(walPath)
	if err != nil {
		return false, fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
	}
	defer walFile.Close()

	scanner := bufio.NewScanner(walFile)
	scanner.Buffer(make([]byte, 0, initialBufferSize), maxWalEntrySize)
	lineNumber := 0
	var corruptedLines [][]byte
	var (
		newSerial     int
		newCLIVersion string
		newFeatures   map[string]struct{}
	)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if lineNumber == 1 {
			var header Header
			if err := json.Unmarshal(line, &header); err != nil {
				return false, fmt.Errorf("failed to parse WAL header: %w", err)
			}

			if db.Data.Lineage == "" && header.Lineage != "" {
				db.Data.Lineage = header.Lineage
			} else if db.Data.Lineage != header.Lineage {
				return false, fmt.Errorf("WAL lineage (%q) does not match state lineage (%q)", header.Lineage, db.Data.Lineage)
			}

			expectedSerial := db.Data.Serial + 1
			if header.Serial < expectedSerial {
				return false, errStaleWAL
			}
			if header.Serial > expectedSerial {
				return false, fmt.Errorf("WAL serial (%d) is ahead of expected (%d), state may be corrupted", header.Serial, expectedSerial)
			}
			if err := assertNoUnsupportedFeatures(header.Features); err != nil {
				return false, err
			}
			newFeatures = maps.Clone(db.Data.Features)
			for feature := range header.Features {
				if newFeatures == nil {
					newFeatures = make(map[string]struct{})
				}
				newFeatures[feature] = struct{}{}
			}
			newSerial = header.Serial
			newCLIVersion = header.CLIVersion
		} else {
			var entry WALEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				log.Warnf(ctx, "Skipping corrupted WAL entry at %s:%d: %v", walPath, lineNumber, err)
				corruptedLines = append(corruptedLines, append([]byte(nil), line...))
				continue
			}
			if db.Data.State == nil {
				db.Data.State = make(map[string]ResourceEntry)
			}
			if entry.Value == nil {
				delete(db.Data.State, entry.Key)
				delete(db.stateIDs, entry.Key)
			} else {
				db.Data.State[entry.Key] = *entry.Value
				db.stateIDs[entry.Key] = entry.Value.ID
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	if len(corruptedLines) > 0 {
		corruptedPath := walPath + ".corrupted"
		corruptedData := bytes.Join(corruptedLines, []byte("\n"))
		if writeErr := os.WriteFile(corruptedPath, corruptedData, 0o600); writeErr != nil {
			log.Warnf(ctx, "Failed to save corrupted WAL entries to %s: %v", corruptedPath, writeErr)
		} else {
			log.Warnf(ctx, "Saved %d corrupted WAL entries to %s", len(corruptedLines), corruptedPath)
		}
	}

	hasEntries := lineNumber > 1 || lineNumber == 1 && !maps.Equal(db.Data.Features, newFeatures)

	// Only advance the serial when the WAL carried changes, because the caller
	// (replayWAL) persists the new state file only in that case. A header-only
	// WAL with unchanged features is a deploy that started but committed nothing; advancing the serial
	// for it leaves the in-memory serial ahead of the persisted one, so the
	// next deploy writes its WAL header at serial+2 and recovery rejects it as
	// "ahead of expected". See acceptance/bundle/deploy/wal/header-only-wal.
	//
	// The CLI version moves with the serial for the same reason: it records the
	// CLI that last wrote the state, so it is only accurate once that write is
	// persisted. Without this the field keeps the version of the CLI that first
	// created the state, no matter how many times a newer CLI deploys over it.
	if hasEntries {
		db.Data.Serial = newSerial
		db.Data.CLIVersion = newCLIVersion
		db.Data.Features = newFeatures
	}

	return hasEntries, nil
}

// Finalize replays the WAL (if open for write), captures the resulting state, and resets.
// Safe to call multiple times or on an already-finalized state.
// Returns the exported state as of the end of this operation.
func (db *DeploymentState) Finalize(ctx context.Context) (resourcestate.ExportedResourcesMap, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path == "" {
		return nil, nil
	}

	var err error

	if db.walFile != nil {
		closeErr := db.walFile.Close()
		if closeErr != nil {
			log.Warnf(ctx, "Error when closing .wal file, possibly corrupted state file: %s", closeErr)
		}
		db.walFile = nil
		err = db.replayWAL(ctx)
	} else if db.mode == WritingDMS {
		// replayWAL is what normally persists the file. Without one, write the tombstone here so
		// the header (lineage, state version, CLI version) still lands on disk - even when no
		// operations were recorded, since the deployment and any version this run created exist
		// either way.
		err = db.unlockedSave()
	}

	// File state is persisted before waiting on optional history recording.
	if db.operationBuffer != nil {
		bestEffort := db.mode == WritingFileWithHistory || db.mode == WritingFileWithHistoryEnforced
		drainCtx := ctx
		cancel := func() {}
		if bestEffort {
			drainCtx, cancel = context.WithTimeout(ctx, db.historyDrainTimeout)
		}
		drainErr := db.operationBuffer.Drain(drainCtx)
		cancel()
		if drainErr != nil {
			if bestEffort {
				log.Warnf(ctx, "Failed to record deployment history: %s", drainErr)
			} else {
				err = errors.Join(err, drainErr)
			}
		}
	}

	state := ExportStateFromData(db.Data)

	db.reset()

	return state, err
}

// UpgradeToWrite selects a write policy without re-reading resource state.
func (db *DeploymentState) UpgradeToWrite(mode Mode) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.unlockedUpgradeToWrite(mode)
}

func (db *DeploymentState) unlockedUpgradeToWrite(mode Mode) error {
	if db.Path == "" {
		return errors.New("internal error: DeploymentState must be opened first")
	}
	if db.mode.isWritable() {
		return errors.New("internal error: DeploymentState is already open for write")
	}
	if !mode.isWritable() {
		return fmt.Errorf("invalid deployment state write mode: %d", mode)
	}
	if err := assertNoUnsupportedFeatures(db.Data.Features); err != nil {
		return err
	}
	_, recorded := db.Data.Features[FeatureDeploymentHistory]
	_, bestEffort := db.Data.Features[FeatureDeploymentHistoryBestEffort]
	if (db.mode == ReadingDMS) != (mode == WritingDMS) || recorded != (mode == WritingDMS) {
		return errors.New("cannot change the deployment state source when upgrading to write")
	}
	if bestEffort && mode != WritingFileWithHistory && mode != WritingFileWithHistoryEnforced {
		return errors.New("deployment_history_best_effort state requires writing with deployment history enabled")
	}

	if mode == WritingDMS {
		db.Data.Lineage = db.GetOrInitLineage()
		db.Data.StateVersion = currentStateVersion
		db.Data.CLIVersion = build.GetInfo().Version
		db.mode = mode
		return nil
	}

	walPath := db.Path + walSuffix
	if err := os.MkdirAll(filepath.Dir(walPath), 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	walFile, err := os.OpenFile(walPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
	}
	features := maps.Clone(db.Data.Features)
	if mode == WritingFileWithHistoryEnforced {
		if features == nil {
			features = make(map[string]struct{})
		}
		features[FeatureDeploymentHistoryBestEffort] = struct{}{}
	}
	walHead := Header{
		Lineage:      db.GetOrInitLineage(),
		Serial:       db.Data.Serial + 1,
		StateVersion: currentStateVersion,
		CLIVersion:   build.GetInfo().Version,
		Features:     features,
	}
	if err := appendJSONLine(walFile, walHead); err != nil {
		// O_EXCL guarantees this is ours; an incomplete header cannot be recovered.
		walFile.Close()
		os.Remove(walPath)
		return fmt.Errorf("failed to write WAL header to %s: %w", walPath, err)
	}
	db.walFile = walFile
	db.mode = mode
	return nil
}

// Persist writes the current in-memory state to its file immediately, without a WAL cycle.
// Used to materialize a freshly opened state that produces no WAL entries - e.g. an empty
// migrated state - as a base file on disk, so a later write has a base and Finalize's
// header-only WAL leaves it intact.
func (db *DeploymentState) Persist() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.unlockedSave()
}

// IsOpen reports whether the state has been opened (for read or write). It lets
// callers probe the state without risking the panic in AssertOpenedForReadOrWrite,
// e.g. code paths shared with the terraform engine where the state DB is never opened.
func (db *DeploymentState) IsOpen() bool {
	return db.Path != ""
}

func (db *DeploymentState) AssertOpenedForReadOrWrite() {
	if db.Path == "" {
		panic("internal error: DeploymentState must be opened first")
	}
}

func (db *DeploymentState) AssertOpenedForRead() {
	db.AssertOpenedForReadOrWrite()
	if db.mode.isWritable() {
		panic("internal error: DeploymentState must be opened in read mode")
	}
}

func (db *DeploymentState) AssertOpenedForWrite() {
	db.AssertOpenedForReadOrWrite()
	if !db.mode.isWritable() {
		panic("internal error: DeploymentState must be opened in write mode")
	}
}

// ExportStateFromData extracts resource IDs and ETags from a database snapshot.
func ExportStateFromData(data Database) resourcestate.ExportedResourcesMap {
	result := make(resourcestate.ExportedResourcesMap)
	for key, entry := range data.State {
		// Match on the exact resource type, not a substring of the key, so a
		// sub-resource entry like resources.<group>.<name>.permissions is not
		// mistaken for the resource itself.
		resourceType := config.GetResourceTypeFromKey(key)

		var etag string
		// Extract etag for resources that use it for drift detection
		// (dashboards and genie_spaces). Both follow the same pattern of
		// persisting the backend-returned etag in state and comparing it
		// against the remote on the next plan via OverrideChangeDesc.
		// covered by test cases:
		//   - bundle/deploy/dashboard/detect-change
		//   - bundle/resources/genie_spaces/simple
		if (resourceType == "dashboards" || resourceType == "genie_spaces") && len(entry.State) > 0 {
			var holder struct {
				Etag string `json:"etag"`
			}
			if err := json.Unmarshal(entry.State, &holder); err == nil {
				etag = holder.Etag
			}
		}

		// Persist a run's resolved job_id so read-only commands can build its
		// URL; in config it is a deploy-time-only ${resources.jobs.*.id} reference.
		var jobID int64
		if resourceType == "job_runs" && len(entry.State) > 0 {
			var holder struct {
				JobID int64 `json:"job_id"`
			}
			if err := json.Unmarshal(entry.State, &holder); err == nil {
				jobID = holder.JobID
			}
		}

		result[key] = resourcestate.ResourceState{
			ID:             entry.ID,
			ETag:           etag,
			JobID:          jobID,
			StateSizeBytes: len(entry.State),
		}
	}
	return result
}

func (db *DeploymentState) ExportState(ctx context.Context) resourcestate.ExportedResourcesMap {
	return ExportStateFromData(db.Data)
}

// unlockedSave persists the in-memory state to db.Path by writing a temp file in
// the same directory and renaming it over the destination, so an interrupted save
// cannot leave a half-written state file behind.
//
// Writing in place would be unrecoverable: replayWAL saves the merged state and
// only then removes the WAL, and Open parses the state file before it looks at
// the WAL. A torn write would therefore leave a state file that Open rejects
// next to an intact WAL it never reads.
func (db *DeploymentState) unlockedSave() error {
	data, err := json.MarshalIndent(db.dataForFile(), "", " ")
	if err != nil {
		return err
	}

	if err := atomicfile.Write(db.Path, data, 0o600, atomicfile.MkDir(0o755)); err != nil {
		return fmt.Errorf("failed to save resources state to %#v: %w", db.Path, err)
	}

	return nil
}

// Data to persist in the remote resources.json file.
func (db *DeploymentState) dataForFile() Database {
	if db.mode == ReadingDMS || db.mode == WritingDMS {
		header := db.Data.Header
		if header.Features == nil {
			header.Features = make(map[string]struct{})
		}
		header.Features[FeatureDeploymentHistory] = struct{}{}
		// Store a copy of the current version to ensure outdated resource state is not read.
		// Open still takes the authoritative version from the service.
		header.Serial = db.VersionID

		return Database{
			Header: header,
			State:  map[string]ResourceEntry{},
		}
	}

	return db.Data
}

func appendJSONLine(file *os.File, obj any) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = file.Write(data)
	// no fsync here, not needed
	return err
}
