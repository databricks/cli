package dstate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/google/uuid"
)

const (
	// currentStateVersion is the schema version written for deployments that record
	// no feature flags, and the version legacy states are migrated up to on load.
	currentStateVersion = 2
	initialBufferSize   = 64 * 1024
	maxWalEntrySize     = 10 * 1024 * 1024
	walSuffix           = ".wal"

	// featureStateVersion is the schema version a CLI writes once it
	// records deployment state "feature flags" (see Header.Features). This CLI writes
	// it for a state that records a feature, and reads such states as follows
	// (see migrateState):
	//   - featureStateVersion with no features  -> accept and leave the version as-is
	//   - featureStateVersion with recognized features -> accept and leave the version as-is
	//   - featureStateVersion with any other feature -> refuse, tell the user to upgrade
	//
	// A featureStateVersion state with no features is equivalent to
	// currentStateVersion, but we deliberately do not flip the on-disk version down
	// to currentStateVersion: a state written at featureStateVersion stays at
	// featureStateVersion. This is forward-compat scaffolding so that a later release
	// can start writing featureStateVersion + features without older CLIs (with this
	// change) either mishandling a feature they lack or rejecting a featureless state
	// outright. featureStateVersion is always 3.
	featureStateVersion = 3

	// supportedStateVersion is the highest schema version this CLI can read. It is
	// normally equal to currentStateVersion — the version this CLI reads is the
	// version it writes — and exceeds it only during a two-phase version bump like
	// the current feature-flag scaffolding, where this CLI writes featureStateVersion
	// only for a state that records a feature. A state newer than this is rejected as too new.
	supportedStateVersion = featureStateVersion
)

// featureRecordDeploymentHistory marks a state whose resources are also recorded with the
// deployment metadata service. Both stores are kept in step, so the marker is what tells a
// reader the two already agree. A CLI that does not know the name refuses the state rather
// than deploying over a deployment it would leave the service out of step with.
//
// The marker is sticky: once a deployment is recorded, the service holds resources that a
// CLI which is not recording must not touch. So turning recording off does not clear it, and
// deploying such a state without recording is refused (see RequiresDeploymentHistory).
const featureRecordDeploymentHistory = "record_deployment_history"

// recognizedFeatures are the state features this CLI understands. A state recording anything
// outside this set is refused (see migrateState).
var recognizedFeatures = map[string]struct{}{
	featureRecordDeploymentHistory: {},
}

// featuresDocURL is the single documentation page describing deployment state
// feature flags. It is shown when a state records a feature this CLI does not
// support; it is a fixed link for all features. The #state-features anchor points
// at the feature table; if it ever breaks, the user still lands on the page.
const featuresDocURL = "https://docs.databricks.com/aws/en/dev-tools/bundles/state-features#state-features"

// errStaleWAL is returned when the WAL serial is behind the expected serial.
// The caller should delete the stale WAL and proceed normally.
var errStaleWAL = errors.New("stale WAL")

type DeploymentState struct {
	Path    string
	Data    Database
	mu      sync.Mutex
	walFile *os.File

	// Maps resource key to ID. Unlike Data.State, this is up to date during writes (deploys).
	stateIDs map[string]string

	// dmsBufferedClient records each state write with DMS. Nil unless the bundle records deployment
	// history, in which case RegisterDmsBufferedClient installs it.
	dmsBufferedClient *dms.BufferedClient
}

