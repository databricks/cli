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
			Name:          name,
			Description:   meta.Description,
			Version:       meta.Version,
			Files:         meta.Files,
			Commands:      meta.Commands,
			MinCLIVersion: meta.MinCLIVer,
			Experimental:  meta.IsExperimental(),
		})
	}
	return catalog, nil
}

func (s *fsSource) ReadFile(_ context.Context, skill, file string) ([]byte, error) {
	m, err := s.manifest()
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
	data, err := fs.ReadFile(s.fsys, path.Join(meta.RepoDir, skill, file))
	if err != nil {
		return nil, fmt.Errorf("read %q in skill %q: %w", file, skill, err)
	}
	return data, nil
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
	if err := CheckManifest(&m); err != nil {
		return nil, err
	}
	if m.CatalogRules != "" {
		// Clean the path the way skill files are cleaned by path.Join, so that
		// "./RULES.md" and "docs/../RULES.md" are read like "RULES.md".
		m.CatalogRules = path.Clean(m.CatalogRules)
	}
	return &m, nil
}
