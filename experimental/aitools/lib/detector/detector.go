// Package detector provides project context detection for Databricks aitools.
package detector

import (
	"context"
)

// BundleInfo contains information about a detected Databricks bundle.
type BundleInfo struct {
	Name    string
	Target  string
	RootDir string
}

// DetectedContext represents the detected project context.
type DetectedContext struct {
	InProject   bool
	TargetTypes []string // ["apps", "jobs"] - supports combined bundles
	Template    string   // "appkit-typescript", "python", etc.
	BundleInfo  *BundleInfo
	Metadata    map[string]string
	IsAppOnly   bool // True if project contains only app resources, no jobs/pipelines/etc.
}

// Detector detects project context from a working directory.
type Detector interface {
	// Detect examines the working directory and updates the context.
	Detect(ctx context.Context, workDir string, detected *DetectedContext) error
}

// Registry manages a collection of detectors.
type Registry struct {
	detectors []Detector
}

// NewRegistry creates a new detector registry with default detectors.
func NewRegistry() *Registry {
	return &Registry{
		detectors: []Detector{
			&BundleDetector{},
			&TemplateDetector{},
		},
	}
}

// Detect runs all detectors and returns the combined context.
func (r *Registry) Detect(ctx context.Context, workDir string) *DetectedContext {
	detected := &DetectedContext{
		Metadata: make(map[string]string),
	}

	for _, d := range r.detectors {
		// ignore errors - detectors should be resilient
		_ = d.Detect(ctx, workDir, detected)
	}

	return detected
}
