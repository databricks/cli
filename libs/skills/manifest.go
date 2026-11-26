package skills

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const (
	// StableRepoDir is the release directory of stable skills (the repo_dir default).
	StableRepoDir = "skills"

	// ExperimentalRepoDir is the release directory of experimental skills.
	ExperimentalRepoDir = "experimental"

	// StateFileName is the install state file the installer keeps beside the
	// skill directories. No skill may take this name.
	StateFileName = ".state.json"
)

// ErrUnsafeManifestPath reports a skills manifest that names a skill directory,
// a skill file or catalog_rules outside the release. Its message keeps the
// wording install has always printed. The command layer classifies it like a
// manifest that does not parse.
var ErrUnsafeManifestPath = errors.New("skills manifest names a path outside the skills directory")

// Manifest describes the skills manifest fetched from the skills repo.
type Manifest struct {
	Version   string               `json:"version"`
	UpdatedAt string               `json:"updated_at"`
	Skills    map[string]SkillMeta `json:"skills"`

	// CatalogRules names a file in the release, relative to its root, whose text
	// the skills catalog shows above the skill list (manifest field catalog_rules).
	CatalogRules string `json:"catalog_rules,omitempty"`
}

// SkillMeta describes a single skill entry in the manifest.
type SkillMeta struct {
	Version     string   `json:"version"`
	UpdatedAt   string   `json:"updated_at"`
	Files       []string `json:"files"`
	Description string   `json:"description,omitempty"`
	MinCLIVer   string   `json:"min_cli_version,omitempty"`

	// RepoDir is "skills" or "experimental" (manifest field repo_dir).
	RepoDir string `json:"repo_dir,omitempty"`

	// Commands lists the CLI command groups the skill covers (manifest field
	// commands). Each entry is a group's path below the root command, such as
	// "fs" or "bundle deployment"; the match is exact.
	Commands []string `json:"commands,omitempty"`

	// SourceName is the upstream skill directory name within RepoDir.
	// Set during normalization, not from JSON.
	SourceName string `json:"-"`
}

// IsExperimental reports whether the skill lives in the experimental directory.
func (s SkillMeta) IsExperimental() bool {
	return s.RepoDir == ExperimentalRepoDir
}

// NormalizeManifest stamps SourceName and defaults RepoDir for older manifests.
func NormalizeManifest(m *Manifest) {
	if m.Skills == nil {
		m.Skills = map[string]SkillMeta{}
	}
	for name, meta := range m.Skills {
		if meta.RepoDir == "" {
			meta.RepoDir = StableRepoDir
		}
		meta.SourceName = name
		m.Skills[name] = meta
	}
}

// CheckManifestPaths fails a manifest that names a skill or file outside its
// directory, or a skill named like the state file. The manifest comes from the
// network, so it must not be able to name ../../.bashrc, and only a broken or
// tampered manifest lists such an entry. The rule is the same on every OS.
// Entries that stay inside their directory but are not portable are left to
// the installer, which skips only that skill.
func CheckManifestPaths(m *Manifest) error {
	for _, name := range slices.Sorted(maps.Keys(m.Skills)) {
		if name == "" || name == "." || name == ".." || name == StateFileName || strings.ContainsAny(name, `/\`) {
			return fmt.Errorf("%w: skill name %q", ErrUnsafeManifestPath, name)
		}
		for _, file := range m.Skills[name].Files {
			if escapesDir(file) {
				return fmt.Errorf("%w: skill %q lists file %q", ErrUnsafeManifestPath, name, file)
			}
		}
	}
	return nil
}

// escapesDir reports whether file, joined onto a directory, names a path
// outside it on some OS: an absolute path, a Windows drive, or more ".."
// elements than the elements before them, with either slash as the separator.
func escapesDir(file string) bool {
	if strings.HasPrefix(file, "/") || strings.HasPrefix(file, `\`) || (len(file) >= 2 && file[1] == ':') {
		return true
	}
	depth := 0
	for _, elem := range strings.FieldsFunc(file, func(r rune) bool { return r == '/' || r == '\\' }) {
		switch elem {
		case ".":
		case "..":
			depth--
			if depth < 0 {
				return true
			}
		default:
			depth++
		}
	}
	return false
}
