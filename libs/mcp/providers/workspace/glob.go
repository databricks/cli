package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
)

// GlobArgs contains arguments for glob operation
type GlobArgs struct {
	Pattern string `json:"pattern"`
}

// GlobResult contains the result of a glob operation
type GlobResult struct {
	Files []string `json:"files"`
	Total int      `json:"total"`
}

// Glob matches files against a pattern in the workspace
func (p *Provider) Glob(ctx context.Context, args *GlobArgs) (*GlobResult, error) {
	workDir, err := p.getWorkDir(ctx)
	if err != nil {
		return nil, err
	}

	// Resolve pattern relative to work dir
	pattern := filepath.Join(workDir, args.Pattern)

	// Use filepath.Glob
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob failed: %w", err)
	}

	// Convert to relative paths
	relMatches := make([]string, len(matches))
	for i, match := range matches {
		relPath, err := filepath.Rel(workDir, match)
		if err != nil {
			relPath = match
		}
		relMatches[i] = relPath
	}

	// Sort results
	sort.Strings(relMatches)

	return &GlobResult{
		Files: relMatches,
		Total: len(relMatches),
	}, nil
}
