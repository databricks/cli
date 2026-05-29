package lakebox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

// stateFile stores per-profile lakebox state on the local filesystem.
// Located at ~/.databricks/lakebox.json.
type stateFile struct {
	// Profile name → default lakebox ID.
	Defaults map[string]string `json:"defaults"`
	// Profile name → SSH gateway hostname returned by the manager for any
	// sandbox in that workspace. Cached so `ssh <id>` does not need to fetch
	// the sandbox just to learn where to connect. Empty until the first
	// command that reads a sandbox response populates it.
	GatewayHosts map[string]string `json:"gatewayHosts,omitempty"`
	// Profile name → cached (id, name) pairs, used purely for local
	// name-based resolution (`lakebox ssh my-project` finds the matching
	// sandbox ID without an extra API call). Refreshed in full by
	// `lakebox list`; mutated in-place by `create`, `config --name`,
	// `delete`, and `status`. Cache misses fall through to treating the
	// argument as an ID, so this never *blocks* an operation — it only
	// adds the name-to-ID shortcut.
	Sandboxes map[string][]cachedSandbox `json:"sandboxes,omitempty"`
}

// cachedSandbox is the minimal (id, name) pair we need to resolve a
// user-typed name to a wire ID without calling the server.
type cachedSandbox struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

func stateFilePath(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "lakebox.json"), nil
}

func loadState(ctx context.Context) (*stateFile, error) {
	path, err := stateFilePath(ctx)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &stateFile{Defaults: make(map[string]string)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var state stateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	if state.Defaults == nil {
		state.Defaults = make(map[string]string)
	}
	return &state, nil
}

func saveState(ctx context.Context, state *stateFile) error {
	path, err := stateFilePath(ctx)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func getDefault(ctx context.Context, profile string) string {
	state, err := loadState(ctx)
	if err != nil {
		return ""
	}
	return state.Defaults[profile]
}

func setDefault(ctx context.Context, profile, lakeboxID string) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	if state.Defaults[profile] == lakeboxID {
		return nil
	}
	state.Defaults[profile] = lakeboxID
	return saveState(ctx, state)
}

func clearDefault(ctx context.Context, profile string) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	if _, ok := state.Defaults[profile]; !ok {
		return nil
	}
	delete(state.Defaults, profile)
	return saveState(ctx, state)
}

// getGatewayHost returns the cached SSH gateway hostname for the workspace
// behind `profile`, or "" if nothing has been cached yet.
func getGatewayHost(ctx context.Context, profile string) string {
	state, err := loadState(ctx)
	if err != nil {
		return ""
	}
	return state.GatewayHosts[profile]
}

// setGatewayHost caches the SSH gateway hostname for `profile`. No-op when
// `host` is empty or already equal to the cached value, so callers can pipe
// every Sandbox response through here without churning the state file.
func setGatewayHost(ctx context.Context, profile, host string) error {
	if host == "" {
		return nil
	}
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	if state.GatewayHosts[profile] == host {
		return nil
	}
	if state.GatewayHosts == nil {
		state.GatewayHosts = make(map[string]string)
	}
	state.GatewayHosts[profile] = host
	return saveState(ctx, state)
}

// getSandboxes returns the locally-cached (id, name) pairs for `profile`,
// or nil if nothing has been cached yet.
func getSandboxes(ctx context.Context, profile string) []cachedSandbox {
	state, err := loadState(ctx)
	if err != nil {
		return nil
	}
	return state.Sandboxes[profile]
}

// setSandboxes replaces the cached sandbox list for `profile`. Used by
// `lakebox list` to keep the cache in sync with the server's view.
// No-op when the new list is byte-for-byte identical to the cached one,
// to avoid rewriting the state file on every `list` call.
func setSandboxes(ctx context.Context, profile string, sbs []cachedSandbox) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	if sandboxesEqual(state.Sandboxes[profile], sbs) {
		return nil
	}
	if state.Sandboxes == nil {
		state.Sandboxes = make(map[string][]cachedSandbox)
	}
	if sbs == nil {
		sbs = []cachedSandbox{}
	}
	state.Sandboxes[profile] = sbs
	return saveState(ctx, state)
}

// upsertSandbox adds or updates a single cached (id, name) entry for
// `profile`. Use this when a single command observes one sandbox's
// state (create, status, config), so the cache stays warm without
// requiring a full `list`.
func upsertSandbox(ctx context.Context, profile, id, name string) error {
	if id == "" {
		return nil
	}
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	existing := state.Sandboxes[profile]
	for i, s := range existing {
		if s.ID == id {
			if s.Name == name {
				return nil // no change
			}
			existing[i].Name = name
			if state.Sandboxes == nil {
				state.Sandboxes = make(map[string][]cachedSandbox)
			}
			state.Sandboxes[profile] = existing
			return saveState(ctx, state)
		}
	}
	if state.Sandboxes == nil {
		state.Sandboxes = make(map[string][]cachedSandbox)
	}
	state.Sandboxes[profile] = append(existing, cachedSandbox{ID: id, Name: name})
	return saveState(ctx, state)
}

// removeSandbox drops a single cached entry for `profile`. Called from
// `delete` so the cache doesn't keep referencing sandboxes that no
// longer exist server-side.
func removeSandbox(ctx context.Context, profile, id string) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	existing := state.Sandboxes[profile]
	for i, s := range existing {
		if s.ID == id {
			state.Sandboxes[profile] = append(existing[:i], existing[i+1:]...)
			return saveState(ctx, state)
		}
	}
	return nil
}

// sandboxesEqual reports whether two slices contain the same entries in
// the same order. Used by setSandboxes to short-circuit no-op writes.
func sandboxesEqual(a, b []cachedSandbox) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
