package diag

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

type Diagnostic struct {
	Severity Severity

	// Summary is a short description of the diagnostic.
	// This is expected to be a single line and always present.
	Summary string

	// Detail is a longer description of the diagnostic.
	// This may be multiple lines and may be nil.
	Detail string

	// Locations are the source code locations associated with the diagnostic message.
	// It may be empty if there are no associated locations.
	Locations []dyn.Location

	// Paths are paths to the values in the configuration tree that the diagnostic is associated with.
	// It may be nil if there are no associated paths.
	Paths []dyn.Path

	// A diagnostic ID. Only used for select diagnostic messages.
	ID ID
}

// Error implements the error interface so an error-severity Diagnostic can be
// returned and propagated as a regular Go error. The message mirrors the
// formatting used by [Diagnostics.Error].
func (d Diagnostic) Error() string {
	message := d.Detail
	if message == "" {
		message = d.Summary
	}
	if d.ID != "" {
		message = string(d.ID) + ": " + message
	}
	return message
}

// Errorf creates a new error diagnostic.
//
// The returned value implements the error interface so it can be returned and
// propagated like any other Go error while still carrying the diagnostic's
// Summary/Detail/ID for rendering at the top level.
func Errorf(format string, args ...any) error {
	return Diagnostic{
		Severity: Error,
		Summary:  fmt.Sprintf(format, args...),
	}
}

// FromErr returns a new error diagnostic from the specified error, if any.
func FromErr(err error) Diagnostics {
	if err == nil {
		return nil
	}
	return []Diagnostic{
		{
			Severity: Error,
			Summary:  FormatAPIErrorSummary(err),
			Detail:   FormatAPIErrorDetails(err),
		},
	}
}

// DiagnosticFromError converts an error into a single error-severity Diagnostic
// for rendering. If the error (or anything it wraps) is already a Diagnostic, it
// is returned unchanged so its Locations/Paths/Detail/ID render as authored.
// Otherwise the error is formatted as an API/error diagnostic.
func DiagnosticFromError(err error) Diagnostic {
	if d, ok := errors.AsType[Diagnostic](err); ok {
		return d
	}
	return Diagnostic{
		Severity: Error,
		Summary:  FormatAPIErrorSummary(err),
		Detail:   FormatAPIErrorDetails(err),
	}
}

// FromErr returns a new warning diagnostic from the specified error, if any.
func WarningFromErr(err error) Diagnostics {
	if err == nil {
		return nil
	}
	return []Diagnostic{
		{
			Severity: Warning,
			Summary:  err.Error(),
		},
	}
}

// Warningf creates a new warning diagnostic.
func Warningf(format string, args ...any) Diagnostics {
	return []Diagnostic{
		{
			Severity: Warning,
			Summary:  fmt.Sprintf(format, args...),
		},
	}
}

// Recommendationf creates a new recommendation diagnostic.
func Recommendationf(format string, args ...any) Diagnostics {
	return []Diagnostic{
		{
			Severity: Recommendation,
			Summary:  fmt.Sprintf(format, args...),
		},
	}
}

// Diagnostics holds zero or more instances of [Diagnostic].
type Diagnostics []Diagnostic

// Append adds a new diagnostic to the end of the list.
func (ds Diagnostics) Append(d Diagnostic) Diagnostics {
	return append(ds, d)
}

// Extend adds all diagnostics from another list to the end of the list.
func (ds Diagnostics) Extend(other Diagnostics) Diagnostics {
	return append(ds, other...)
}

// HasError returns true if any of the diagnostics are errors.
func (ds Diagnostics) HasError() bool {
	for _, d := range ds {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// Error returns the error-severity diagnostics in the set as a single error, or
// nil if there are none. A single error is returned as the [Diagnostic] itself
// (it implements the error interface); multiple errors are combined with
// [errors.Join] so they unpack via Unwrap() []error and render as separate
// diagnostic blocks (see [FlushError]). Either way Locations/Paths/Detail/ID are
// preserved and render correctly when surfaced via [DiagnosticFromError].
func (ds Diagnostics) Error() error {
	var errs []error
	for _, d := range ds {
		if d.Severity == Error {
			errs = append(errs, d)
		}
	}
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errors.Join(errs...)
	}
}

// Filter returns a new list of diagnostics that match the specified severity.
func (ds Diagnostics) Filter(severity Severity) Diagnostics {
	var out Diagnostics
	for _, d := range ds {
		if d.Severity == severity {
			out = append(out, d)
		}
	}
	return out
}
