package skills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"time"
)

const manifestFile = "manifest.json"

type fsSource struct {
	read  func(ctx context.Context, name string) ([]byte, error)
	label string
}

// NewFSSource returns a Source over fsys, laid out like a skills release:
// manifest.json at the root, and each skill's files under its repo_dir. The
// label names where the release comes from; its backslashes become forward
// slashes, so a Windows folder reads the same as on other hosts.
func NewFSSource(fsys fs.FS, label string) Source {
	return &fsSource{read: func(_ context.Context, name string) ([]byte, error) {
		return fs.ReadFile(fsys, name)
	}, label: strings.ReplaceAll(label, `\`, "/")}
}

func (s *fsSource) Catalog(ctx context.Context) (*Catalog, error) {
	m, err := s.manifest(ctx)
	if err != nil {
		return nil, err
	}
	catalog := &Catalog{Version: m.Version, Label: s.label, FetchedAt: time.Now()}
	if m.CatalogRules != "" {
		data, err := s.read(ctx, m.CatalogRules)
		if err != nil {
			return nil, fmt.Errorf("read catalog_rules: %w", err)
		}
		catalog.Rules = string(data)
	}
	for _, name := range slices.Sorted(maps.Keys(m.Skills)) {
		meta := m.Skills[name]
		catalog.Skills = append(catalog.Skills, Skill{
			Name:         name,
			Description:  meta.Description,
			Version:      meta.Version,
			MinCLIVer:    meta.MinCLIVer,
			Files:        meta.Files,
			Commands:     meta.Commands,
			Experimental: meta.IsExperimental(),
		})
	}
	return catalog, nil
}

func (s *fsSource) ReadFile(ctx context.Context, skill, file string) ([]byte, error) {
	m, err := s.manifest(ctx)
	if err != nil {
		return nil, err
	}
	meta, ok := m.Skills[skill]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSkill, skill)
	}
	if !slices.Contains(meta.Files, file) {
		return nil, fmt.Errorf("%w: %q in skill %q", ErrUnknownFile, file, skill)
	}
	data, err := s.read(ctx, path.Join(meta.RepoDir, skill, file))
	if err != nil {
		return nil, fmt.Errorf("read %q in skill %q: %w", file, skill, err)
	}
	return data, nil
}

// manifest reads and checks the release manifest. Every path it names, the
// skill files, repo_dir and catalog_rules, must stay inside the release.
func (s *fsSource) manifest(ctx context.Context) (*Manifest, error) {
	data, err := s.read(ctx, manifestFile)
	if errors.Is(err, ErrManifestTooLarge) || errors.Is(err, errReleaseNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, unreachable(err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", manifestFile, err)
	}
	NormalizeManifest(&m)
	if err := CheckManifestPaths(&m); err != nil {
		return nil, err
	}
	for _, name := range slices.Sorted(maps.Keys(m.Skills)) {
		if repoDir := m.Skills[name].RepoDir; escapesDir(repoDir) {
			return nil, fmt.Errorf("%w: skill %q has repo_dir %q", ErrUnsafeManifestPath, name, repoDir)
		}
	}
	if m.CatalogRules != "" {
		// Clean the path the way skill files are cleaned by path.Join, so that
		// "./RULES.md" and "docs/../RULES.md" are read like "RULES.md".
		m.CatalogRules = path.Clean(m.CatalogRules)
	}
	// "docs/.." cleans to ".", a directory that cannot be read as rules.
	if m.CatalogRules == "." || escapesDir(m.CatalogRules) {
		return nil, fmt.Errorf("%w: catalog_rules %q", ErrUnsafeManifestPath, m.CatalogRules)
	}
	return &m, nil
}
