// Package safeerr provides errors that retain their message template: the
// format string, with values that came from the user left as verbs. Telemetry
// can then report which error occurred without shipping the values
// interpolated into it.
//
// An error created here behaves exactly like the fmt.Errorf it wraps. Error()
// returns the fully interpolated message and errors.Is/errors.AsType walk the
// same chain, so converting a call site changes nothing for existing consumers.
// The template is reachable only through ErrorTemplate.
//
// The safe/unsafe vocabulary follows github.com/cockroachdb/redact: every
// value is unsafe unless the caller marks it Safe.
// See https://github.com/cockroachdb/errors#pii-free-error-reporting
package safeerr

import (
	"errors"
	"fmt"
	"strings"
)

// Errorf formats an error exactly like fmt.Errorf and retains format as the
// error's message template.
//
// format must be a string constant. It is the one part of the error reported to
// telemetry, so building it dynamically defeats the point of the package.
func Errorf(format string, args ...any) error {
	if false {
		// Tells the vet printf analyzer that this is a printf wrapper, which it
		// cannot infer on its own because the call below does not forward args
		// verbatim. fmt.Errorf rather than fmt.Sprintf, so %w is accepted here
		// too. Documented at
		// https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/printf
		_ = fmt.Errorf(format, args...)
	}

	return &templateError{
		err:      fmt.Errorf(format, renderArgs(args)...),
		template: format,
		args:     templateArgs(args),
	}
}

// New returns an error whose message is entirely literal, making the whole
// message its own template.
func New(text string) error {
	return &templateError{err: errors.New(text), template: text, args: nil}
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

// ErrorTemplate returns the message template of the outermost templated error
// in err's chain, or "" when the chain holds none.
//
// Templates chain: a %w whose wrapped error is itself templated contributes its
// own template in place of the verb, recursively. A %w wrapping any other error
// stays a bare %w, since nothing about that error is known to be safe.
func ErrorTemplate(err error) string {
	te, ok := errors.AsType[*templateError](err)
	if !ok {
		return ""
	}
	return te.expand()
}

// safeValue marks one argument of Errorf as free of user data.
type safeValue struct{ v any }

type templateError struct {
	// err is the fmt.Errorf result. It carries the interpolated message and
	// the unwrap chain, so this type never reimplements either.
	err error

	// template is the format string passed to Errorf.
	template string

	// args holds only what expand can use: safe values and wrapped errors.
	// Every other argument is nil, so no user data stays reachable here.
	args []any
}

func (e *templateError) Error() string {
	return e.err.Error()
}

// Unwrap returns the fmt.Errorf result rather than the wrapped error itself, so
// an error built with several %w verbs keeps working: that value implements
// Unwrap() []error, which errors.Is and errors.AsType traverse.
func (e *templateError) Unwrap() error {
	return e.err
}

// renderArgs strips Safe markers so the message fmt produces is identical to
// the one a plain fmt.Errorf call with the same values would have produced.
func renderArgs(args []any) []any {
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

// templateArgs keeps the arguments expand may substitute — safe values, the
// stand-ins of SafeStringer values, and errors whose own template can be
// chained in — and drops the rest.
func templateArgs(args []any) []any {
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

// expand walks the template, substituting safe values and chained templates and
// leaving every other verb in place.
func (e *templateError) expand() string {
	var sb strings.Builder
	argIndex := 0

	for i := 0; i < len(e.template); {
		if e.template[i] != '%' {
			sb.WriteByte(e.template[i])
			i++
			continue
		}

		spec, verb, next, ok := parseVerb(e.template, i)
		if !ok {
			// A construct expand does not model shifts which argument each
			// later verb consumes, so substituting past it could attribute a
			// safe value to the wrong verb. Fall back to the format string,
			// which is a source literal and safe to report on its own.
			return e.template
		}

		i = next
		if verb == '%' {
			sb.WriteString(spec)
			continue
		}

		sb.WriteString(e.substitute(spec, verb, argIndex))
		argIndex++
	}

	return sb.String()
}

// substitute returns the text to emit for a single verb.
func (e *templateError) substitute(spec string, verb byte, argIndex int) string {
	if argIndex >= len(e.args) {
		// More verbs than arguments; go vet reports the call itself.
		return spec
	}

	arg := e.args[argIndex]
	if s, ok := arg.(safeValue); ok {
		arg = s.v
	} else if verb != 'w' {
		return spec
	}

	if verb == 'w' {
		// %w is only valid in fmt.Errorf, so it is never rendered.
		err, ok := arg.(error)
		if !ok {
			return spec
		}
		// The wrapped error's own template wins. Failing that, an error that
		// supplies a stand-in contributes that instead, which is how the CLI's
		// own typed errors report their classification without the path or name
		// they carry. Only the wrapped error itself is consulted, not its chain.
		if inner := ErrorTemplate(err); inner != "" {
			return inner
		}
		if ss, ok := err.(SafeStringer); ok {
			return ss.SafeString()
		}
		return spec
	}

	return fmt.Sprintf(spec, arg)
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
	if j < len(format) && (format[j] == '*' || format[j] == '[') {
		return 0, false
	}
	for j < len(format) && format[j] >= '0' && format[j] <= '9' {
		j++
	}
	return j, true
}
