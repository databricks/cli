package statemgmt

import (
	"bytes"
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
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/shortid"
)

type AlwaysPull bool

// zipMagic starts every ZIP archive; a state file is JSON and never does.
var zipMagic = []byte("PK\x03\x04")

type StateDesc struct {
	Serial   int                 `json:"serial"`
	Lineage  string              `json:"lineage"`
	Features map[string]struct{} `json:"features,omitempty"`

	// additional fields describing state:
	SourcePath string
	Content    []byte

	Engine  engine.EngineType `json:"-"`
	IsLocal bool              `json:"-"`

	AllStates []*StateDesc

	// Deployment is the deployment-history lookup made alongside the remote state reads; nil when
	// none was made (state not pulled, or deployment history not configured).
	Deployment *DeploymentLookup `json:"-"`
}

// DeploymentLookup is the result of dms.LookupDeploymentID.
type DeploymentLookup struct {
	ID  string
	Err error
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

func localRead(ctx context.Context, fullPath string, engine engine.EngineType) *StateDesc {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			logdiag.LogError(ctx, fmt.Errorf("reading %s: %w", filepath.ToSlash(fullPath), err))
		}
		return nil
	}

	state := &StateDesc{}
	err = json.Unmarshal(content, state)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("parsing %s: %w", filepath.ToSlash(fullPath), err))
	}

	state.SourcePath = filepath.ToSlash(fullPath)
	state.Engine = engine
	state.IsLocal = true
	// not populating .content, not needed for local

	return state
}

func _filerRead(ctx context.Context, f filer.Filer, path string) (*StateDesc, error) {
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

	state := &StateDesc{}
	err = json.Unmarshal(content, state)
	if err != nil {
		// The workspace exports a directory as a ZIP archive rather than failing, so a directory
		// at the state path reads fine and only shows up here.
		if bytes.HasPrefix(content, zipMagic) {
			return nil, errors.New("not a file: the workspace has a directory at this path")
		}
		return nil, fmt.Errorf("parsing state: %w", err)
	}

	// QQQ: would be nice to merge with filer root
	state.SourcePath = filepath.ToSlash(path)
	state.IsLocal = false
	state.Content = content
	return state, nil
}

func filerRead(ctx context.Context, f filer.Filer, path string, engine engine.EngineType) *StateDesc {
	state, err := _filerRead(ctx, f, path)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("reading %s: %w", path, err))
	} else if state != nil {
		log.Debugf(ctx, "read %s: %s", path, state.String())
		state.Engine = engine
	}
	return state
}

// PullResourcesState determines correct state to use by reading all 4 states (terraform/direct, local/remote).
func PullResourcesState(ctx context.Context, b *bundle.Bundle, alwaysPull AlwaysPull) *StateDesc {
	var err error

	// We read all 4 possible states: terraform/direct X local/remote and pick the most recent one.
	_, localPathDirect := b.StateFilenameDirect(ctx)
	_, localPathTerraform := b.StateFilenameTerraform(ctx)

	states, deployment := readStates(ctx, b, alwaysPull)

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
	winner.Deployment = deployment

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

func readStates(ctx context.Context, b *bundle.Bundle, alwaysPull AlwaysPull) ([]*StateDesc, *DeploymentLookup) {
	var states []*StateDesc
	var deployment *DeploymentLookup

	remotePathDirect, localPathDirect := b.StateFilenameDirect(ctx)
	remotePathTerraform, localPathTerraform := b.StateFilenameTerraform(ctx)

	if logdiag.HasError(ctx) {
		return nil, nil
	}

	directLocalState := localRead(ctx, localPathDirect, engine.EngineDirect)
	terraformLocalState := localRead(ctx, localPathTerraform, engine.EngineTerraform)

	if (directLocalState == nil && terraformLocalState == nil) || alwaysPull {
		f, err := deploy.StateFiler(ctx, b)
		if err != nil {
			logdiag.LogError(ctx, err)
			return nil, nil
		}

		var wg sync.WaitGroup
		var directRemoteState, terraformRemoteState *StateDesc

		wg.Go(func() {
			directRemoteState = filerRead(ctx, f, remotePathDirect, engine.EngineDirect)
		})

		wg.Go(func() {
			terraformRemoteState = filerRead(ctx, f, remotePathTerraform, engine.EngineTerraform)
		})

		// Whether the state records deployment history is only known once it is read, but the
		// lookup is needed right after; going by the config overlaps it with the reads. A state
		// that disagrees with the config is either rejected or looked up again by the caller.
		if ConfiguresDeploymentHistory(ctx, b) {
			wg.Go(func() {
				id, err := dms.LookupDeploymentID(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
				deployment = &DeploymentLookup{ID: id, Err: err}
			})
		}

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

	return states, deployment
}

// ConfiguresDeploymentHistory reports whether the bundle config (or env) turns deployment history on.
func ConfiguresDeploymentHistory(ctx context.Context, b *bundle.Bundle) bool {
	var configured *bool
	if b.Config.Experimental != nil {
		configured = b.Config.Experimental.DeploymentHistory
	}
	return bundleenv.RecordsDeploymentHistory(ctx, configured)
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
