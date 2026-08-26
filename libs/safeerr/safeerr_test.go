package safeerr

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unsafeValue appears in every argument that is not marked Safe, so a test can
// assert it never reaches a template.
const unsafeValue = "resources.jobs.my_secret_job"

func TestErrorf(t *testing.T) {
	tests := []struct {
		name            string
		format          string
		args            []any
		wantMessage     string
		wantSafeMessage string
	}{
		{
			name:            "no verbs",
			format:          "state conversion failed",
			wantMessage:     "state conversion failed",
			wantSafeMessage: "state conversion failed",
		},
		{
			name:            "unsafe string",
			format:          "%s: getting config",
			args:            []any{unsafeValue},
			wantMessage:     unsafeValue + ": getting config",
			wantSafeMessage: "%s: getting config",
		},
		{
			name:            "unsafe int",
			format:          "unsupported deployment state version: %d",
			args:            []any{7},
			wantMessage:     "unsupported deployment state version: 7",
			wantSafeMessage: "unsupported deployment state version: %d",
		},
		{
			name:            "safe value is substituted",
			format:          "%s: cannot set resolved value for field %q",
			args:            []any{unsafeValue, Safe("tasks[0].job_id")},
			wantMessage:     unsafeValue + `: cannot set resolved value for field "tasks[0].job_id"`,
			wantSafeMessage: `%s: cannot set resolved value for field "tasks[0].job_id"`,
		},
		{
			name:            "only safe values",
			format:          "cannot convert %s to %s",
			args:            []any{Safe("string"), Safe("int64")},
			wantMessage:     "cannot convert string to int64",
			wantSafeMessage: "cannot convert string to int64",
		},
		{
			// SafeError is a message rather than a format string, so %% renders as
			// a literal percent just as it does in the real message.
			name:            "escaped percent renders as a percent",
			format:          "100%% of %s",
			args:            []any{unsafeValue},
			wantMessage:     "100% of " + unsafeValue,
			wantSafeMessage: "100% of %s",
		},
		{
			name:            "flags and width on an unsafe verb",
			format:          "%-12s|",
			args:            []any{"job"},
			wantMessage:     "job         |",
			wantSafeMessage: "%-12s|",
		},
		{
			name:            "flags and width on a safe verb",
			format:          "%-12s|",
			args:            []any{Safe("job")},
			wantMessage:     "job         |",
			wantSafeMessage: "job         |",
		},
		{
			name:            "width and precision on a safe verb",
			format:          "%08.3f",
			args:            []any{Safe(3.5)},
			wantMessage:     "0003.500",
			wantSafeMessage: "0003.500",
		},
		{
			name:            "plus v on an unsafe verb",
			format:          "reading state: %+v",
			args:            []any{struct{ Path string }{unsafeValue}},
			wantMessage:     "reading state: {Path:" + unsafeValue + "}",
			wantSafeMessage: "reading state: %+v",
		},
		{
			name:            "safe bool",
			format:          "recovery enabled: %v",
			args:            []any{Safe(true)},
			wantMessage:     "recovery enabled: true",
			wantSafeMessage: "recovery enabled: true",
		},
		{
			name:            "mixed safe and unsafe in order",
			format:          "%s: %s for %s in %s",
			args:            []any{Safe("jobs"), unsafeValue, Safe("PrepareState"), "/home/user/bundle"},
			wantMessage:     "jobs: " + unsafeValue + " for PrepareState in /home/user/bundle",
			wantSafeMessage: "jobs: %s for PrepareState in %s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Errorf(tt.format, tt.args...)
			assert.Equal(t, tt.wantMessage, err.Error())
			assert.Equal(t, tt.wantSafeMessage, SafeError(err))
		})
	}
}

