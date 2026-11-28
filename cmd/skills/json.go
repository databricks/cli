package skills

import (
	"path/filepath"
	"strings"
	"time"

	libskills "github.com/databricks/cli/libs/skills"
)

// catalogJSON is the JSON output of skills list and skills search. Both commands
// print this envelope and the same per-skill fields, so one parser reads both.
// The fields are chosen here rather than by marshaling libskills.Catalog, so
// the output does not change when the catalog's fields do.
type catalogJSON struct {
	Source         string             `json:"source"`
	ReleaseVersion string             `json:"release_version"`
	Stale          bool               `json:"stale"`
	FetchedAt      string             `json:"fetched_at,omitempty"`
	Rules          string             `json:"rules"`
	Skills         []catalogJSONSkill `json:"skills"`
}

type catalogJSONSkill struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Version      string   `json:"version"`
	Experimental bool     `json:"experimental"`
	Files        []string `json:"files"`
	Commands     []string `json:"commands"`
}

// newCatalogJSON returns the JSON output for the skills in list, which are the
// skills a command shows from catalog. Rules are the catalog rules, trimmed as
// skills list prints them. skills search carries them too, so both commands
// share one envelope.
func newCatalogJSON(catalog *libskills.Catalog, list []libskills.Skill) catalogJSON {
	out := catalogJSON{
		Source:         filepath.ToSlash(catalog.Label),
		ReleaseVersion: catalog.Version,
		Stale:          catalog.Stale,
		Rules:          strings.TrimSpace(catalog.Rules),
		Skills:         make([]catalogJSONSkill, 0, len(list)),
	}
	if !catalog.FetchedAt.IsZero() {
		out.FetchedAt = catalog.FetchedAt.Format(time.RFC3339)
	}
	for _, s := range list {
		out.Skills = append(out.Skills, catalogJSONSkill{
			Name:         s.Name,
			Description:  s.Description,
			Version:      s.Version,
			Experimental: s.Experimental,
			Files:        nonNilList(s.Files),
			Commands:     nonNilList(s.Commands),
		})
	}
	return out
}

// nonNilList keeps an absent list as [] in JSON rather than null.
func nonNilList(list []string) []string {
	if list == nil {
		return make([]string, 0)
	}
	return list
}
