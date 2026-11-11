package installer

import (
	"errors"
	"fmt"
)

// ErrNothingInstalled reports that update or uninstall found no install state
// in the scope. The command layer matches it with errors.Is to tell "nothing to
// act on" apart from a failure.
var ErrNothingInstalled = errors.New("no skills or plugins installed")

// ErrUnknownAgent reports a plugin record in the install state for an agent
// this CLI does not know (e.g. one written by a newer CLI). Update and
// uninstall leave such a record alone and report it as a failed plugin.
var ErrUnknownAgent = errors.New("unknown agent")

// unknownAgentInStateError returns ErrUnknownAgent for the agent name recorded
// in the state.
func unknownAgentInStateError(name string) error {
	return fmt.Errorf("%w %q in state", ErrUnknownAgent, name)
}

// ErrLegacyInstall reports skills from a previous install that predates state
// tracking: update and uninstall need the install state, so they ask the user
// to run install first.
var ErrLegacyInstall = errors.New("found skills from a previous install without state tracking")

// SkillError reports that a skill named via --skills could not be resolved from
// the manifest or, for uninstall, is not installed. Reason drives telemetry
// categorization (the command layer maps it with errors.AsType); Detail is the
// human-readable remainder of the message. Building the message from fields
// keeps a classification tag out of the user-facing string, so the error is
// stated exactly once.
type SkillError struct {
	Skill  string
	Reason string
	Detail string
}

// Reasons a --skills entry can fail to resolve.
const (
	// ReasonSkillNotFound: the named skill is absent from the resolved manifest
	// or, for uninstall, from the install state.
	ReasonSkillNotFound = "skill-not-found"
	// ReasonVersionIncompatible: the skill requires a newer CLI than the one running.
	ReasonVersionIncompatible = "version-incompatible"
	// ReasonExperimentalSkill: the named skill is experimental and --experimental was not set.
	ReasonExperimentalSkill = "experimental-skill"
)

func (e *SkillError) Error() string {
	// Detail is the human-readable remainder; fall back to Reason when it is empty
	// so the message stays self-describing (mirrors BlockedError.Error()).
	if e.Detail != "" {
		return fmt.Sprintf("skill %q %s", e.Skill, e.Detail)
	}
	return fmt.Sprintf("skill %q %s", e.Skill, e.Reason)
}
