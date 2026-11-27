package skills

import (
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
