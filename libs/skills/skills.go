// Package skills lists and reads Databricks skills from any source that serves
// a skills release: a published release, a bundled snapshot or a local folder.
package skills

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrUnknownSkill reports a skill name the catalog does not list.
	ErrUnknownSkill = errors.New("unknown skill")

	// ErrUnknownFile reports a file the catalog does not list for the skill.
	ErrUnknownFile = errors.New("unknown skill file")

	// ErrSourceUnreachable reports a source whose release cannot be read.
	ErrSourceUnreachable = errors.New("skills source unreachable")
)

// Skill is one skill in a catalog.
type Skill struct {
	Name        string
	Description string
	Version     string

	// Files lists the skill's files relative to the skill directory, SKILL.md included.
	Files []string

	// Commands lists the CLI command groups the skill covers.
	Commands []string

	Experimental bool
}

// Catalog is the set of skills a source serves.
type Catalog struct {
	// Skills are sorted by name.
	Skills []Skill

	// Version is the release version.
	Version string

	// Label says where the catalog came from.
	Label string

	FetchedAt time.Time
	Stale     bool

	// Rules is text shown above the catalog, empty when the release has none.
	Rules string
}

// Source serves skills. Callers list skills and their files from the catalog,
// and read file contents with ReadFile.
type Source interface {
	Catalog(ctx context.Context) (*Catalog, error)
	ReadFile(ctx context.Context, skill, path string) ([]byte, error)
}
