// Package safeerr provides errors that maintain two string representations of the error message:
// - the usual, accessible via Error()
// - "safe", accessible via SafeError()
//
// The format string is considered safe. The arguments to format strings are unsafe by default
// unless wrapped with safeerr.Safe() or implement SafeStringer() interface.
package safeerr

import (
	"errors"
	"fmt"
	"strings"
)

const maxSafeErrorSize = 1000

// safeValue marks one argument of Errorf as free of user data.
type safeValue struct{ v any }

type safeErr struct {
	// err is the fmt.Errorf result
	err error

	// safe error counterpart
	safeErr string
}

// Errorf formats an error exactly like fmt.Errorf and in addition maintains safe error message.
func Errorf(format string, args ...any) error {
	if false {
		// Tells the vet printf analyzer that this is a printf wrapper, which it
		// cannot infer on its own because the call below does not forward args
		// verbatim. fmt.Errorf rather than fmt.Sprintf, so %w is accepted here
		// too. Documented at
		// https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/printf
		_ = fmt.Errorf(format, args...)
	}

	return &safeErr{
		err:     fmt.Errorf(format, unpackArgs(args)...),
		safeErr: SafeSprintf(format, args...),
	}
}

// New returns an error whose message is entirely safe literal
func New(text string) error {
	return &safeErr{err: errors.New(text), safeErr: truncateSafe(text)}
}

// Safe marks v as free of user data, allowing its value to appear in the
// message template. Use it only for values drawn from a set the CLI itself
// defines — field paths, resource groups, API status codes — never for names,
// paths, or anything else that originates in the user's configuration.
//
// Marking a value Safe does not change the error message: the value is
// formatted by its verb exactly as it would have been without the marker.
//
// Safe does not override a value's own SafeStringer stand-in; see SafeStringer.
func Safe(v any) any {
	return safeValue{v: v}
}

// SafeStringer is implemented by values that are only partly free of user data
// and can supply a PII-free stand-in for themselves. The full value still
// appears in the error message; only the stand-in reaches the template.
//
// A resource key is the motivating case: "resources.jobs.my_job" mixes a group
// the CLI defines with a name the user chose, so it stands in for itself as
// "jobs.*" — enough to tell a failing job from a failing pipeline without
// reporting which one.
//
// An error may also be a SafeStringer: under %w its own template is preferred,
// and the stand-in is used only when it has none. SafeStringer otherwise wins
// over Safe, so wrapping such a value in Safe cannot put its user-supplied part
// back into the template.
type SafeStringer interface {
	SafeString() string
}

// SafeError returns the redacted error message: only format strings and safe values are visible.
func SafeError(err error) string {
	te, ok := errors.AsType[*safeErr](err)
	if !ok {
		return ""
	}
	return te.safeErr
}

func (e *safeErr) Error() string {
	return e.err.Error()
}

// Unwrap returns the fmt.Errorf result rather than the wrapped error itself, so
// an error built with several %w verbs keeps working: that value implements
// Unwrap() []error, which errors.Is and errors.AsType traverse.
func (e *safeErr) Unwrap() error {
	return e.err
}

// unpackArgs strips Safe markers so the message fmt produces is identical to
// the one a plain fmt.Errorf call with the same values would have produced.
func unpackArgs(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		if s, ok := a.(safeValue); ok {
			out[i] = s.v
		} else {
			out[i] = a
		}
	}
	return out
}

// SafeSprintf renders format with only its safe arguments substituted. Every
// other verb is escaped so it appears literally in the result — "%s: getting
// config" stays "%s: getting config" — which is what makes the result reportable
// without scrubbing: the format string is a source literal, and nothing else
// reaches the output.
func SafeSprintf(format string, args ...any) string {
	safeFormat, safe, ok := escapeUnsafeVerbs(format, safeArgs(args))
	if !ok {
		// The format uses a construct the scanner does not model, so which
		// argument each verb consumes is no longer known and substituting any of
		// them could pair a safe value with the wrong verb. The format is a
		// source literal, so reporting it unrendered is still safe.
		return truncateSafe(format + "(safeerrerr)")
	}

	// fmt.Errorf rather than fmt.Sprintf so %w stays valid: a nested safe error
	// is passed through as a proxy whose Error() is its own safe message.
	return truncateSafe(fmt.Errorf(safeFormat, safe...).Error())
}

