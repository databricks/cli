package io

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/databricks/cli/libs/filer"
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
	State StateType `json:"state"`
	Data  any       `json:"data,omitempty"`
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

	dataMap, ok := ps.Data.(map[string]any)
	if !ok {
		return nil, errors.New("invalid validated state data")
	}

	validatedAtStr, ok := dataMap["validated_at"].(string)
	if !ok {
		return nil, errors.New("missing validated_at in state data")
	}

	validatedAt, err := time.Parse(time.RFC3339, validatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid validated_at format: %w", err)
	}

	checksum, ok := dataMap["checksum"].(string)
	if !ok {
		return nil, errors.New("missing checksum in state data")
	}

	return &ValidatedData{
		ValidatedAt: validatedAt,
		Checksum:    checksum,
	}, nil
}

func (ps *ProjectState) extractChecksumFromMap() (string, bool) {
	dataMap, ok := ps.Data.(map[string]any)
	if !ok {
		return "", false
	}
	checksum, ok := dataMap["checksum"].(string)
	return checksum, ok
}

func (ps *ProjectState) Deploy() (*ProjectState, error) {
	if !ps.CanTransitionTo(StateDeployed) {
		if ps.State == StateScaffolded {
			return nil, errors.New("cannot deploy: project not validated")
		}
		if ps.State == StateDeployed {
			return nil, errors.New("cannot deploy: project already deployed (re-validate first)")
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
	case StateScaffolded:
		return "", false
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

func LoadState(ctx context.Context, workDir string) (*ProjectState, error) {
	f, err := filer.NewLocalClient(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create filer: %w", err)
	}

	r, err := f.Read(ctx, StateFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.As(err, &filer.FileDoesNotExistError{}) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file content: %w", err)
	}

	var state ProjectState
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

func SaveState(ctx context.Context, workDir string, state *ProjectState) error {
	f, err := filer.NewLocalClient(workDir)
	if err != nil {
		return fmt.Errorf("failed to create filer: %w", err)
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	if err := f.Write(ctx, StateFileName, bytes.NewReader(content), filer.OverwriteIfExists); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func ComputeChecksum(ctx context.Context, workDir string) (string, error) {
	f, err := filer.NewLocalClient(workDir)
	if err != nil {
		return "", fmt.Errorf("failed to create filer: %w", err)
	}

	var filesToHash []string

	for _, dir := range []string{"client", "server"} {
		// Check if directory exists
		info, err := f.Stat(ctx, dir)
		if err == nil && info.IsDir() {
			if err := collectSourceFiles(ctx, f, dir, &filesToHash); err != nil {
				return "", err
			}
		}
	}

	packageJSON := "package.json"
	if _, err := f.Stat(ctx, packageJSON); err == nil {
		filesToHash = append(filesToHash, packageJSON)
	}

	sort.Strings(filesToHash)

	if len(filesToHash) == 0 {
		return "", errors.New("no files to hash - project structure appears invalid")
	}

	hasher := sha256.New()

	for _, filePath := range filesToHash {
		r, err := f.Read(ctx, filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		if _, err := io.Copy(hasher, r); err != nil {
			r.Close()
			return "", fmt.Errorf("failed to hash %s: %w", filePath, err)
		}
		r.Close()
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func VerifyChecksum(ctx context.Context, workDir, expected string) (bool, error) {
	current, err := ComputeChecksum(ctx, workDir)
	if err != nil {
		return false, err
	}
	return current == expected, nil
}

func collectSourceFiles(ctx context.Context, f filer.Filer, dir string, files *[]string) error {
	entries, err := f.ReadDir(ctx, dir)
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
		// Use path.Join for forward slashes in relative paths (filer compatible)
		// filepath.Join might use backslashes on Windows, but here we are in CLI running on darwin.
		// Ideally use path.Join for abstract filesystem paths.
		relativePath := path.Join(dir, entry.Name())

		if entry.IsDir() {
			if excludedDirs[entry.Name()] {
				continue
			}
			if err := collectSourceFiles(ctx, f, relativePath, files); err != nil {
				return err
			}
		} else {
			ext := filepath.Ext(entry.Name())
			if validExtensions[ext] {
				*files = append(*files, relativePath)
			}
		}
	}

	return nil
}
