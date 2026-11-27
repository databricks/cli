package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	libskills "github.com/databricks/cli/libs/skills"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func TestSkillsEventSuccessfulRead(t *testing.T) {
	rec := &libskills.Record{Origin: libskills.OriginLive, Version: "1.4.0", SkillName: "databricks-apps"}
	assert.Equal(t, protos.SkillsCommandEvent{
		Command:               protos.SkillsCommandRead,
		SkillName:             "databricks-apps",
		CatalogReleaseVersion: "1.4.0",
		Source:                protos.SkillsSourceLive,
		Outcome:               protos.SkillsCommandOutcomeSucceeded,
	}, skillsEvent(protos.SkillsCommandRead, rec, nil))
}

// A failed read never sends the name it was asked for: it may be a typo or private text.
func TestSkillsEventFailedReadSendsNoName(t *testing.T) {
	rec := &libskills.Record{Origin: libskills.OriginFolder, Version: "1.2.0", SkillName: "typo"}
	err := fmt.Errorf("%w %q", libskills.ErrUnknownSkill, "typo")
	event := skillsEvent(protos.SkillsCommandRead, rec, err)
	assert.Empty(t, event.SkillName)
	assert.Equal(t, protos.SkillsCommandOutcomeFailed, event.Outcome)
	assert.Equal(t, protos.SkillsErrorCategorySkillNotFound, event.ErrorCategory)
}

func TestReleaseVersion(t *testing.T) {
	for in, want := range map[string]string{
		"1.4.0": "1.4.0", "10.0.12": "10.0.12",
		"v1.4.0": "", "1.4": "", "1.4.0-rc1": "", "1..0": "", "1.4.x": "", "": "",
	} {
		assert.Equal(t, want, releaseVersion(in), in)
	}
}

func TestErrorCategory(t *testing.T) {
	var v map[string]any
	tests := map[string]struct {
		err  error
		want protos.SkillsErrorCategory
	}{
		"unknown skill":   {fmt.Errorf("%w", libskills.ErrUnknownSkill), protos.SkillsErrorCategorySkillNotFound},
		"unknown file":    {libskills.ErrUnknownFile, protos.SkillsErrorCategoryFileNotFound},
		"unreachable":     {fmt.Errorf("%w: %w", libskills.ErrSourceUnreachable, errors.New("dial")), protos.SkillsErrorCategorySourceUnavailable},
		"manifest syntax": {json.Unmarshal([]byte("{"), &v), protos.SkillsErrorCategoryCatalogInvalid},
		"permission":      {fmt.Errorf("read: %w", fs.ErrPermission), protos.SkillsErrorCategoryPermissionDenied},
		"filesystem":      {&fs.PathError{Op: "open", Path: "x", Err: errors.New("i/o")}, protos.SkillsErrorCategoryFilesystem},
		"anything else":   {errors.New("something else"), protos.SkillsErrorCategoryUncategorized},
	}
	for name, tc := range tests {
		assert.Equal(t, tc.want, errorCategory(tc.err), name)
	}
}
