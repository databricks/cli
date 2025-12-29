package state

import (
	"errors"
	"time"
)

// StateType represents the project lifecycle state
type StateType string

const (
	StateScaffolded StateType = "scaffolded"
	StateValidated  StateType = "validated"
	StateDeployed   StateType = "deployed"
)

// ProjectState represents the current state of a project
type ProjectState struct {
	State       StateType  `json:"state"`
	ValidatedAt *time.Time `json:"validated_at,omitempty"`
	Checksum    string     `json:"checksum,omitempty"`
	DeployedAt  *time.Time `json:"deployed_at,omitempty"`
}

// NewScaffolded creates a new scaffolded state
func NewScaffolded() *ProjectState {
	return &ProjectState{State: StateScaffolded}
}

// Validate transitions the state to validated
func (s *ProjectState) Validate(checksum string) *ProjectState {
	now := time.Now().UTC()
	return &ProjectState{
		State:       StateValidated,
		ValidatedAt: &now,
		Checksum:    checksum,
	}
}

// Deploy transitions the state to deployed
func (s *ProjectState) Deploy() (*ProjectState, error) {
	switch s.State {
	case StateValidated:
		now := time.Now().UTC()
		return &ProjectState{
			State:       StateDeployed,
			ValidatedAt: s.ValidatedAt,
			Checksum:    s.Checksum,
			DeployedAt:  &now,
		}, nil
	case StateScaffolded:
		return nil, errors.New("cannot deploy: project not validated (run validate first)")
	case StateDeployed:
		return nil, errors.New("cannot deploy: project already deployed (run validate to re-deploy)")
	default:
		return nil, errors.New("cannot deploy: unknown state")
	}
}

// IsValidated returns true if the project has been validated
func (s *ProjectState) IsValidated() bool {
	return s.State == StateValidated || s.State == StateDeployed
}

// GetChecksum returns the checksum if available
func (s *ProjectState) GetChecksum() string {
	return s.Checksum
}
