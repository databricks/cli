package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/direct/dstate"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/shortid"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/google/uuid"
)

type AlwaysPull bool

type StateDesc struct {
	Serial   int                 `json:"serial"`
	Lineage  string              `json:"lineage"`
	Features map[string]struct{} `json:"features,omitempty"`

	// additional fields describing state:
	SourcePath string
	Content    []byte
	readErr    error

	Engine  engine.EngineType `json:"-"`
	IsLocal bool              `json:"-"`

	AllStates []*StateDesc

	// Deployment retains a confirmed repair lookup for state initialization.
	DeploymentID string                        `json:"-"`
	Deployment   *bundledeployments.Deployment `json:"-"`
}

func (s *StateDesc) String() string {
	source := "remote"
	if s.IsLocal {
		source = "local"
	}
	return fmt.Sprintf("%s: %s %s state serial=%d lineage=%q", s.SourcePath, source, s.Engine, s.Serial, s.Lineage)
}

// LineageHash returns the lineage in the compact form reported in the user agent
// and telemetry, or "" when the state has no lineage yet (nothing deployed).
func (s *StateDesc) LineageHash() string {
	if s.Lineage == "" {
		return ""
	}
	return shortid.Hash(s.Lineage)
}

func (s *StateDesc) HasRemoteTerraformState() bool {
	for _, s := range s.AllStates {
		if !s.Engine.IsDirect() && !s.IsLocal {
			return true
		}
	}
	return false
}

// IsDMS reports whether the state records deployment history in the deployment metadata service.
func (s *StateDesc) IsDMS() bool {
	_, ok := s.Features["deployment_history"]
	return ok
}

func localRead(fullPath string, engine engine.EngineType) *StateDesc {
	content, err := os.ReadFile(fullPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	state := &StateDesc{}
	if err != nil {
		state.readErr = fmt.Errorf("reading %s: %w", filepath.ToSlash(fullPath), err)
	} else {
		state, err = parseResourceState(content, engine)
		if err != nil {
			state.readErr = fmt.Errorf("parsing %s: %w", filepath.ToSlash(fullPath), err)
		}
	}

	state.SourcePath = filepath.ToSlash(fullPath)
	state.Engine = engine
	state.IsLocal = true
	// not populating .content, not needed for local

	return state
}

// parseResourceState reads the state metadata while validating its JSON shape.
func parseResourceState(content []byte, engine engine.EngineType) (*StateDesc, error) {
	state := &StateDesc{}
	if !engine.IsDirect() {
		return state, json.Unmarshal(content, state)
	}
	var data dstate.Database
	err := json.Unmarshal(content, &data)
	state.Serial = data.Serial
	state.Lineage = data.Lineage
	state.Features = data.Features
	return state, err
}

// isJSONStateError distinguishes damaged JSON from access and transport failures.
func isJSONStateError(err error) bool {
	_, syntaxError := errors.AsType[*json.SyntaxError](err)
	_, typeError := errors.AsType[*json.UnmarshalTypeError](err)
	return syntaxError || typeError
}

func _filerRead(ctx context.Context, f filer.Filer, path string, engine engine.EngineType) (*StateDesc, error) {
	r, err := f.Read(ctx, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening: %w", err)
	}
	defer r.Close()
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading data: %w", err)
	}

	state, parseErr := parseResourceState(content, engine)

	// QQQ: would be nice to merge with filer root
	state.SourcePath = filepath.ToSlash(path)
	state.IsLocal = false
	state.Content = content
	if parseErr != nil {
		return state, fmt.Errorf("parsing state: %w", parseErr)
	}
	return state, nil
}

func filerRead(ctx context.Context, f filer.Filer, path string, engine engine.EngineType) *StateDesc {
	state, err := _filerRead(ctx, f, path, engine)
	if err != nil {
		if state == nil {
			state = &StateDesc{SourcePath: path}
		}
		state.readErr = fmt.Errorf("reading %s: %w", path, err)
	} else if state != nil {
		log.Debugf(ctx, "read %s: %s", path, state.String())
	}
	if state != nil {
		state.Engine = engine
	}
	return state
}