type Header struct {
	StateVersion int `json:"state_version"`

	// CLIVersion is the version of the CLI that last wrote this state. It is
	// refreshed from the WAL header on every deploy that commits changes, so it
	// tracks the most recent writer rather than the CLI that created the state.
	CLIVersion string `json:"cli_version"`

	Lineage string `json:"lineage"`
	Serial  int    `json:"serial"`

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

// RegisterDmsBufferedClient has every subsequent state write recorded with client, so what the
// service holds mirrors the WAL. A nil client records nothing. Called once the version exists,
// which is why it is not an Open option.
func (db *DeploymentState) RegisterDmsBufferedClient(client *dms.BufferedClient) {
	if client == nil {
		return
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	db.dmsBufferedClient = client
}

// RecordFailure records that a resource did not apply, so the history says why rather than
// leaving the resource out. resourceID is the id it had before the failure.
func (db *DeploymentState) RecordFailure(resourceKey, resourceID string, cause error) {
	r := db.recorder()
	if r == nil {
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

	r.RecordFailure(resourceKey, resourceID, recorded, cause)
}

// recorder reads the client under db.mu, which guards it, and returns nil when the bundle does
// not record deployment history.
func (db *DeploymentState) recorder() *dms.BufferedClient {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.dmsBufferedClient
}

func NewDatabase(lineage string, serial int) Database {
	return Database{
		Header: Header{
			StateVersion: currentStateVersion,
			CLIVersion:   build.GetInfo().Version,
			Lineage:      lineage,
			Serial:       serial,
		},
		State: make(map[string]ResourceEntry),
	}
}

// SaveState records the resource's state after an operation was applied to it.
func (db *DeploymentState) SaveState(ctx context.Context, key, newID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	db.AssertOpenedForWrite()

	jsonMessage, err := json.Marshal(state)
	if err != nil {
		return err
	}
	entry := ResourceEntry{
		ID:        newID,
		State:     json.RawMessage(jsonMessage),
		DependsOn: dependsOn,
	}

	db.mu.Lock()
	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}
	err = appendJSONLine(db.walFile, WALEntry{Key: key, Value: &entry})
	if err == nil {
		db.stateIDs[key] = newID
	}
	dmsBufferedClient := db.dmsBufferedClient
	db.mu.Unlock()

	if err != nil {
		return err
	}
	if dmsBufferedClient == nil {
		return nil
	}

	// Recorded after the WAL write, so DMS never reports state the deploy failed to persist,
	// and outside db.mu because recording waits when the service is behind - waiting under the
	// lock would hold up every other resource's write.
	recorded, err := json.Marshal(RecordedState{State: entry.State, DependsOn: dependsOn})
	if err != nil {
		return err
	}
	dmsBufferedClient.RecordOperation(ctx, key, false, newID, recorded)

	return nil
}

// DeleteState drops the resource's state entry: the resource is gone. inProgress records the
// operation as unfinished, which is what the first half of a recreate wants - an interrupted
// deploy must not leave the resource described as finished.
func (db *DeploymentState) DeleteState(ctx context.Context, key string, inProgress bool) error {
	db.AssertOpenedForWrite()

	db.mu.Lock()
	if db.Data.State == nil {
		db.mu.Unlock()
		return nil
	}
	// Read before the delete: DMS needs the id to say which resource went away.
	deletedID := db.stateIDs[key]
	err := appendJSONLine(db.walFile, WALEntry{Key: key})
	if err == nil {
		delete(db.stateIDs, key)
	}
	dmsBufferedClient := db.dmsBufferedClient
	db.mu.Unlock()

	if err != nil {
		return err
	}

	// State is nil: the resource no longer exists. Recorded outside the lock for the
	// same reason as SaveState.
	if dmsBufferedClient != nil {
		dmsBufferedClient.RecordOperation(ctx, key, inProgress, deletedID, nil)
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

// RequiresDeploymentHistory reports whether the state depends on the deployment metadata
// service recording it. Deploying such a state without recording would leave the service
// holding a deployment that no longer matches, so the caller refuses instead.
func (db *DeploymentState) RequiresDeploymentHistory() bool {
	db.AssertOpenedForReadOrWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	_, ok := db.Data.Features[featureRecordDeploymentHistory]
	return ok
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

type (
	// If true, then Open reads the WAL and merges it in the state. If false, and WAL is present, Open returns an error.
	WithRecovery bool

	// If true, the state is opened in Write mode, which enables methods such as SaveState
	// but disables GetResourceEntry (since writes go strictly into WAL and not in memory).
	WithWrite bool
)

// Open reads the deployment state from disk, recovering the WAL when withRecovery is set.
// With a non-nil dmsBufferedClient the resources come from DMS instead; lineage and serial
// still come from the file, since that is what the write path increments. Open only reads
// through the client - RegisterDmsBufferedClient installs it for writing, once a version exists
// to write under.
func (db *DeploymentState) Open(ctx context.Context, path string, withRecovery WithRecovery, withWrite WithWrite, dmsBufferedClient *dms.BufferedClient) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		panic(fmt.Sprintf("state already opened: %v, cannot open %v", db.Path, path))
	}

	err := db.unlockedOpen(ctx, path, withRecovery, withWrite, dmsBufferedClient)
	if err != nil {
		// A failed open must leave the receiver closed. unlockedOpen assigns
		// db.Path before every fallible step, so without this the receiver stays
		// half-initialized and the next Open on it hits the panic above instead
		// of reporting the real error.
		db.reset()
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
}

func (db *DeploymentState) unlockedOpen(ctx context.Context, path string, withRecovery WithRecovery, withWrite WithWrite, dmsBufferedClient *dms.BufferedClient) error {
	db.Path = path
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

	db.stateIDs = make(map[string]string)
	for key, entry := range db.Data.State {
		db.stateIDs[key] = entry.ID
	}

	walPath := db.Path + walSuffix
	_, err = os.Stat(walPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// no WAL, nothing to do
	case err != nil:
		return fmt.Errorf("failed to stat WAL file %s: %w", walPath, err)
	default: // WAL exists
		if withRecovery {
			if err := db.replayWAL(ctx); err != nil {
				return fmt.Errorf("reading state from %s: %w", path, err)
			}
		} else {
			return fmt.Errorf("unexpected WAL file found at %s", walPath)
		}
	}

	if err := migrateState(&db.Data); err != nil {
		return fmt.Errorf("migrating state %s: %w", path, err)
	}

	if dmsBufferedClient != nil {
		// A state this CLI wrote while recording is already in step with the service, so it is
		// safe to carry on from. One written without recording tracks resources the service
		// never saw, and treating it as recorded would create them a second time, so refuse it.
		// The marker is the only thing that distinguishes the two. TODO(DMS): support handing
		// an unmarked state over to the service, via per-resource tombstones.
		_, alreadyRecorded := db.Data.Features[featureRecordDeploymentHistory]
		if !alreadyRecorded && dmsBufferedClient.DeploymentID() == "" && len(db.Data.State) > 0 {
			// The remedy is ordered deliberately: this error also blocks destroy, so the
			// setting has to come out first or there is no way to tear the bundle down.
			return fmt.Errorf(`cannot record deployment history for a bundle that already has deployed resources tracked in %s: only new deployments can be recorded

To record this bundle's history, start it over as a new deployment:
  1. remove experimental.record_deployment_history from your bundle configuration
  2. run "databricks bundle destroy" to delete the existing resources
  3. add experimental.record_deployment_history back and deploy again

To keep the existing resources instead, unset experimental.record_deployment_history`, path)
		}

		// The state file keeps tracking every resource the service records, so mark it as
		// depending on that: a CLI without this feature refuses it (see migrateState) instead
		// of deploying over the deployment and leaving the service behind.
		db.Data.StateVersion = featureStateVersion
		if db.Data.Features == nil {
			db.Data.Features = make(map[string]struct{}, 1)
		}
		db.Data.Features[featureRecordDeploymentHistory] = struct{}{}

		// The service owns the resource set while recording, so the file's copy is never read
		// back as the truth. With no deployment yet there is nothing recorded, and an empty set
		// is the honest answer: this deploy creates the deployment and records into it.
		var recorded []dms.Resource
		if dmsBufferedClient.DeploymentID() != "" {
			var err error
			recorded, err = dmsBufferedClient.ListResources(ctx)
			if err != nil {
				return err
			}
		}
		if err := db.applyDMSState(recorded); err != nil {
			return err
		}
	}

	if withWrite {
		if err := os.MkdirAll(filepath.Dir(walPath), 0o755); err != nil {
			return fmt.Errorf("failed to create state directory: %w", err)
		}
		walFile, err := os.OpenFile(walPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
		}
		db.walFile = walFile
		walHead := Header{
			Lineage:      db.GetOrInitLineage(),
			Serial:       db.Data.Serial + 1,
			StateVersion: currentStateVersion,
			CLIVersion:   build.GetInfo().Version,
		}
		return appendJSONLine(db.walFile, walHead)
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

	hasEntries := lineNumber > 1

	// Only advance the serial when the WAL carried entries, because the caller
	// (replayWAL) persists the new state file only in that case. A header-only
	// WAL is a deploy that started but committed nothing; advancing the serial
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
	}

	state := ExportStateFromData(db.Data)

	db.reset()

	return state, err
}

// UpgradeToWrite transitions from read mode to write mode without re-reading state.
// State must already be open for read. This initializes the WAL for writing.
func (db *DeploymentState) UpgradeToWrite() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path == "" {
		return errors.New("internal error: DeploymentState must be opened first")
	}
	if db.walFile != nil {
		return errors.New("internal error: DeploymentState is already open for write")
	}

	walPath := db.Path + walSuffix
	if err := os.MkdirAll(filepath.Dir(walPath), 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	walFile, err := os.OpenFile(walPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
	}
	db.walFile = walFile

	walHead := Header{
		Lineage:      db.GetOrInitLineage(),
		Serial:       db.Data.Serial + 1,
		StateVersion: currentStateVersion,
		CLIVersion:   build.GetInfo().Version,
	}
	return appendJSONLine(db.walFile, walHead)
}

func (db *DeploymentState) AssertOpenedForReadOrWrite() {
	if db.Path == "" {
		panic("internal error: DeploymentState must be opened first")
	}
}

func (db *DeploymentState) AssertOpenedForRead() {
	db.AssertOpenedForReadOrWrite()
	if db.walFile != nil {
		panic("internal error: DeploymentState must be opened in read mode")
	}
}

func (db *DeploymentState) AssertOpenedForWrite() {
	db.AssertOpenedForReadOrWrite()
	if db.walFile == nil {
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
	data, err := json.MarshalIndent(db.Data, "", " ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(db.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %#v: %w", dir, err)
	}

	// CreateTemp creates the file with mode 0o600, matching the state file.
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(db.Path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for %#v: %w", db.Path, err)
	}
	tmpPath := tmp.Name()
	// Cleans up the temp file on failure; a no-op once the rename succeeded.
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write %#v: %w", tmpPath, err)
	}

	// Close before the rename: on Windows the file must not be open for writing.
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close %#v: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, db.Path); err != nil {
		return fmt.Errorf("failed to save resources state to %#v: %w", db.Path, err)
	}

	return nil
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