// TestErrorfMessageMatchesFmt is the property that makes converting a call site
// a no-op for every existing consumer, including acceptance test goldens.
func TestErrorfMessageMatchesFmt(t *testing.T) {
	tests := []struct {
		name   string
		format string
		safe   []any
		raw    []any
	}{
		{
			name:   "quoted safe value",
			format: "field %q",
			safe:   []any{Safe("tasks[0].job_id")},
			raw:    []any{"tasks[0].job_id"},
		},
		{
			name:   "safe struct with plus v",
			format: "%+v",
			safe:   []any{Safe(struct{ A int }{1})},
			raw:    []any{struct{ A int }{1}},
		},
		{
			name:   "safe nil",
			format: "%v",
			safe:   []any{Safe(nil)},
			raw:    []any{nil},
		},
		{
			name:   "safe error with s verb",
			format: "%s",
			safe:   []any{Safe(fs.ErrNotExist)},
			raw:    []any{fs.ErrNotExist},
		},
		{
			name:   "too few arguments",
			format: "%s and %s",
			safe:   []any{Safe("one")},
			raw:    []any{"one"},
		},
		{
			name:   "too many arguments",
			format: "%s",
			safe:   []any{Safe("one"), Safe("two")},
			raw:    []any{"one", "two"},
		},
		{
			name:   "wrong verb for type",
			format: "%d",
			safe:   []any{Safe("not a number")},
			raw:    []any{"not a number"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, fmt.Errorf(tt.format, tt.raw...).Error(), Errorf(tt.format, tt.safe...).Error())
		})
	}
}

func TestSafeErrorChains(t *testing.T) {
	inner := Errorf("cannot convert %s to %s", Safe("string"), Safe("int64"))
	middle := Errorf("%s: cannot set resolved value for field %q: %w", unsafeValue, Safe("tasks[0].job_id"), inner)
	outer := Errorf("%s: SaveState: %w", unsafeValue, middle)

	assert.Equal(t,
		unsafeValue+": SaveState: "+unsafeValue+`: cannot set resolved value for field "tasks[0].job_id": cannot convert string to int64`,
		outer.Error())
	assert.Equal(t,
		`%s: SaveState: %s: cannot set resolved value for field "tasks[0].job_id": cannot convert string to int64`,
		SafeError(outer))

	// Each level still reports its own template.
	assert.Equal(t, `%s: cannot set resolved value for field "tasks[0].job_id": cannot convert string to int64`, SafeError(middle))
	assert.Equal(t, "cannot convert string to int64", SafeError(inner))
}

func TestSafeErrorChainsThroughForeignError(t *testing.T) {
	// A %w wrapping an error with no template keeps the bare verb.
	err := Errorf("reading %s: %w", "/home/user/state.json", fs.ErrNotExist)

	assert.Equal(t, "reading /home/user/state.json: "+fs.ErrNotExist.Error(), err.Error())
	assert.Equal(t, "reading %s: %w", SafeError(err))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestSafeErrorChainsThroughPlainWrap(t *testing.T) {
	// A plain fmt.Errorf in the middle of the chain contributes nothing, so the
	// innermost template that is known reaches the top.
	inner := Errorf("cannot look up %q", Safe("continuous.pause_status"))
	err := fmt.Errorf("%s: %w", unsafeValue, inner)

	assert.Equal(t, `cannot look up "continuous.pause_status"`, SafeError(err))
}

func TestSafeErrorChainsSeveralWrappedErrors(t *testing.T) {
	first := Errorf("group %s has no adapter", Safe("quality_monitors"))
	err := Errorf("%s: %w and %w", unsafeValue, first, fs.ErrPermission)

	assert.Equal(t, "%s: group quality_monitors has no adapter and %w", SafeError(err))
	assert.ErrorIs(t, err, fs.ErrPermission)
	assert.ErrorIs(t, err, first)
}

func TestSafeErrorWithoutSafeErr(t *testing.T) {
	assert.Empty(t, SafeError(nil))
	assert.Empty(t, SafeError(errors.New(unsafeValue)))
	assert.Empty(t, SafeError(fmt.Errorf("reading %s", unsafeValue)))
	assert.Empty(t, SafeError(fs.ErrNotExist))
}

func TestNew(t *testing.T) {
	err := New("state conversion failed")
	assert.Equal(t, "state conversion failed", err.Error())
	assert.Equal(t, "state conversion failed", SafeError(err))
}

func TestNewDoesNotFormat(t *testing.T) {
	// New takes a literal message, so verbs in it are neither expanded nor lost.
	err := New("100% of %s attempts failed")
	assert.Equal(t, "100% of %s attempts failed", err.Error())
	assert.Equal(t, "100% of %s attempts failed", SafeError(err))
}

func TestSafeErrorFallsBackToRawFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []any
	}{
		{
			name:   "explicit argument index",
			format: "%[1]s and %[1]s",
			args:   []any{Safe("jobs")},
		},
		{
			name:   "star width",
			format: "%*d",
			args:   []any{Safe(5), Safe(42)},
		},
		{
			name:   "star precision",
			format: "%.*f",
			args:   []any{Safe(2), Safe(3.5)},
		},
		{
			name:   "dangling percent",
			format: "done: 100%",
			args:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The format string is a source literal, so reporting it verbatim is
			// safe even though its verbs were not analysed. The marker says the
			// scan bailed rather than that the format happened to render this way.
			assert.Equal(t, tt.format+"(safeerrerr)", SafeError(Errorf(tt.format, tt.args...)))
		})
	}
}

func TestErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := Errorf("%s: %w", unsafeValue, sentinel)

	assert.ErrorIs(t, err, sentinel)
	assert.ErrorIs(t, Errorf("outer: %w", err), sentinel)
	assert.NotErrorIs(t, Errorf("no wrapping: %s", sentinel), sentinel)
}

func TestErrorsAsType(t *testing.T) {
	err := Errorf("%s: %w", unsafeValue, &fs.PathError{Op: "open", Path: "/tmp/x", Err: fs.ErrNotExist})

	pathErr, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pathErr.Op)
}

// TestSafeErrorNeverLeaksUnsafeValues is the security property of the package:
// nothing that was not marked Safe reaches the template.
func TestSafeErrorNeverLeaksUnsafeValues(t *testing.T) {
	const secret = "SECRET-a1b2c3"

	errs := []error{
		Errorf("%s", secret),
		Errorf("%q: %v", secret, errors.New(secret)),
		Errorf("%s: %w", secret, errors.New(secret)),
		Errorf("%s: %w", secret, Errorf("inner %s: %w", secret, fs.ErrNotExist)),
		Errorf("%v", struct{ Name string }{secret}),
		Errorf("%s and %s", Safe("jobs"), secret),
		Errorf("%[1]s", secret),
		Errorf("%*s", Safe(4), secret),
	}

	for _, err := range errs {
		require.Contains(t, err.Error(), secret, "message should carry the value")
		assert.NotContains(t, SafeError(err), secret, "safe message must not carry the value")
	}
}

func TestParseVerb(t *testing.T) {
	tests := []struct {
		format   string
		wantSpec string
		wantVerb byte
		wantNext int
		wantOk   bool
	}{
		{
			format:   "%s",
			wantSpec: "%s",
			wantVerb: 's',
			wantNext: 2,
			wantOk:   true,
		},
		{
			format:   "%%",
			wantSpec: "%%",
			wantVerb: '%',
			wantNext: 2,
			wantOk:   true,
		},
		{
			format:   "%+v",
			wantSpec: "%+v",
			wantVerb: 'v',
			wantNext: 3,
			wantOk:   true,
		},
		{
			format:   "%#v",
			wantSpec: "%#v",
			wantVerb: 'v',
			wantNext: 3,
			wantOk:   true,
		},
		{
			format:   "%-12q",
			wantSpec: "%-12q",
			wantVerb: 'q',
			wantNext: 5,
			wantOk:   true,
		},
		{
			format:   "%08.3f",
			wantSpec: "%08.3f",
			wantVerb: 'f',
			wantNext: 6,
			wantOk:   true,
		},
		{
			format:   "% d",
			wantSpec: "% d",
			wantVerb: 'd',
			wantNext: 3,
			wantOk:   true,
		},
		{
			format:   "%w",
			wantSpec: "%w",
			wantVerb: 'w',
			wantNext: 2,
			wantOk:   true,
		},
		{
			format: "%[1]s",
			wantOk: false,
		},
		{
			format: "%2[2]s",
			wantOk: false,
		},
		{
			format: "%.2[2]s",
			wantOk: false,
		},
		{
			format: "%*d",
			wantOk: false,
		},
		{
			format: "%.*f",
			wantOk: false,
		},
		{
			format: "%",
			wantOk: false,
		},
		{
			format: "%-",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			spec, verb, next, ok := parseVerb(tt.format, 0)
			assert.Equal(t, tt.wantOk, ok)
			if !tt.wantOk {
				return
			}
			assert.Equal(t, tt.wantSpec, spec)
			assert.Equal(t, string(tt.wantVerb), string(verb))
			assert.Equal(t, tt.wantNext, next)
		})
	}
}