// escapeUnsafeVerbs rewrites format so that only verbs with a safe argument
// still consume one, and returns those arguments in order. It reports false for
// a construct parseVerb does not model.
func escapeUnsafeVerbs(format string, safe []any) (string, []any, bool) {
	var sb strings.Builder
	var kept []any
	argIndex := 0

	for i := 0; i < len(format); {
		if format[i] != '%' {
			sb.WriteByte(format[i])
			i++
			continue
		}

		spec, verb, next, ok := parseVerb(format, i)
		if !ok {
			return "", nil, false
		}
		i = next

		// %% consumes no argument, so it passes through untouched.
		if verb == '%' {
			sb.WriteString(spec)
			continue
		}

		value, safeToRender := safeValueFor(safe, argIndex)
		argIndex++
		if !safeToRender {
			// Escaping the percent leaves the verb as literal text and consumes
			// no argument, which is what keeps the rest of the format in step.
			sb.WriteString("%" + spec)
			continue
		}

		sb.WriteString(spec)
		kept = append(kept, value)
	}

	return sb.String(), kept, true
}

// safeErrProxy carries a safe message under a verb expecting an error, so %w and
// %s render that rather than the wrapped error's full text.
type safeErrProxy struct{ msg string }

func (p safeErrProxy) Error() string { return p.msg }

// safeValueFor returns what to render for the argument at argIndex, and whether
// rendering it is safe at all.
func safeValueFor(safe []any, argIndex int) (any, bool) {
	if argIndex >= len(safe) {
		// More verbs than arguments; go vet reports the call itself.
		return nil, false
	}

	switch v := safe[argIndex].(type) {
	case safeValue:
		return v.v, true
	case error:
		// A nested safe error contributes its own safe message. Failing that, a
		// typed error can still describe itself, which is how libs/filer reports
		// a classification without the path it carries.
		if msg := SafeError(v); msg != "" {
			return safeErrProxy{msg: msg}, true
		}
		if ss, ok := v.(SafeStringer); ok {
			return safeErrProxy{msg: ss.SafeString()}, true
		}
	}

	return nil, false
}

// truncateSafe caps a safe message so a deep chain cannot produce an unbounded
// telemetry field.
func truncateSafe(s string) string {
	if len(s) > maxSafeErrorSize {
		return s[:maxSafeErrorSize]
	}
	return s
}

// safeArgs keeps the arguments expand may substitute — safe values, the
// stand-ins of SafeStringer values, and errors whose own template can be
// chained in — and drops the rest.
func safeArgs(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case safeValue:
			// A value that declares a stand-in keeps it even here: the type
			// knows which part of itself is user data, so it outranks a
			// call-site assertion that the whole value is safe.
			if ss, ok := v.v.(SafeStringer); ok {
				out[i] = safeValue{v: ss.SafeString()}
			} else {
				out[i] = v
			}
		case error:
			out[i] = v
		case SafeStringer:
			// Resolve the stand-in now and keep only that, so the value it came
			// from — which holds user data — is not retained by the error.
			out[i] = safeValue{v: v.SafeString()}
		}
	}
	return out
}

// verbFlags are the flag characters fmt accepts between '%' and the verb.
const verbFlags = "+-# 0"

// parseVerb parses the verb that starts at the '%' at index i, returning its
// full spec (e.g. "%-10q"), the verb letter, and the index just past it.
//
// It reports false for a dangling '%' and for the two constructs that change
// which argument a verb consumes: an explicit argument index (%[2]s) and a '*'
// width or precision (%*d).
func parseVerb(format string, i int) (spec string, verb byte, next int, ok bool) {
	j := i + 1
	for j < len(format) && strings.IndexByte(verbFlags, format[j]) >= 0 {
		j++
	}

	j, ok = skipNumber(format, j)
	if !ok {
		return "", 0, 0, false
	}

	if j < len(format) && format[j] == '.' {
		j, ok = skipNumber(format, j+1)
		if !ok {
			return "", 0, 0, false
		}
	}

	if j >= len(format) {
		return "", 0, 0, false
	}

	return format[i : j+1], format[j], j + 1, true
}

// skipNumber advances past a width or precision, rejecting the '*' and '['
// forms that consume an argument of their own.
func skipNumber(format string, j int) (int, bool) {
	if j < len(format) && format[j] == '*' {
		return 0, false
	}
	for j < len(format) && format[j] >= '0' && format[j] <= '9' {
		j++
	}
	// An explicit argument index may also follow a literal width or precision
	// (%2[2]s, %.2[2]s), so reject '[' here and not only ahead of the digits.
	if j < len(format) && format[j] == '[' {
		return 0, false
	}
	return j, true
}