// PullResourcesState selects the newest Terraform or direct state.
// Deployment history selects only remote state and writes a local copy when present.
func PullResourcesState(ctx context.Context, b *bundle.Bundle, alwaysPull AlwaysPull) *StateDesc {
	var err error

	// We read all 4 possible states: terraform/direct X local/remote and pick the most recent one.
	_, localPathDirect := b.StateFilenameDirect(ctx)
	_, localPathTerraform := b.StateFilenameTerraform(ctx)

	var configured *bool
	if b.Config.Experimental != nil {
		configured = b.Config.Experimental.DeploymentHistory
	}
	historyEnabled := bundleenv.RecordsDeploymentHistory(ctx, configured)
	states, readRemote := readStates(ctx, b, AlwaysPull(bool(alwaysPull) || historyEnabled))
	if logdiag.HasError(ctx) {
		return nil
	}
	// DMS owns recorded resource state; a remote marker's serial is only a checkpoint.
	remoteRecorded := slices.ContainsFunc(states, func(state *StateDesc) bool {
		return !state.IsLocal && state.Engine.IsDirect() && state.readErr == nil && state.IsDMS()
	})
	for _, state := range states {
		if state.IsLocal && (historyEnabled || remoteRecorded) {
			continue
		}
		if state.readErr != nil && (!state.Engine.IsDirect() || !isJSONStateError(state.readErr)) {
			logdiag.LogError(ctx, state.readErr)
		}
	}
	if logdiag.HasError(ctx) {
		return nil
	}
	repairCandidate := historyEnabled || slices.ContainsFunc(states, func(state *StateDesc) bool {
		return state.Engine.IsDirect() && (isJSONStateError(state.readErr) || (state.IsLocal && state.IsDMS()))
	})
	if readRemote && repairCandidate {
		repaired, err := recoverDMSState(ctx, b, states)
		if err != nil {
			logdiag.LogError(ctx, err)
			return nil
		}
		if repaired != nil {
			states = []*StateDesc{repaired}
			remoteRecorded = true
		}
	}
	hasRemoteState := slices.ContainsFunc(states, func(state *StateDesc) bool { return !state.IsLocal })
	for _, state := range states {
		if state.readErr == nil || (state.IsLocal && (remoteRecorded || (historyEnabled && hasRemoteState))) {
			continue
		}
		logdiag.LogError(ctx, state.readErr)
	}
	if logdiag.HasError(ctx) {
		return nil
	}
	if remoteRecorded || (historyEnabled && hasRemoteState) {
		states = slices.DeleteFunc(states, func(state *StateDesc) bool {
			if !state.IsLocal {
				return false
			}
			if !state.Engine.IsDirect() {
				if err := os.Remove(state.SourcePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
					logdiag.LogError(ctx, fmt.Errorf("removing cached resource state %s: %w", state.SourcePath, err))
				}
			}
			return true
		})
	}
	if logdiag.HasError(ctx) {
		return nil
	}

	var winner *StateDesc

	if len(states) == 0 {
		winner = &StateDesc{
			// No state: a new deployment uses the direct engine.
			Engine:  engine.EngineDirect,
			IsLocal: true,
			// Lineage and Serial are empty
		}
	} else {
		winner = states[len(states)-1]
		winner.AllStates = states
	}

	log.Infof(ctx, "Available resource state files (from least to most preferred): %v", states)

	err = validateStates(states)
	if err != nil {
		logStatesError(ctx, err.Error(), states)
		return winner
	}

	if len(states) == 0 {
		return winner
	}

	if winner.IsLocal {
		// local state is fresh, nothing to do
		return winner
	}

	if !winner.IsLocal {
		log.Info(ctx, "Remote state is newer than local state. Using remote resources state.")

		localStatePath := localPathTerraform
		if winner.Engine == engine.EngineDirect {
			localStatePath = localPathDirect
		}

		err := atomicfile.Write(localStatePath, winner.Content, 0o600, atomicfile.MkDir(0o700))
		if err != nil {
			logdiag.LogError(ctx, err)
			return winner
		}
	}

	return winner
}

