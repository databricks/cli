package diag

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/safeerr"
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

	// ErrorTemplate is a PII-free description of the error this diagnostic came
	// from, suitable for telemetry: the message template of a safeerr error plus
	// the safe fields of any API error at the end of its chain. Empty when the
	// diagnostic was not built from an error, or from an error carrying neither.
	//
	// Unlike Summary, this is never user-authored, so it can be aggregated
	// across the fleet without scrubbing. See libs/safeerr.
	ErrorTemplate string
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
			Severity:      Error,
			Summary:       FormatAPIErrorSummary(err),
			Detail:        FormatAPIErrorDetails(err),
			ErrorTemplate: ErrorTemplate(err),
		},
	}
}

// ErrorTemplate builds the PII-free description recorded in
// Diagnostic.ErrorTemplate, for callers holding an error rather than a
// diagnostic. The two halves are independent: a CLI error carries a template but
// may wrap no API error, and an API error reached without any safeerr wrapping
// has safe fields but no template.
func ErrorTemplate(err error) string {
	template := safeerr.ErrorTemplate(err)
	if template == "" {
		// Not raised through safeerr, but a typed error can still describe itself
		// — libs/filer's errors do. This covers the call sites not yet converted,
		// which is most of them.
		if ss, ok := err.(safeerr.SafeStringer); ok {
			template = ss.SafeString()
		}
	}
	apiDescription := SafeAPIErrorDescription(err)

	switch {
	case apiDescription == "":
		return template
	case template == "":
		return apiDescription
	default:
		return template + " [" + apiDescription + "]"
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

// Return first error in the set of diagnostics.
func (ds Diagnostics) Error() error {
	for _, d := range ds {
		if d.Severity == Error {
			message := d.Detail
			if message == "" {
				message = d.Summary
			}
			if d.ID != "" {
				message = string(d.ID) + ": " + message
			}
			return errors.New(message)
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