// TestSafeErrorIsStableAcrossCalls guards against expand mutating state.
func TestSafeErrorIsStableAcrossCalls(t *testing.T) {
	err := Errorf("%s: field %q: %w", unsafeValue, Safe("id"), Errorf("inner %d", Safe(2)))
	first := SafeError(err)
	assert.Equal(t, first, SafeError(err))
	assert.Equal(t, first, SafeError(err))
}

// TestUnsafeArgsAreNotRetained checks that unsafe values are dropped at
// construction rather than kept alive inside the error for later inspection.
func TestUnsafeArgsAreNotRetained(t *testing.T) {
	// The safe message is rendered at construction, so the error holds a string
	// and never the arguments themselves. Nothing unsafe is reachable from it.
	err := Errorf("%s: %d", unsafeValue, 42)

	te, ok := errors.AsType[*safeErr](err)
	require.True(t, ok)
	assert.Equal(t, "%s: %d", te.safeErr)
	assert.NotContains(t, te.safeErr, unsafeValue)
}

func TestSafeArgsAreRendered(t *testing.T) {
	err := Errorf("%s: %s", Safe("jobs"), unsafeValue)

	assert.Equal(t, "jobs: "+unsafeValue, err.Error())
	assert.Equal(t, "jobs: %s", SafeError(err))
}

func TestSafeErrorDeepChain(t *testing.T) {
	err := New("root cause")
	for range 5 {
		err = Errorf("%s: %w", unsafeValue, err)
	}

	assert.Equal(t, strings.Repeat("%s: ", 5)+"root cause", SafeError(err))
}

// safeStringerKey stands in for a resource key: the full value carries a
// user-chosen name, the stand-in keeps only the group.
type safeStringerKey string

func (k safeStringerKey) SafeString() string { return "jobs.*" }

// standInOnlyErr is a typed error carrying user data in its message and a
// PII-free classification as its stand-in, modelled on libs/filer's errors.
type standInOnlyErr struct{}

func (standInOnlyErr) Error() string      { return "access denied: /Workspace/Users/a@b.com/x" }
func (standInOnlyErr) SafeString() string { return "access denied" }

func TestSafeStringer(t *testing.T) {
	key := safeStringerKey("resources.jobs.my_job")

	tests := []struct {
		name            string
		format          string
		args            []any
		wantMessage     string
		wantSafeMessage string
	}{
		{
			name:            "s verb",
			format:          "cannot update %s: %w",
			args:            []any{key, fs.ErrPermission},
			wantMessage:     "cannot update resources.jobs.my_job: " + fs.ErrPermission.Error(),
			wantSafeMessage: "cannot update jobs.*: %w",
		},
		{
			name:            "q verb quotes the stand-in like the value",
			format:          "%q not found",
			args:            []any{key},
			wantMessage:     `"resources.jobs.my_job" not found`,
			wantSafeMessage: `"jobs.*" not found`,
		},
		{
			name:            "alongside Safe and unsafe args",
			format:          "cannot %s %s: field %q: %s",
			args:            []any{Safe("update"), key, Safe("tasks[0].job_id"), unsafeValue},
			wantMessage:     "cannot update resources.jobs.my_job: field \"tasks[0].job_id\": " + unsafeValue,
			wantSafeMessage: `cannot update jobs.*: field "tasks[0].job_id": %s`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Errorf(tt.format, tt.args...)
			assert.Equal(t, tt.wantMessage, err.Error())
			assert.Equal(t, tt.wantSafeMessage, SafeError(err))
		})
	}
}

func TestSafeStringerValueIsNotRetained(t *testing.T) {
	err := Errorf("%s", safeStringerKey("resources.jobs.my_job"))

	te, ok := errors.AsType[*safeErr](err)
	require.True(t, ok)
	// Only the stand-in reaches the safe message; the key itself is gone.
	assert.Equal(t, "jobs.*", te.safeErr)
	assert.NotContains(t, te.safeErr, "my_job")
}

