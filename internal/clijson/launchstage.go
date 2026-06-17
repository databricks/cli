package clijson

import (
	"fmt"
	"slices"
)

// LaunchStage is a field, enum value, or type's release stage, carried by the
// contract's launch_stage. Only the constants below are valid.
//
// The producer currently emits launch_stage as a plain string, so SchemaJSON
// and SchemaFieldJSON store it as such and ParseLaunchStage enforces the closed
// set where the contract is consumed. Once the producer is upstreamed to emit
// this enum directly, no other value could reach the CLI.
type LaunchStage string

const (
	LaunchStageGA             LaunchStage = "GA"
	LaunchStagePublicPreview  LaunchStage = "PUBLIC_PREVIEW"
	LaunchStagePublicBeta     LaunchStage = "PUBLIC_BETA"
	LaunchStagePrivatePreview LaunchStage = "PRIVATE_PREVIEW"
)

// LaunchStages is the closed set of valid launch stages. ParseLaunchStage
// validates against it, so extending the contract with a new stage means adding
// it here too.
var LaunchStages = []LaunchStage{
	LaunchStageGA,
	LaunchStagePublicPreview,
	LaunchStagePublicBeta,
	LaunchStagePrivatePreview,
}

// ParseLaunchStage converts a raw launch_stage string from the contract into a
// LaunchStage, mapping the empty string to GA (the implicit default). It errors
// on any value outside the closed set.
func ParseLaunchStage(s string) (LaunchStage, error) {
	if s == "" {
		return LaunchStageGA, nil
	}
	stage := LaunchStage(s)
	if !slices.Contains(LaunchStages, stage) {
		return "", fmt.Errorf("unknown launch stage %q: add it to LaunchStages in internal/clijson/launchstage.go", s)
	}
	return stage, nil
}
