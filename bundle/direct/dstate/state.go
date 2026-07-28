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
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/google/uuid"
)

const (
	// currentStateVersion is the schema version written for deployments that record
	// no feature flags, and the version legacy states are migrated up to on load.
	currentStateVersion = 2
	initialBufferSize   = 64 * 1024
	maxWalEntrySize     = 10 * 1024 * 1024
	walSuffix           = ".wal"

	// featureStateVersion is the schema version written once a state records
	// deployment state "feature flags" (see Header.Features). Reading such a state
	// (see migrateState):
	//   - featureStateVersion with no features        -> accept, leave the version as-is
	//   - featureStateVersion with known features     -> accept
	//   - featureStateVersion with unknown features   -> refuse, tell the user to upgrade
	//
	// A featureStateVersion state with no features is equivalent to
	// currentStateVersion, but we deliberately do not flip the on-disk version down
	// to currentStateVersion: a state written at featureStateVersion stays at
	// featureStateVersion. That way a release can start writing
	// featureStateVersion + features without older CLIs either mishandling a feature
	// they lack or rejecting a featureless state outright. featureStateVersion is
	// always 3.
	featureStateVersion = 3

	// FeatureDeploymentHistory marks a state whose resources live in the deployment
	// metadata service rather than in the state file. A CLI that does not know this
	// feature refuses the state instead of deploying against a resource set it
	// cannot see - the file's resources are not authoritative.
	FeatureDeploymentHistory = "deployment_history"

	// supportedStateVersion is the highest schema version this CLI can read. It is
	// normally equal to currentStateVersion — the version this CLI reads is the
	// version it writes — and exceeds it only during a two-phase version bump like
	// the current feature-flag scaffolding, where this CLI reads (but does not
	// write) featureStateVersion. A state newer than this is rejected as too new.
	supportedStateVersion = featureStateVersion
)

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
}

type Header struct {
	StateVersion int    `json:"state_version"`
	CLIVersion   string `json:"cli_version"`
	Lineage      string `json:"lineage"`
	Serial       int    `json:"serial"`

	// Features maps each feature flag this state depends on to a (currently empty)
	// value. A CLI that does not implement one of them refuses the state rather than
	// deploying against it (see migrateState). It is a map so a future CLI can attach
	// per-feature data without reshaping the state. Empty/omitted for states that use
	// no features.
	Features map[string]struct{} `json:"features,omitempty"`
}

// hasFeature reports whether the state records the given feature flag.
func (h *Header) hasFeature(name string) bool {
	_, ok := h.Features[name]
	return ok
}

// setFeature records a feature flag in the state.
func (h *Header) setFeature(name string) {
	if h.Features == nil {
		h.Features = make(map[string]struct{})
	}
	h.Features[name] = struct{}{}
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

func (db *DeploymentState) SaveState(key, newID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}

	// Redact sensitive fields before persisting: secrets must not appear on disk
	// in plaintext. The original struct is not modified; the plan uses the
	// unredacted in-memory value for API calls.
	jsonMessage, err := structwalk.RedactSensitiveFields(state, dyn.SensitiveValueRedacted)
	if err != nil {
		return err
	}

	entry := ResourceEntry{
		ID:        newID,
		State:     json.RawMessage(jsonMessage),
		DependsOn: dependsOn,
	}

	err = appendJSONLine(db.walFile, WALEntry{Key: key, Value: &entry})
	if err == nil {
		db.stateIDs[key] = newID
	}
	return err
}

func (db *DeploymentState) DeleteState(key string) error {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return nil
	}

	err := appendJSONLine(db.walFile, WALEntry{Key: key})
	if err == nil {
		delete(db.stateIDs, key)
	}
	return err
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

// DMSSource tells Open to read resource state from the deployment metadata
// service instead of the state file. Callers pass it only when the bundle set
// experimental.record_deployment_history; a nil *DMSSource keeps Open file-only.
type DMSSource struct {
	Client bundledeployments.BundleDeploymentsInterface

	// DeploymentID is resolved from the deployment's workspace node (see
	// dms.ResolveDeploymentID), and empty before the first recorded deploy.
	DeploymentID string

	// TargetName names the bundle target in the error Open returns for a state that
	// predates the opt-in, since the feature is enabled per target.
	TargetName string
}

// Open reads the deployment state from disk (and recovers the WAL when
// withRecovery is set). With a non-nil dmsSource, resources come from DMS rather
// than the file. Lineage and serial always come from the file, since that is
// what the write path increments.
func (db *DeploymentState) Open(ctx context.Context, path string, withRecovery WithRecovery, withWrite WithWrite, dmsSource *DMSSource) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		panic(fmt.Sprintf("state already opened: %v, cannot open %v", db.Path, path))
	}

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

	if dmsSource != nil {
		// An existing state file has to be marked as DMS-owned before its resources
		// can be read from DMS. Recording only starts on an empty state, so a state
		// with resources and no feature flag predates the opt-in: DMS never saw those
		// resources and is authoritative for the whole set once enabled (see
		// readDMSState), so they would look absent and be created a second time.
		// Migrating such a target is not supported yet.
		if len(db.Data.State) > 0 && !db.Data.hasFeature(FeatureDeploymentHistory) {
			return fmt.Errorf("target %q was deployed without experimental.record_deployment_history and cannot be migrated to it: %s tracks resources this deployment recorded before the feature was enabled. Use a new target, destroy this one and deploy it again, or unset experimental.record_deployment_history", dmsSource.TargetName, path)
		}
		// Mark the state as DMS-owned so a CLI without this feature refuses it rather
		// than deploying against a resource set it cannot see.
		db.Data.setFeature(FeatureDeploymentHistory)
		if dmsSource.DeploymentID != "" {
			if err := db.readDMSState(ctx, dmsSource); err != nil {
				return err
			}
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
		return appendJSONLine(db.walFile, db.newWalHeader())
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
	var newSerial int

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

			// Carry the WAL's features (and the version that goes with them) into the
			// state being written, so recovering a WAL written by a DMS-recording deploy
			// still produces a state marked as DMS-owned.
			for name := range header.Features {
				db.Data.setFeature(name)
			}
			if len(db.Data.Features) > 0 {
				db.Data.StateVersion = featureStateVersion
			}
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
	if hasEntries {
		db.Data.Serial = newSerial
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

	db.Path = ""
	db.Data = Database{}
	db.stateIDs = nil

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

	return appendJSONLine(db.walFile, db.newWalHeader())
}

// newWalHeader builds the header for a fresh WAL. Features carry over from the
// state being written, and a state that records any feature is written at
// featureStateVersion so a CLI that lacks the feature refuses it instead of
// deploying against it. The caller holds db.mu.
func (db *DeploymentState) newWalHeader() Header {
	version := currentStateVersion
	if len(db.Data.Features) > 0 {
		version = featureStateVersion
	}
	return Header{
		Lineage:      db.GetOrInitLineage(),
		Serial:       db.Data.Serial + 1,
		StateVersion: version,
		CLIVersion:   build.GetInfo().Version,
		Features:     db.Data.Features,
	}
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

func (db *DeploymentState) unlockedSave() error {
	data, err := json.MarshalIndent(db.Data, "", " ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(db.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %#v: %w", dir, err)
	}

	err = os.WriteFile(db.Path, data, 0o600)
	if err != nil {
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