func TestSafeStringerErrorPrefersItsSafeMessage(t *testing.T) {
	// A templated error under %w contributes its template, not a stand-in.
	inner := Errorf("inner %d", Safe(1))
	err := Errorf("outer: %w", inner)
	assert.Equal(t, "outer: inner 1", SafeError(err))
}

func TestSafeStringerErrorFallsBackToStandIn(t *testing.T) {
	// An error with no template of its own contributes its stand-in, which is how
	// a typed error reports its classification without the path it carries.
	err := Errorf("writing state: %w", standInOnlyErr{})
	assert.Equal(t, "writing state: access denied: /Workspace/Users/a@b.com/x", err.Error())
	assert.Equal(t, "writing state: access denied", SafeError(err))
	assert.NotContains(t, SafeError(err), "/Workspace")

	// Without a stand-in the verb stays, as before.
	assert.Equal(t, "writing state: %w", SafeError(Errorf("writing state: %w", fs.ErrPermission)))
}

func TestSafeStringerOutranksSafe(t *testing.T) {
	// Wrapping a SafeStringer in Safe must not put its user-supplied part back
	// into the template.
	err := Errorf("%s", Safe(safeStringerKey("resources.jobs.my_job")))
	assert.Equal(t, "resources.jobs.my_job", err.Error())
	assert.Equal(t, "jobs.*", SafeError(err))
}

func TestSafeErrorWrapNilError(t *testing.T) {
	// %w whose argument holds no error: a nil error interface matches no case in
	// templateArgs, so nothing is retained and the verb stays in the template
	// because there is no error to chain one from.
	var wrapped error

	err := Errorf("wrapping: %w", wrapped)
	assert.Equal(t, fmt.Errorf("wrapping: %w", wrapped).Error(), err.Error())
	assert.Equal(t, "wrapping: %w", SafeError(err))
}

// cyclicErr unwraps to whatever it is pointed at, which a test uses to close a
// loop back to an ancestor.
type cyclicErr struct{ inner error }

func (*cyclicErr) Error() string      { return "cyclic" }
func (c *cyclicErr) Unwrap() error    { return c.inner }
func (*cyclicErr) SafeString() string { return "cyclic" }

func TestSafeErrorCyclicChainTerminates(t *testing.T) {
	// An error whose Unwrap reaches back to the templated error that wraps it.
	// Following it would recurse forever, so the traversal is bounded.
	c := &cyclicErr{}
	err := Errorf("outer: %w", c)
	c.inner = err

	// The assertion is that this returns at all rather than exhausting the stack.
	assert.NotEmpty(t, SafeError(err))
}

func TestSafeErrorDeepChainTerminates(t *testing.T) {
	// Each layer's safe message is rendered at construction from the layer below,
	// so a deep chain costs nothing at read time and cannot recurse.
	err := New("root")
	for range 100 {
		err = Errorf("%w", err)
	}
	assert.NotPanics(t, func() { SafeError(err) })
	assert.Equal(t, "root", SafeError(err))
}

func TestSafeErrorStandInUnderOrdinaryVerb(t *testing.T) {
	// %s rather than %w: the error still contributes its stand-in, so a typed
	// error reports its classification either way.
	err := Errorf("writing state: %s", standInOnlyErr{})

	assert.Equal(t, "writing state: access denied: /Workspace/Users/a@b.com/x", err.Error())
	assert.Equal(t, "writing state: access denied", SafeError(err))
	assert.NotContains(t, SafeError(err), "/Workspace")
}

func TestSafeErrorIsCapped(t *testing.T) {
	// A safe message is a telemetry field, so it cannot grow without bound even
	// though every part of it comes from source literals.
	err := New(strings.Repeat("x", maxSafeErrorSize*2))
	assert.Len(t, SafeError(err), maxSafeErrorSize)

	err = Errorf("%s", Safe(strings.Repeat("y", maxSafeErrorSize*2)))
	assert.Len(t, SafeError(err), maxSafeErrorSize)
}
