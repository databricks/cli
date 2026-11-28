package skills

import (
	"context"
	"os"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const recordFixture = "testdata/release"

// catalogOnly is a source that cannot size its files.
type catalogOnly struct{ Source }

func TestRecordFolderSendsNoName(t *testing.T) {
	ctx, rec := WithRecord(env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", recordFixture))
	src, err := Resolve(ctx)
	require.NoError(t, err)

	_, err = src.ReadFile(ctx, "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, Record{Origin: OriginFolder, Version: "1.2.0"}, *rec)
}

// Only a listed skill of a published source is named, and a ref or base URL the user chose makes the names their own.
func TestRecordSkillName(t *testing.T) {
	tests := []struct {
		name, key, skill, file, want string
	}{
		{name: "listed skill", skill: "alpha", file: "SKILL.md", want: "alpha"},
		{name: "unlisted skill", skill: "missing", file: "SKILL.md"},
		{name: "unlisted file", skill: "alpha", file: "nope.md"},
		{name: "ref set", key: "DATABRICKS_SKILLS_REF", skill: "alpha", file: "SKILL.md"},
		{name: "base URL set", key: "DATABRICKS_SKILLS_BASE_URL", skill: "alpha", file: "SKILL.md"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			if tc.key != "" {
				ctx = env.Set(ctx, tc.key, "test-value")
			}
			ctx, rec := WithRecord(ctx)
			src := record(ctx, NewFSSource(os.DirFS(recordFixture), "fixture"), OriginLive)

			_, _ = src.ReadFile(ctx, tc.skill, tc.file)
			assert.Equal(t, tc.want, rec.SkillName)
		})
	}
}

func TestRecordKeepsFileSizer(t *testing.T) {
	ctx, _ := WithRecord(t.Context())
	_, sized := record(ctx, NewFSSource(os.DirFS(recordFixture), "fixture"), OriginFolder).(FileSizer)
	_, plain := record(ctx, catalogOnly{NewFSSource(os.DirFS(recordFixture), "fixture")}, OriginFolder).(FileSizer)
	assert.True(t, sized)
	assert.False(t, plain)
}

// countingSource counts the catalog reads made through it.
type countingSource struct {
	Source
	catalogs int
}

func (c *countingSource) Catalog(ctx context.Context) (*Catalog, error) {
	c.catalogs++
	return c.Source.Catalog(ctx)
}

// A successful read reads the catalog once, for the version and the listing. A failed read reads neither.
func TestRecordReadFileCatalogCalls(t *testing.T) {
	tests := []struct {
		name, file string
		origin     Origin
		want       int
	}{
		{name: "listed skill", file: "SKILL.md", origin: OriginLive, want: 1},
		{name: "failed read", file: "nope.md", origin: OriginLive, want: 0},
		{name: "folder", file: "SKILL.md", origin: OriginFolder, want: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := WithRecord(t.Context())
			src := &countingSource{Source: NewFSSource(os.DirFS(recordFixture), "fixture")}
			_, _ = record(ctx, src, tc.origin).ReadFile(ctx, "alpha", tc.file)
			assert.Equal(t, tc.want, src.catalogs)
		})
	}
}
