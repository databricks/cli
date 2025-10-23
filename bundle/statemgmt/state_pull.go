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
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/useragent"
)

type state struct {
	Serial  int64  `json:"serial"`
	Lineage string `json:"lineage"`

	// additional fields describing state:
	content  []byte
	isDirect bool `json:"-"`
	isLocal  bool `json:"-"`
}

func (s *state) String() string {
	kind := "direct"
	if !s.isDirect {
		kind = "terraform"
	}
	source := "local"
	if !s.isLocal {
		source = "remote"
	}
	return fmt.Sprintf("<%s %s state serial=%d lineage=%q>", source, kind, s.Serial, s.Lineage)
}

func localRead(ctx context.Context, fullPath string, isDirect bool) *state {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logdiag.LogError(ctx, fmt.Errorf("reading %s: %w", fullPath, err))
		}
		return nil
	}

	state := &state{}
	err = json.Unmarshal(content, state)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("parsing %s: %w", fullPath, err))
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

func PullResourcesState(ctx context.Context, b *bundle.Bundle) context.Context {
	_, localPathDirect := b.StateFilenameDirect(ctx)
	_, localPathTerraform := b.StateFilenameTerraform(ctx)

	states := readStates(ctx, b)

	if logdiag.HasError(ctx) {
		return ctx
	}

	winner := states[3]

	if winner == nil {
		// no local or remote state; set b.DirectDeployment based on env vars
		isDirect, err := getDirectDeploymentEnv(ctx)
		if err != nil {
			logdiag.LogError(ctx, err)
			return nil
		}
		b.DirectDeployment = &isDirect
	} else {
		b.DirectDeployment = ptrBool(winner.isDirect)
	}

	engine := "direct"

	if !*b.DirectDeployment {
		engine = "terraform"
		// XXX move this close to tf.Init()
		bundle.ApplyContext(ctx, b, terraform.Initialize())
	}

	// Set the engine in the user agent
	ctx = useragent.InContext(ctx, "engine", engine)

	if winner == nil {
		log.Infof(ctx, "No existing resource state found")
		return ctx
	}

	var stateStrs []string
	for _, state := range states {
		if state == nil {
			continue
		}
		stateStrs = append(stateStrs, state.String())
	}

	log.Infof(ctx, "Available resource state files (from least to most preferred): %s", strings.Join(stateStrs, ", "))

	var lastLineage *state

	for _, state := range states {
		if state == nil {
			continue
		}
		if lastLineage == nil {
			lastLineage = state
		} else if lastLineage.Lineage != state.Lineage {
			logdiag.LogError(ctx, fmt.Errorf("lineage mismatch in state files: %s vs %s", lastLineage.String(), state.String()))
		}
	}

	if logdiag.HasError(ctx) {
		return ctx
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

// Guess engine based on local state alone and set it on context User-Agent as engine/direct or engine/terraform
// This guess might be overriden later when we pull remote state.
func GuessEngine(ctx context.Context, b *bundle.Bundle) context.Context {
	states := readLocalStates(ctx, b)

	if logdiag.HasError(ctx) {
		log.Warnf(ctx, "Error reading state")
		return ctx
	}

	winner := states[1]
	var isDirect bool

	if winner == nil {
		// no local or remote state; set b.DirectDeployment based on env vars
		var err error
		isDirect, err = getDirectDeploymentEnv(ctx)
		if err != nil {
			logdiag.LogError(ctx, err)
			return ctx
		}
	} else {
		isDirect = winner.isDirect
	}

	engine := "direct"

	if !isDirect {
		engine = "terraform"
	}

	log.Warnf(ctx, "Setting engine=%v", engine)
	return useragent.InContext(ctx, "engine", engine)
}

func readStates(ctx context.Context, b *bundle.Bundle) []*state {
	remotePathDirect, localPathDirect := b.StateFilenameDirect(ctx)
	remotePathTerraform, localPathTerraform := b.StateFilenameTerraform(ctx)

	f, err := deploy.StateFiler(b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}

	var wg sync.WaitGroup
	var directLocalState, terraformLocalState *state
	var directRemoteState, terraformRemoteState *state

	wg.Go(func() {
		directRemoteState = filerRead(ctx, f, remotePathDirect, true)
	})

	wg.Go(func() {
		terraformRemoteState = filerRead(ctx, f, remotePathTerraform, false)
	})

	wg.Go(func() {
		directLocalState = localRead(ctx, localPathDirect, true)
	})

	wg.Go(func() {
		terraformLocalState = localRead(ctx, localPathTerraform, false)
	})

	wg.Wait()

	// find highest serial across all state files
	// sorting is stable, so initial setting represents preference:
	states := []*state{terraformRemoteState, terraformLocalState, directRemoteState, directLocalState}
	sortStates(states)
	return states
}

func sortStates(states []*state) {
	slices.SortStableFunc(states, func(a, b *state) int {
		if b == nil && a == nil {
			return 0
		}
		// put nils first
		if a == nil {
			return -1
		}
		if b == nil {
			return 1
		}
		// otherwise sort by serial
		return int(a.Serial - b.Serial)
	})
}

func readLocalStates(ctx context.Context, b *bundle.Bundle) []*state {
	_, localPathDirect := b.StateFilenameDirect(ctx)
	_, localPathTerraform := b.StateFilenameTerraform(ctx)

	var wg sync.WaitGroup
	var directLocalState, terraformLocalState *state

	wg.Go(func() {
		directLocalState = localRead(ctx, localPathDirect, true)
	})

	wg.Go(func() {
		terraformLocalState = localRead(ctx, localPathTerraform, false)
	})

	wg.Wait()

	// find highest serial across all state files
	// sorting is stable, so initial setting represents preference:
	states := []*state{terraformLocalState, directLocalState}
	sortStates(states)
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
