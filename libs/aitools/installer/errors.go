package installer

import "fmt"

// SkillError reports that a skill named via --skills could not be resolved from
// the manifest. Reason drives telemetry categorization (the command layer maps
// it with errors.AsType); Detail is the human-readable remainder of the message.
// Building the message from fields keeps a classification tag out of the
// user-facing string, so the error is stated exactly once.
type SkillError struct {
	Skill  string
	Reason string
	Detail string
}

// Reasons a --skills entry can fail to resolve.
const (
	// ReasonSkillNotFound: the named skill is absent from the resolved manifest.
	ReasonSkillNotFound = "skill-not-found"
	// ReasonVersionIncompatible: the skill requires a newer CLI than the one running.
	ReasonVersionIncompatible = "version-incompatible"
)

func (e *SkillError) Error() string {
	return fmt.Sprintf("skill %q %s", e.Skill, e.Detail)
}
