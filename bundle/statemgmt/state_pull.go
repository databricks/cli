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
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/useragent"
)

type AlwaysPull bool

type state struct {
	Serial  int64  `json:"serial"`
	Lineage string `json:"lineage"`

	// additional fields describing state:
	content  []byte
	isDirect bool `json:"-"`
	isLocal  bool `json:"-"`
}

func (s *state) String() string {
	kind := "terraform"
	if s.isDirect {
		kind = "direct"
	}
	source := "remote"
	if s.isLocal {
		source = "local"
	}
	return fmt.Sprintf("<%s %s state serial=%d lineage=%q>", source, kind, s.Serial, s.Lineage)
}

func localRead(ctx context.Context, fullPath string, isDirect bool) *state {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logdiag.LogError(ctx, fmt.Errorf("reading %s: %w", filepath.ToSlash(fullPath), err))
		}
		return nil
	}

	state := &state{}
	err = json.Unmarshal(content, state)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("parsing %s: %w", filepath.ToSlash(fullPath), err))
	}

	state.isDirect = isDirect
	state.isLocal = true
	// not populating .content, not needed for local

	return state
}

func _filerRead(ctx context.Context, f filer.Filer, path string) (*state, error) {
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

	state := &state{}
	err = json.Unmarshal(content, state)
	if err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}

	state.isLocal = false
	state.content = content
	return state, nil
}

func filerRead(ctx context.Context, f filer.Filer, path string, isDirect bool) *state {
	state, err := _filerRead(ctx, f, path)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("reading %s: %w", path, err))
	} else if state != nil {
		log.Debugf(ctx, "read %s: %s", path, state.String())
		state.isDirect = isDirect
	}
	return state
}

func PullResourcesState(ctx context.Context, b *bundle.Bundle, alwaysPull AlwaysPull) context.Context {
	_, localPathDirect := b.StateFilenameDirect(ctx)
	_, localPathTerraform := b.StateFilenameTerraform(ctx)

	states := readStates(ctx, b, alwaysPull)

	if logdiag.HasError(ctx) {
		return ctx
	}

	var winner *state

	if len(states) == 0 {
		// no local or remote state; set b.DirectDeployment based on env vars
		isDirect, err := getDirectDeploymentEnv(ctx)
		if err != nil {
			logdiag.LogError(ctx, err)
			return nil
		}
		b.DirectDeployment = &isDirect
	} else {
		winner = states[len(states)-1]
		b.DirectDeployment = ptrBool(winner.isDirect)
	}

	engine := "direct"

	if !*b.DirectDeployment {
		engine = "terraform"
	}

	// Set the engine in the user agent
	ctx = useragent.InContext(ctx, "engine", engine)

	if winner == nil {
		log.Infof(ctx, "No existing resource state found")
		return ctx
	}

	var stateStrs []string
	for _, state := range states {
		stateStrs = append(stateStrs, state.String())
	}

	log.Infof(ctx, "Available resource state files (from least to most preferred): %s", strings.Join(stateStrs, ", "))

	var lastLineage *state

	for _, state := range states {
		if lastLineage == nil {
			lastLineage = state
		} else if lastLineage.Lineage != state.Lineage {
			logdiag.LogError(ctx, fmt.Errorf("lineage mismatch in state files: %s", strings.Join(stateStrs, ", ")))
			return ctx
		}
	}

	if winner.isLocal {
		// local state is fresh, nothing to do
		return ctx
	}

	if !winner.isLocal {
		log.Info(ctx, "Remote state is newer than local state. Using remote resources state.")

		localStatePath := localPathTerraform
		if winner.isDirect {
			localStatePath = localPathDirect
		}

		localStateDir := filepath.Dir(localStatePath)

		err := os.MkdirAll(localStateDir, 0o700)
		if err != nil {
			logdiag.LogError(ctx, err)
			return ctx
		}

		// TODO: write + rename
		err = os.WriteFile(localStatePath, winner.content, 0o600)
		if err != nil {
			logdiag.LogError(ctx, err)
			return ctx
		}
	}

	return ctx
}

func readStates(ctx context.Context, b *bundle.Bundle, alwaysPull AlwaysPull) []*state {
	var states []*state

	remotePathDirect, localPathDirect := b.StateFilenameDirect(ctx)
	remotePathTerraform, localPathTerraform := b.StateFilenameTerraform(ctx)

	if logdiag.HasError(ctx) {
		return nil
	}

	directLocalState := localRead(ctx, localPathDirect, true)
	terraformLocalState := localRead(ctx, localPathTerraform, false)

	if (directLocalState == nil && terraformLocalState == nil) || alwaysPull {
		f, err := deploy.StateFiler(b)
		if err != nil {
			logdiag.LogError(ctx, err)
			return nil
		}

		var wg sync.WaitGroup
		var directRemoteState, terraformRemoteState *state

		wg.Go(func() {
			directRemoteState = filerRead(ctx, f, remotePathDirect, true)
		})

		wg.Go(func() {
			terraformRemoteState = filerRead(ctx, f, remotePathTerraform, false)
		})

		wg.Wait()

		// find highest serial across all state files
		// sorting is stable, so initial setting represents preference:
		states = []*state{terraformRemoteState, terraformLocalState, directRemoteState, directLocalState}
	} else {
		states = []*state{terraformLocalState, directLocalState}
	}
	states = slices.DeleteFunc(states, func(p *state) bool { return p == nil })
	slices.SortStableFunc(states, func(a, b *state) int {
		return int(a.Serial - b.Serial)
	})

	return states
}

func getDirectDeploymentEnv(ctx context.Context) (bool, error) {
	engine := env.Get(ctx, "DATABRICKS_BUNDLE_ENGINE")

	switch engine {
	case "":
		// By default, use Terraform
		return false, nil
	case "terraform":
		return false, nil
	case "direct-exp":
		// We use "direct-exp" while direct backend is not suitable for end users.
		// Once we consider it usable we'll change the value to "direct".
		// This is to prevent accidentally running direct backend with older CLI versions where it was still considered experimental.
		return true, nil
	default:
		return false, fmt.Errorf("unexpected setting for DATABRICKS_BUNDLE_ENGINE=%#v (expected 'terraform' or 'direct-exp' or absent/empty which means 'terraform')", engine)
	}
}

func ptrBool(x bool) *bool { return &x }
