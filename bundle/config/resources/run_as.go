package resources

import "github.com/databricks/databricks-sdk-go/service/jobs"

// RunAsSupport is how a resource type treats the bundle's run_as identity.
type RunAsSupport int

const (
	// RunAsNotApplicable means run_as semantics do not apply to the type's API,
	// so its resources deploy unchanged whatever the run_as identity is.
	RunAsNotApplicable RunAsSupport = iota

	// RunAsApplied means resources of the type run as the bundle's run_as identity.
	// The type implements [RunAsReceiver].
	RunAsApplied

	// RunAsRejected means the type's API does not support a run_as identity other
	// than the deploying user. Unless the type implements [RunAsConditionalRejecter],
	// this applies to every resource of the type.
	RunAsRejected
)

// RunAsDeclarer is implemented by every resource type to declare its [RunAsSupport].
type RunAsDeclarer interface {
	RunAsSupport() RunAsSupport
}

// RunAsReceiver is implemented by resource types that declare [RunAsApplied].
type RunAsReceiver interface {
	// ApplyRunAs sets runAs as the run_as identity of the resource,
	// unless the resource sets its own.
	ApplyRunAs(runAs *jobs.JobRunAs)
}

// RunAsValidator is implemented by a [RunAsReceiver] that cannot run as every
// kind of identity.
type RunAsValidator interface {
	// ValidateRunAs returns an error if the resource cannot run as runAs,
	// an identity other than the deploying user.
	ValidateRunAs(runAs *jobs.JobRunAs) error
}

// RunAsLegacyReceiver is implemented by a [RunAsReceiver] that
// experimental.use_legacy_run_as applies to.
type RunAsLegacyReceiver interface {
	// ApplyLegacyRunAs applies runAs to the resource in legacy mode.
	// currentUser is the user name of the deploying user.
	ApplyLegacyRunAs(runAs *jobs.JobRunAs, currentUser string)
}

// RunAsConditionalRejecter is implemented by a resource type that declares
// [RunAsRejected] when only the resources that meet a condition are rejected.
type RunAsConditionalRejecter interface {
	// RunAsRejectedSubject names the rejected resources in the error message.
	RunAsRejectedSubject() string

	// RejectsRunAs reports whether the resource meets the condition.
	RejectsRunAs() bool
}
