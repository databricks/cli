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
	"time"

	"github.com/databricks/cli/libs/log"
)

const manifestFile = "manifest.json"

type fsSource struct {
	read func(ctx context.Context, name string) ([]byte, error)
	// stat reports the info of a file in the release. It is nil when the
	// source cannot stat its files, as for a release fetched over HTTP.
	stat  func(name string) (fs.FileInfo, error)
	label string
}

// NewFSSource returns a Source over fsys, laid out like a skills release:
// manifest.json at the root, and each skill's files under its repo_dir. The
// label names where the release comes from, and is kept as given. A caller
// that has a host path converts it first, as Resolve does with filepath.ToSlash.
func NewFSSource(fsys fs.FS, label string) Source {
	return &fsSource{read: func(_ context.Context, name string) ([]byte, error) {
		return fs.ReadFile(fsys, name)
	}, stat: func(name string) (fs.FileInfo, error) {
		return fs.Stat(fsys, name)
	}, label: label}
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
		if !IsSkillName(name) {
			// The installer rejects this name, so the catalog leaves the skill out.
			log.Debugf(ctx, "skills catalog: skipping skill %q: the name is not a valid skill name", name)
			continue
		}
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
	name, err := s.filePath(ctx, skill, file)
	if err != nil {
		return nil, err
	}
	data, err := s.read(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("read %q in skill %q: %w", file, skill, err)
	}
	return data, nil
}

// FileSize returns the size of a skill file. It returns ErrNoSize for a folder,
// and for a source that cannot stat its files.
func (s *fsSource) FileSize(ctx context.Context, skill, file string) (int64, error) {
	name, err := s.filePath(ctx, skill, file)
	if err != nil {
		return 0, err
	}
	if s.stat == nil {
		return 0, fmt.Errorf("%w: %q in skill %q", ErrNoSize, file, skill)
	}
	info, err := s.stat(name)
	if err != nil {
		return 0, fmt.Errorf("stat %q in skill %q: %w", file, skill, err)
	}
	if info.IsDir() {
		return 0, fmt.Errorf("%w: %q in skill %q is a folder", ErrNoSize, file, skill)
	}
	return info.Size(), nil
}

// filePath returns the path of a skill file in the release. The manifest must
// list the file for the skill, and a name IsSkillName rejects is unknown, as it
// is to Catalog.
func (s *fsSource) filePath(ctx context.Context, skill, file string) (string, error) {
	m, err := s.manifest(ctx)
	if err != nil {
		return "", err
	}
	meta, ok := m.Skills[skill]
	if !ok || !IsSkillName(skill) {
		return "", fmt.Errorf("%w: %q", ErrUnknownSkill, skill)
	}
	if !slices.Contains(meta.Files, file) {
		return "", fmt.Errorf("%w: %q in skill %q", ErrUnknownFile, file, skill)
	}
	return path.Join(meta.RepoDir, skill, file), nil
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
