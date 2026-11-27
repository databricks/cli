package skills

import (
	"encoding/json"
	"errors"
	"io/fs"
	"strings"

	"github.com/databricks/cli/cmd/aitools"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// recordRuns makes each command of the group log one SkillsCommandEvent per run, and resolves the auth its upload
// needs in the background, as the aitools commands do. It is best effort: it never fails, prompts or changes the
// command's output or exit code, and it logs nothing when telemetry is disabled.
func recordRuns(group *cobra.Command) {
	for _, c := range group.Commands() {
		command := commandType(c.Name())
		run := c.RunE
		c.RunE = func(cmd *cobra.Command, args []string) error {
			ctx, rec := libskills.WithRecord(cmd.Context())
			cmd.SetContext(ctx)
			err := run(cmd, args)
			logEvent(cmd, command, rec, err)
			return err
		}
		// A run that logged its event waits up to the auth budget for the upload's auth, as install does.
		aitools.WithTelemetryAuth(c).WaitAtExit = true
	}
}

func commandType(name string) protos.SkillsCommand {
	switch name {
	case "list":
		return protos.SkillsCommandList
	case "read":
		return protos.SkillsCommandRead
	case "list-files":
		return protos.SkillsCommandListFiles
	case "read-file":
		return protos.SkillsCommandReadFile
	case "search":
		return protos.SkillsCommandSearch
	}
	return protos.SkillsCommandUnspecified
}

// logEvent buffers the event of a run for the upload at exit, and sets the calling agent for its execution context.
func logEvent(cmd *cobra.Command, command protos.SkillsCommand, rec *libskills.Record, err error) {
	ctx := cmd.Context()
	if telemetry.Disabled(ctx) {
		return
	}
	event := skillsEvent(command, rec, err)
	telemetry.Log(ctx, protos.DatabricksCliLog{SkillsCommandEvent: &event})
	if agent := aitools.CallingAgentType(); agent != protos.AitoolsAgentTypeUnspecified {
		telemetry.SetCallingAgent(ctx, agent)
	}
}

// skillsEvent builds the event of one run. The skill name is sent only for a successful run: a failed read may have
// been asked for a name the catalog does not list.
func skillsEvent(command protos.SkillsCommand, rec *libskills.Record, err error) protos.SkillsCommandEvent {
	event := protos.SkillsCommandEvent{
		Command:               command,
		CatalogReleaseVersion: releaseVersion(rec.Version),
		Source:                protos.SkillsSource(rec.Origin),
		Outcome:               protos.SkillsCommandOutcomeSucceeded,
	}
	if err != nil {
		event.Outcome = protos.SkillsCommandOutcomeFailed
		event.ErrorCategory = errorCategory(err)
		return event
	}
	event.SkillName = rec.SkillName
	return event
}

// releaseVersion returns v when it is <major>.<minor>.<patch> with no leading "v", as the proto requires, else "".
func releaseVersion(v string) string {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return ""
	}
	for _, p := range parts {
		if p == "" || strings.Trim(p, "0123456789") != "" {
			return ""
		}
	}
	return v
}

// errorCategory classifies a failed run by the kind of its error. The error's text is never sent, since it can carry a
// name the user typed.
func errorCategory(err error) protos.SkillsErrorCategory {
	_, syntax := errors.AsType[*json.SyntaxError](err)
	_, typed := errors.AsType[*json.UnmarshalTypeError](err)
	_, pathErr := errors.AsType[*fs.PathError](err)
	switch {
	case errors.Is(err, libskills.ErrUnknownSkill):
		return protos.SkillsErrorCategorySkillNotFound
	case errors.Is(err, libskills.ErrUnknownFile):
		return protos.SkillsErrorCategoryFileNotFound
	case errors.Is(err, libskills.ErrNoSource), errors.Is(err, libskills.ErrSourceUnreachable):
		return protos.SkillsErrorCategorySourceUnavailable
	case errors.Is(err, libskills.ErrUnsafeManifestPath), syntax, typed:
		return protos.SkillsErrorCategoryCatalogInvalid
	case errors.Is(err, fs.ErrPermission):
		return protos.SkillsErrorCategoryPermissionDenied
	case pathErr:
		return protos.SkillsErrorCategoryFilesystem
	}
	return protos.SkillsErrorCategoryUncategorized
}
