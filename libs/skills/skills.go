// Package skills lists and reads Databricks skills from any source that serves
// a skills release: a published release, a bundled snapshot or a local folder.
//
// The catalog and source code is source-neutral. The release lookup in ref.go
// and the fetch helpers in fetch.go are the only code here that talks to GitHub
// or reads DATABRICKS_SKILLS_REF and DATABRICKS_SKILLS_BASE_URL. They sit in
// this package for now, next to the installer code that already uses them.
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
	MinCLIVer   string

	// Files lists the skill's files relative to the skill directory, SKILL.md included.
	Files []string

	// Commands lists the CLI command groups the skill covers.
	Commands []string

	Experimental bool
}

// Catalog is the set of skills a source serves.
type Catalog struct {
	// Skills are sorted by name. Install also skips non-portable names and
	// paths, which the catalog does not check yet.
	Skills []Skill

	// Version is the release version.
	Version string

	// Label says where the catalog came from.
	Label string

	// FetchedAt is when the source produced the catalog. A source that serves
	// a cached copy sets it to the time the copy was fetched. Zero means the
	// time is unknown, as for the bundled copy.
	FetchedAt time.Time

	// Stale reports that the catalog is not the current release. The source
	// served a cached copy it could not refresh, or it is the bundled copy,
	// which is only served when the release cannot be reached. Sources that
	// always read the release themselves leave it false.
	Stale bool

	// Rules is text shown above the catalog, empty when the release has none.
	Rules string
}

// Source serves skills. Callers list skills and their files from the catalog,
// and read file contents with ReadFile.
type Source interface {
	Catalog(ctx context.Context) (*Catalog, error)
	ReadFile(ctx context.Context, skill, path string) ([]byte, error)
}
