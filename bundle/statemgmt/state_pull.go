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
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/useragent"
)

type AlwaysPull bool

type StateDesc struct {
	Serial  int    `json:"serial"`
	Lineage string `json:"lineage"`

	// additional fields describing state:
	SourcePath string
	Content    []byte

	Engine  engine.EngineType `json:"-"`
	IsLocal bool              `json:"-"`

	AllStates []*StateDesc
}

func (s *StateDesc) String() string {
	source := "remote"
	if s.IsLocal {
		source = "local"
	}
	return fmt.Sprintf("%s: %s %s state serial=%d lineage=%q", s.SourcePath, source, s.Engine, s.Serial, s.Lineage)
}

func localRead(ctx context.Context, fullPath string, engine engine.EngineType) *StateDesc {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if !os.IsNotExist(err) {
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
// It will also ensure that if there is state present and env var set then they do not disagree.
func PullResourcesState(ctx context.Context, b *bundle.Bundle, alwaysPull AlwaysPull, requiredEngine engine.EngineType) (context.Context, *StateDesc) {
	var err error

	// We read all 4 possible states: terraform/direct X local/remote and then use env var to validate that correct one is used.
	// However, states and env var cannot disagree.
	_, localPathDirect := b.StateFilenameDirect(ctx)
	_, localPathTerraform := b.StateFilenameTerraform(ctx)

	states := readStates(ctx, b, alwaysPull)

	if logdiag.HasError(ctx) {
		return ctx, nil
	}

	var winner *StateDesc

	if len(states) == 0 {
		winner = &StateDesc{
			// No state, go with user-provided or default
			Engine:  requiredEngine.ThisOrDefault(),
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
		return ctx, winner
	}

	if requiredEngine != engine.EngineNotSet && requiredEngine != winner.Engine {
		logStatesError(ctx, fmt.Sprintf("Required engine %q does not match present state files. Set required engine via %q env var.", requiredEngine, engine.EnvVar), states)
	}

	// Set the engine in the user agent
	// XXX move this outside this function to bundle/config/engine
	ctx = useragent.InContext(ctx, "engine", string(winner.Engine))

	if len(states) == 0 {
		return ctx, winner
	}

	if winner.IsLocal {
		// local state is fresh, nothing to do
		return ctx, winner
	}

	if !winner.IsLocal {
		log.Info(ctx, "Remote state is newer than local state. Using remote resources state.")

		localStatePath := localPathTerraform
		if winner.Engine == engine.EngineDirect {
			localStatePath = localPathDirect
		}

		localStateDir := filepath.Dir(localStatePath)

		err := os.MkdirAll(localStateDir, 0o700)
		if err != nil {
			logdiag.LogError(ctx, err)
			return ctx, winner
		}

		// TODO: write + rename
		err = os.WriteFile(localStatePath, winner.Content, 0o600)
		if err != nil {
			logdiag.LogError(ctx, err)
			return ctx, winner
		}
	}

	return ctx, winner
}

func readStates(ctx context.Context, b *bundle.Bundle, alwaysPull AlwaysPull) []*StateDesc {
	var states []*StateDesc

	remotePathDirect, localPathDirect := b.StateFilenameDirect(ctx)
	remotePathTerraform, localPathTerraform := b.StateFilenameTerraform(ctx)

	if logdiag.HasError(ctx) {
		return nil
	}

	directLocalState := localRead(ctx, localPathDirect, engine.EngineDirect)
	terraformLocalState := localRead(ctx, localPathTerraform, engine.EngineTerraform)

	if (directLocalState == nil && terraformLocalState == nil) || alwaysPull {
		f, err := deploy.StateFiler(b)
		if err != nil {
			logdiag.LogError(ctx, err)
			return nil
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

	return states
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
	var stateStrs []string
	for _, state := range states {
		stateStrs = append(stateStrs, state.String())
	}
	logdiag.LogDiag(ctx, diag.Diagnostic{
		Summary:  msg,
		Severity: diag.Error,
		Detail:   "Available state files:\n- " + strings.Join(stateStrs, "\n- "),
	})
}
