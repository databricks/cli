package skills

import (
	"encoding/json"
	"errors"
	"io/fs"
	"strings"

	"github.com/databricks/cli/cmd/root"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// recordRuns logs one SkillsCommandEvent per run of each command in the group, and resolves the upload's auth in the
// background, as aitools does. It is best effort: it never fails, prompts or changes output or exit codes.
func recordRuns(group *cobra.Command) {
	for _, c := range group.Commands() {
		command := commandType(c.Name())
		run := c.RunE
		c.RunE = func(cmd *cobra.Command, args []string) error {
			if telemetry.Disabled(cmd.Context()) {
				return run(cmd, args)
			}
			ctx, rec := libskills.WithRecord(cmd.Context())
			cmd.SetContext(ctx)
			err := run(cmd, args)
			logEvent(cmd, command, rec, err)
			return err
		}
		// A run that logged its event waits up to the auth budget for the upload's auth, as install does.
		root.WithTelemetryAuth(c).WaitAtExit = true
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
	event := skillsEvent(command, rec, err)
	telemetry.Log(ctx, protos.DatabricksCliLog{SkillsCommandEvent: &event})
	if agent := root.CallingAgentType(); agent != protos.AitoolsAgentTypeUnspecified {
		telemetry.SetCallingAgent(ctx, agent)
	}
}

// skillsEvent builds one run's event. A failed run sends no skill name: it may have asked for one the catalog lacks.
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

// errorCategory classifies a failed run by its error kind. The error's text is never sent: it can carry a typed name.
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
	case errors.Is(err, libskills.ErrUnsafeManifestPath), errors.Is(err, libskills.ErrManifestTooLarge), syntax, typed:
		return protos.SkillsErrorCategoryCatalogInvalid
	case errors.Is(err, fs.ErrPermission):
		return protos.SkillsErrorCategoryPermissionDenied
	case pathErr:
		return protos.SkillsErrorCategoryFilesystem
	}
	return protos.SkillsErrorCategoryUncategorized
}
