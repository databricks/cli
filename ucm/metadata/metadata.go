// Package metadata models the ucm deployment-provenance blob that is written
// next to ucm-state.json after a successful deploy. It parallels
// bundle/metadata for DAB; the ucm shape is intentionally narrower and only
// captures identity + CLI build info — no per-resource annotations and no git
// details (ucm doesn't track git yet).
package metadata

import "time"

// Version is bumped on incompatible changes to the on-wire Metadata shape.
const Version = 1

// Metadata is the ucm-side deployment-provenance record, written to the
// remote state dir as ucm-metadata.json after a successful deploy.
type Metadata struct {
	Version    int       `json:"version"`
	CliVersion string    `json:"cli_version"`
	Ucm        UcmMeta   `json:"ucm"`
	// DeploymentID is the terraform-engine deployment ID read from the local
	// ucm-state.json cache. Empty on direct-engine deploys until the direct
	// State grows an ID field. Tracked in follow-up (note issue #83).
	DeploymentID string    `json:"deployment_id,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// UcmMeta captures the subset of ucm.Ucm fields that are stable enough to
// serialise into a metadata blob downstream consumers may rely on.
type UcmMeta struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}
