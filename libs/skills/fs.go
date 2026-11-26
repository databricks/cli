package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"time"
)

const manifestFile = "manifest.json"

type fsSource struct {
	fsys  fs.FS
	label string
}

// NewFSSource returns a Source over fsys, laid out like a skills release:
// manifest.json at the root, and each skill's files under its repo_dir.
func NewFSSource(fsys fs.FS, label string) Source {
	return &fsSource{fsys: fsys, label: label}
}

func (s *fsSource) Catalog(_ context.Context) (*Catalog, error) {
	m, err := s.manifest()
	if err != nil {
		return nil, err
	}
	catalog := &Catalog{Version: m.Version, Label: s.label, FetchedAt: time.Now()}
	if m.CatalogRules != "" {
		data, err := fs.ReadFile(s.fsys, m.CatalogRules)
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

func (s *fsSource) ReadFile(_ context.Context, skill, file string) ([]byte, error) {
	name, err := s.filePath(skill, file)
	if err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(s.fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read %q in skill %q: %w", file, skill, err)
	}
	return data, nil
}

func (s *fsSource) FileSize(_ context.Context, skill, file string) (int64, error) {
	name, err := s.filePath(skill, file)
	if err != nil {
		return 0, err
	}
	info, err := fs.Stat(s.fsys, name)
	if err != nil {
		return 0, fmt.Errorf("stat %q in skill %q: %w", file, skill, err)
	}
	return info.Size(), nil
}

// filePath returns the path of a skill file in the release. The manifest must
// list the file for the skill.
func (s *fsSource) filePath(skill, file string) (string, error) {
	m, err := s.manifest()
	if err != nil {
		return "", err
	}
	meta, ok := m.Skills[skill]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownSkill, skill)
	}
	if !slices.Contains(meta.Files, file) {
		return "", fmt.Errorf("%w: %q in skill %q", ErrUnknownFile, file, skill)
	}
	return path.Join(meta.RepoDir, skill, file), nil
}

// manifest reads and checks the release manifest. Every path it names, the
// skill files, repo_dir and catalog_rules, must stay inside the release.
func (s *fsSource) manifest() (*Manifest, error) {
	data, err := fs.ReadFile(s.fsys, manifestFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSourceUnreachable, err)
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
