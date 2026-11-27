package skills

import (
	"context"
	"slices"

	"github.com/databricks/cli/libs/env"
)

// Origin says which kind of source served a catalog. Its values are the names of the proto's SkillsSource enum.
type Origin string

const (
	OriginLive    Origin = "LIVE"
	OriginCache   Origin = "CACHE"
	OriginBundled Origin = "BUNDLED"
	OriginFolder  Origin = "FOLDER"
)

// Record is what a skills command read from its source, for the command's telemetry event. WithRecord installs one on
// a context, and Resolve then returns a source that fills it in.
type Record struct {
	Origin Origin

	// Version is the release version of the catalog read, empty when none was.
	Version string

	// SkillName is the skill a ReadFile read, when the catalog lists it and its names are published.
	SkillName string
}

type recordKey struct{}

// WithRecord returns ctx with a new Record for one command run, and that Record.
func WithRecord(ctx context.Context) (context.Context, *Record) {
	rec := &Record{}
	return context.WithValue(ctx, recordKey{}, rec), rec
}

// record wraps src so that reads through it fill in the Record of ctx. Without a Record, src is returned as it is.
func record(ctx context.Context, src Source, origin Origin) Source {
	rec, ok := ctx.Value(recordKey{}).(*Record)
	if !ok {
		return src
	}
	rec.Origin = origin
	r := &recordingSource{Source: src, rec: rec}
	if sizer, ok := src.(FileSizer); ok {
		return &recordingSizer{recordingSource: r, sizer: sizer}
	}
	return r
}

// recordingSource passes every call to the source it wraps, and records what was read.
type recordingSource struct {
	Source
	rec *Record

	// catalog is the last catalog read through this source.
	catalog *Catalog
}

func (r *recordingSource) Catalog(ctx context.Context) (*Catalog, error) {
	catalog, err := r.Source.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	r.catalog = catalog
	r.rec.Version = catalog.Version
	return catalog, nil
}

func (r *recordingSource) ReadFile(ctx context.Context, skill, file string) ([]byte, error) {
	data, err := r.Source.ReadFile(ctx, skill, file)
	listed := r.lists(ctx, skill)
	if err == nil && listed && r.namesArePublished(ctx) {
		r.rec.SkillName = skill
	}
	return data, err
}

// namesArePublished reports whether this source's skill names are published ones. A folder's names, and those of a ref
// or base URL the user chose, are the user's own.
func (r *recordingSource) namesArePublished(ctx context.Context) bool {
	return r.rec.Origin != OriginFolder && env.Get(ctx, "DATABRICKS_SKILLS_REF") == "" && env.Get(ctx, "DATABRICKS_SKILLS_BASE_URL") == ""
}

// lists reports whether the catalog lists the skill. A read-file does not read the catalog, so it is read here.
func (r *recordingSource) lists(ctx context.Context, skill string) bool {
	if r.catalog == nil {
		if _, err := r.Catalog(ctx); err != nil {
			return false
		}
	}
	return slices.ContainsFunc(r.catalog.Skills, func(s Skill) bool { return s.Name == skill })
}

// recordingSizer is a recordingSource over a source that sizes its files, so that list-files still finds a FileSizer.
type recordingSizer struct {
	*recordingSource
	sizer FileSizer
}

func (r *recordingSizer) FileSize(ctx context.Context, skill, file string) (int64, error) {
	return r.sizer.FileSize(ctx, skill, file)
}
