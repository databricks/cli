package lakebox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// stateFile stores per-profile lakebox defaults on the local filesystem.
// Located at ~/.databricks/lakebox.json.
type stateFile struct {
	// Profile name → default lakebox ID.
	Defaults map[string]string `json:"defaults"`
}

func stateFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "lakebox.json"), nil
}

func loadState() (*stateFile, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &stateFile{Defaults: make(map[string]string)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var state stateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return &stateFile{Defaults: make(map[string]string)}, nil
	}
	if state.Defaults == nil {
		state.Defaults = make(map[string]string)
	}
	return &state, nil
}

func saveState(state *stateFile) error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func getDefault(profile string) string {
	state, err := loadState()
	if err != nil {
		return ""
	}
	return state.Defaults[profile]
}

func setDefault(profile, lakeboxID string) error {
	state, err := loadState()
	if err != nil {
		return err
	}
	state.Defaults[profile] = lakeboxID
	return saveState(state)
}

func clearDefault(profile string) error {
	state, err := loadState()
	if err != nil {
		return err
	}
	delete(state.Defaults, profile)
	return saveState(state)
}