func readStates(ctx context.Context, b *bundle.Bundle, alwaysPull AlwaysPull) ([]*StateDesc, bool) {
	var states []*StateDesc

	remotePathDirect, localPathDirect := b.StateFilenameDirect(ctx)
	remotePathTerraform, localPathTerraform := b.StateFilenameTerraform(ctx)

	if logdiag.HasError(ctx) {
		return nil, false
	}

	directLocalState := localRead(localPathDirect, engine.EngineDirect)
	terraformLocalState := localRead(localPathTerraform, engine.EngineTerraform)

	readRemote := (directLocalState == nil && terraformLocalState == nil) || bool(alwaysPull)
	if readRemote {
		f, err := deploy.StateFiler(ctx, b)
		if err != nil {
			logdiag.LogError(ctx, err)
			return nil, true
		}

		var wg sync.WaitGroup
		var directRemoteState, terraformRemoteState *StateDesc

		wg.Go(func() {
			directRemoteState = filerRead(ctx, f, remotePathDirect, engine.EngineDirect)
		})

		wg.Go(func() {
			terraformRemoteState = filerRead(ctx, f, remotePathTerraform, engine.EngineTerraform)
		})

		wg.Wait()

		// find highest serial across all state files
		// sorting is stable, so initial setting represents preference (later is preferred):
		states = []*StateDesc{terraformRemoteState, terraformLocalState, directRemoteState, directLocalState}
	} else {
		states = []*StateDesc{terraformLocalState, directLocalState}
	}
	states = slices.DeleteFunc(states, func(p *StateDesc) bool { return p == nil })
	slices.SortStableFunc(states, func(a, b *StateDesc) int {
		return a.Serial - b.Serial
	})

	return states, readRemote
}

// recoverDMSState reconstructs an unusable marker only for a confirmed recorded deployment.
func recoverDMSState(ctx context.Context, b *bundle.Bundle, states []*StateDesc) (*StateDesc, error) {
	var surviving *StateDesc
	for _, state := range states {
		if state.readErr == nil {
			if !state.Engine.IsDirect() || !state.IsDMS() || !state.IsLocal {
				return nil, nil
			}
			surviving = state
		}
	}

	deploymentID, deployment, err := dms.FetchDeploymentFromStatePath(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil || deployment == nil {
		return nil, err
	}

	data := dstate.NewDatabase("", 0)
	if surviving != nil {
		var db dstate.DeploymentState
		if err := db.Open(ctx, surviving.SourcePath, dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{}); err != nil {
			return nil, fmt.Errorf("recovering deployment history marker: %w", err)
		}
		data.Header = db.Data.Header
		if len(db.Data.State) != 0 {
			return nil, fmt.Errorf("state file for a recorded deployment carries %d resources, expected none", len(db.Data.State))
		}
		if _, err := db.Finalize(ctx); err != nil {
			return nil, err
		}
	}
	if data.Lineage == "" {
		data.Lineage = uuid.NewString()
		logdiag.LogDiag(ctx, diag.Warningf("Recovering deployment history state with a new lineage; existing saved plans are no longer valid")[0])
	}
	if data.Features == nil {
		data.Features = make(map[string]struct{})
	}
	data.Features[dstate.FeatureDeploymentHistory] = struct{}{}
	content, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("reconstructing deployment history marker: %w", err)
	}
	remotePath, _ := b.StateFilenameDirect(ctx)
	b.NeedsDMSStateUpload = true
	return &StateDesc{
		Serial:       data.Serial,
		Lineage:      data.Lineage,
		Features:     data.Features,
		SourcePath:   remotePath,
		Content:      content,
		Engine:       engine.EngineDirect,
		DeploymentID: deploymentID,
		Deployment:   deployment,
	}, nil
}

func validateStates(states []*StateDesc) error {
	if len(states) == 0 {
		return nil
	}

	var lastLineage *StateDesc

	for _, state := range states {
		if lastLineage == nil {
			lastLineage = state
		} else if lastLineage.Lineage != state.Lineage {
			return errors.New("lineage mismatch in state files")
		}
	}

	terraformSerial := -1
	directSerial := -1

	for _, state := range states {
		if state.Engine.IsDirect() {
			directSerial = max(directSerial, state.Serial)
		} else {
			terraformSerial = max(terraformSerial, state.Serial)
		}
	}

	if directSerial == terraformSerial {
		return errors.New("same serial number in terraform and direct states")
	}

	return nil
}

func logStatesError(ctx context.Context, msg string, states []*StateDesc) {
	logStatesDiag(ctx, diag.Error, msg, states)
}

func logStatesDiag(ctx context.Context, severity diag.Severity, msg string, states []*StateDesc) {
	var stateStrs []string
	for _, state := range states {
		stateStrs = append(stateStrs, state.String())
	}
	logdiag.LogDiag(ctx, diag.Diagnostic{
		Summary:  msg,
		Severity: severity,
		Detail:   "Available state files:\n- " + strings.Join(stateStrs, "\n- "),
	})
}
