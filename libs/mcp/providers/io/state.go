package io

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/appdotbuild/go-mcp/pkg/fileutil"
	"github.com/zeebo/blake3"
)

const StateFileName = ".edda_state"

// StateType represents the lifecycle state of a scaffolded project.
type StateType string

const (
	StateScaffolded StateType = "Scaffolded"
	StateValidated  StateType = "Validated"
	StateDeployed   StateType = "Deployed"
)

// String returns the string representation of the state.
func (s StateType) String() string {
	return string(s)
}

// IsValid checks if the state type is a valid value.
func (s StateType) IsValid() bool {
	switch s {
	case StateScaffolded, StateValidated, StateDeployed:
		return true
	default:
		return false
	}
}

// ValidatedData contains metadata for a validated project state.
type ValidatedData struct {
	ValidatedAt time.Time `json:"validated_at"`
	Checksum    string    `json:"checksum"`
}

// DeployedData contains metadata for a deployed project state.
type DeployedData struct {
	ValidatedAt time.Time `json:"validated_at"`
	Checksum    string    `json:"checksum"`
	DeployedAt  time.Time `json:"deployed_at"`
}

// ProjectState tracks the current state and metadata of a scaffolded project.
type ProjectState struct {
	State StateType   `json:"state"`
	Data  interface{} `json:"data,omitempty"`
}

func NewProjectState() *ProjectState {
	return &ProjectState{
		State: StateScaffolded,
	}
}

func (ps *ProjectState) Validate(checksum string) *ProjectState {
	return &ProjectState{
		State: StateValidated,
		Data: ValidatedData{
			ValidatedAt: time.Now().UTC(),
			Checksum:    checksum,
		},
	}
}

func (ps *ProjectState) extractValidatedData() (*ValidatedData, error) {
	if data, ok := ps.Data.(ValidatedData); ok {
		return &data, nil
	}

	dataMap, ok := ps.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid validated state data")
	}

	validatedAtStr, ok := dataMap["validated_at"].(string)
	if !ok {
		return nil, fmt.Errorf("missing validated_at in state data")
	}

	validatedAt, err := time.Parse(time.RFC3339, validatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid validated_at format: %w", err)
	}

	checksum, ok := dataMap["checksum"].(string)
	if !ok {
		return nil, fmt.Errorf("missing checksum in state data")
	}

	return &ValidatedData{
		ValidatedAt: validatedAt,
		Checksum:    checksum,
	}, nil
}

func (ps *ProjectState) extractChecksumFromMap() (string, bool) {
	dataMap, ok := ps.Data.(map[string]interface{})
	if !ok {
		return "", false
	}
	checksum, ok := dataMap["checksum"].(string)
	return checksum, ok
}

func (ps *ProjectState) Deploy() (*ProjectState, error) {
	if !ps.CanTransitionTo(StateDeployed) {
		if ps.State == StateScaffolded {
			return nil, fmt.Errorf("cannot deploy: project not validated")
		}
		if ps.State == StateDeployed {
			return nil, fmt.Errorf("cannot deploy: project already deployed (re-validate first)")
		}
		return nil, fmt.Errorf("invalid state transition: %s -> Deployed", ps.State)
	}

	data, err := ps.extractValidatedData()
	if err != nil {
		return nil, err
	}

	return &ProjectState{
		State: StateDeployed,
		Data: DeployedData{
			ValidatedAt: data.ValidatedAt,
			Checksum:    data.Checksum,
			DeployedAt:  time.Now().UTC(),
		},
	}, nil
}

func (ps *ProjectState) Checksum() (string, bool) {
	switch ps.State {
	case StateValidated:
		if data, ok := ps.Data.(ValidatedData); ok {
			return data.Checksum, true
		}
		return ps.extractChecksumFromMap()
	case StateDeployed:
		if data, ok := ps.Data.(DeployedData); ok {
			return data.Checksum, true
		}
		return ps.extractChecksumFromMap()
	}
	return "", false
}

func (ps *ProjectState) IsValidated() bool {
	return ps.State == StateValidated || ps.State == StateDeployed
}

// CanTransitionTo checks if a state transition is valid according to the state machine rules.
// Valid transitions:
// - Scaffolded -> Validated
// - Validated -> Deployed (or re-validate to Validated)
// - Deployed -> Validated (re-validation allowed before re-deployment)
func (ps *ProjectState) CanTransitionTo(next StateType) bool {
	switch ps.State {
	case StateScaffolded:
		return next == StateValidated
	case StateValidated:
		return next == StateDeployed || next == StateValidated
	case StateDeployed:
		return next == StateValidated
	default:
		return false
	}
}

// TransitionTo attempts to transition to a new state, returning an error if invalid.
func (ps *ProjectState) TransitionTo(next StateType) error {
	if !next.IsValid() {
		return fmt.Errorf("invalid target state: %s", next)
	}
	if !ps.CanTransitionTo(next) {
		return fmt.Errorf("invalid state transition: %s -> %s", ps.State, next)
	}
	ps.State = next
	return nil
}

func LoadState(workDir string) (*ProjectState, error) {
	statePath := filepath.Join(workDir, StateFileName)

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return nil, nil
	}

	content, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ProjectState
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

func SaveState(workDir string, state *ProjectState) error {
	statePath := filepath.Join(workDir, StateFileName)

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	if err := fileutil.AtomicWriteFile(statePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func ComputeChecksum(workDir string) (string, error) {
	var filesToHash []string

	for _, dir := range []string{"client", "server"} {
		dirPath := filepath.Join(workDir, dir)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			if err := collectSourceFiles(dirPath, &filesToHash); err != nil {
				return "", err
			}
		}
	}

	packageJSON := filepath.Join(workDir, "package.json")
	if _, err := os.Stat(packageJSON); err == nil {
		filesToHash = append(filesToHash, packageJSON)
	}

	sort.Strings(filesToHash)

	if len(filesToHash) == 0 {
		return "", fmt.Errorf("no files to hash - project structure appears invalid")
	}

	hasher := blake3.New()

	for _, filePath := range filesToHash {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", filePath, err)
		}
		hasher.Write(content)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func VerifyChecksum(workDir string, expected string) (bool, error) {
	current, err := ComputeChecksum(workDir)
	if err != nil {
		return false, err
	}
	return current == expected, nil
}

func collectSourceFiles(dir string, files *[]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	excludedDirs := map[string]bool{
		"node_modules": true,
		"dist":         true,
		".git":         true,
		"build":        true,
		"coverage":     true,
	}

	validExtensions := map[string]bool{
		".ts":   true,
		".tsx":  true,
		".js":   true,
		".jsx":  true,
		".json": true,
		".css":  true,
		".html": true,
		".yaml": true,
		".yml":  true,
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if excludedDirs[entry.Name()] {
				continue
			}
			if err := collectSourceFiles(path, files); err != nil {
				return err
			}
		} else {
			ext := filepath.Ext(path)
			if validExtensions[ext] {
				*files = append(*files, path)
			}
		}
	}

	return nil
}
