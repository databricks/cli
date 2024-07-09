package diag

import (
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

	// Location is a source code location associated with the diagnostic message.
	// It may be zero if there is no associated location.
	Location dyn.Location

	// Path is a path to the value in a configuration tree that the diagnostic is associated with.
	// It may be nil if there is no associated path.
	Path dyn.Path
}

// Errorf creates a new error diagnostic.
func Errorf(format string, args ...any) Diagnostics {
	return []Diagnostic{
		{
			Severity: Error,
			Summary:  fmt.Sprintf(format, args...),
		},
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

// Infof creates a new info diagnostic.
func Infof(format string, args ...any) Diagnostics {
	return []Diagnostic{
		{
			Severity: Info,
			Summary:  fmt.Sprintf(format, args...),
		},
	}
}

// Diagsnostics holds zero or more instances of [Diagnostic].
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

// Return first error in the set of diagnostics.
func (ds Diagnostics) Error() error {
	for _, d := range ds {
		if d.Severity == Error {
			return fmt.Errorf(d.Summary)
		}
	}
	return nil
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

// Returns true if the diagnostics contain the specified error
func (ds Diagnostics) ContainsError(err error) bool {
	for _, d := range ds {
		if d.Severity == Error && d.Summary == err.Error() {
			return true
		}
	}
	return false
}

// Filter returns a new list of diagnostics that do not contain the specified error
// Rename this to filter back again?
func (ds Diagnostics) RemoveError(err error) Diagnostics {
	var out Diagnostics
	for _, d := range ds {
		if d.Severity != Error || d.Summary != err.Error() {
			out = append(out, d)
		}
	}
	return out
}
