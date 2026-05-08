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

// stateFile stores per-profile lakebox defaults on the local filesystem.
// Located at ~/.databricks/lakebox.json.
type stateFile struct {
	// Profile name → default lakebox ID.
	Defaults map[string]string `json:"defaults"`
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
	state.Defaults[profile] = lakeboxID
	return saveState(ctx, state)
}

func clearDefault(ctx context.Context, profile string) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}
	delete(state.Defaults, profile)
	return saveState(ctx, state)
}
