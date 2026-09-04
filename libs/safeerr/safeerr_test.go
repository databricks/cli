package safeerr

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unsafeValue appears in every argument that is not marked Safe, so a test can
// assert it never reaches a template.
const unsafeValue = "resources.jobs.my_secret_job"

// nilError is a nil error interface, for the row that wraps nothing.
var nilError error

// unsafeMarkers are the user-data-shaped strings the fixtures use. A safe message
// must never contain one, whichever row produced it.
var unsafeMarkers = []string{unsafeValue, "/Workspace", "a@b.com", "my_job"}

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
		{
			name:            "wrapping a safe error contributes its safe message",
			format:          "%s: SaveState: %w",
			args:            []any{unsafeValue, Errorf("cannot convert %s to %s", Safe("string"), Safe("int64"))},
			wantMessage:     unsafeValue + ": SaveState: cannot convert string to int64",
			wantSafeMessage: "%s: SaveState: cannot convert string to int64",
		},
		{
			name:            "wrapping a foreign error keeps the verb",
			format:          "reading %s: %w",
			args:            []any{"/home/user/state.json", fs.ErrNotExist},
			wantMessage:     "reading /home/user/state.json: " + fs.ErrNotExist.Error(),
			wantSafeMessage: "reading %s: %w",
		},
		{
			name:            "several wrapped errors",
			format:          "%s: %w and %w",
			args:            []any{unsafeValue, Errorf("group %s has no adapter", Safe("quality_monitors")), fs.ErrPermission},
			wantMessage:     unsafeValue + ": group quality_monitors has no adapter and " + fs.ErrPermission.Error(),
			wantSafeMessage: "%s: group quality_monitors has no adapter and %w",
		},

		// Malformed calls. go vet reports these at a real call site; the point here
		// is that the message still matches fmt's and the safe half stays sane.
		{
			name:            "too few arguments",
			format:          "%s and %s",
			args:            []any{Safe("one")},
			wantMessage:     "one and %!s(MISSING)",
			wantSafeMessage: "one and %s",
		},
		{
			// The extra value is dropped rather than reported, since only verbs
			// with an argument survive into the safe format.
			name:            "too many arguments",
			format:          "%s",
			args:            []any{Safe("one"), Safe("two")},
			wantMessage:     "one%!(EXTRA string=two)",
			wantSafeMessage: "one",
		},
		{
			// A safe value renders through the wrong verb, marker and all. An
			// unsafe one would have had its verb escaped instead.
			name:            "wrong verb for type",
			format:          "%d",
			args:            []any{Safe("not a number")},
			wantMessage:     "%!d(string=not a number)",
			wantSafeMessage: "%!d(string=not a number)",
		},
		{
			name:            "safe nil",
			format:          "%v",
			args:            []any{Safe(nil)},
			wantMessage:     "<nil>",
			wantSafeMessage: "<nil>",
		},
		{
			name:            "safe struct with plus v",
			format:          "%+v",
			args:            []any{Safe(struct{ A int }{1})},
			wantMessage:     "{A:1}",
			wantSafeMessage: "{A:1}",
		},
		{
			// Marked Safe, so the error's own message is what is reported.
			name:            "safe error under an ordinary verb",
			format:          "%s",
			args:            []any{Safe(fs.ErrNotExist)},
			wantMessage:     fs.ErrNotExist.Error(),
			wantSafeMessage: fs.ErrNotExist.Error(),
		},
		{
			name:            "wrapping nothing keeps the verb",
			format:          "wrapping: %w",
			args:            []any{nilError},
			wantMessage:     "wrapping: %!w(<nil>)",
			wantSafeMessage: "wrapping: %w",
		},

		// A SafeStringer contributes its stand-in, whatever the verb.
		{
			name:            "stand-in under s verb",
			format:          "cannot update %s: %w",
			args:            []any{safeStringerKey("resources.jobs.my_job"), fs.ErrPermission},
			wantMessage:     "cannot update resources.jobs.my_job: " + fs.ErrPermission.Error(),
			wantSafeMessage: "cannot update jobs.*: %w",
		},
		{
			name:            "stand-in quoted by q verb like the value",
			format:          "%q not found",
			args:            []any{safeStringerKey("resources.jobs.my_job")},
			wantMessage:     `"resources.jobs.my_job" not found`,
			wantSafeMessage: `"jobs.*" not found`,
		},
		{
			name:            "stand-in alongside safe and unsafe args",
			format:          "cannot %s %s: field %q: %s",
			args:            []any{Safe("update"), safeStringerKey("resources.jobs.my_job"), Safe("tasks[0].job_id"), unsafeValue},
			wantMessage:     "cannot update resources.jobs.my_job: field \"tasks[0].job_id\": " + unsafeValue,
			wantSafeMessage: `cannot update jobs.*: field "tasks[0].job_id": %s`,
		},
		{
			// Safe cannot put a stand-in value's user-supplied part back in.
			name:            "stand-in outranks Safe",
			format:          "%s",
			args:            []any{Safe(safeStringerKey("resources.jobs.my_job"))},
			wantMessage:     "resources.jobs.my_job",
			wantSafeMessage: "jobs.*",
		},
		{
			// Safe must not cost a stand-in error its error-ness, or %w would have
			// a string to render and report %!w(string=...).
			name:            "Safe keeps a stand-in error usable by w verb",
			format:          "pushing state: %w",
			args:            []any{Safe(standInOnlyErr{})},
			wantMessage:     "pushing state: " + standInOnlyErr{}.Error(),
			wantSafeMessage: "pushing state: access denied",
		},
		{
			// An empty safe message is still one, so its verb is rendered, not
			// escaped.
			name:            "wrapped empty safe message",
			format:          "outer: %w",
			args:            []any{New("")},
			wantMessage:     "outer: ",
			wantSafeMessage: "outer: ",
		},
		{
			// An error with a safe message of its own prefers that to a stand-in.
			name:            "wrapped safe error beats a stand-in",
			format:          "outer: %w",
			args:            []any{Errorf("inner %d", Safe(1))},
			wantMessage:     "outer: inner 1",
			wantSafeMessage: "outer: inner 1",
		},
		{
			// A typed error with no safe message of its own falls back to its
			// stand-in, which is how libs/filer reports a classification.
			name:            "typed error stand-in under w verb",
			format:          "writing state: %w",
			args:            []any{standInOnlyErr{}},
			wantMessage:     "writing state: access denied: /Workspace/Users/a@b.com/x",
			wantSafeMessage: "writing state: access denied",
		},
		{
			name:            "typed error stand-in under s verb",
			format:          "writing state: %s",
			args:            []any{standInOnlyErr{}},
			wantMessage:     "writing state: access denied: /Workspace/Users/a@b.com/x",
			wantSafeMessage: "writing state: access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Errorf(tt.format, tt.args...)
			assert.Equal(t, tt.wantMessage, err.Error())
			assert.Equal(t, tt.wantSafeMessage, SafeError(err))
		})
	}

	// The same table drives SafeSprintf, which is what Errorf stores and is worth
	// exercising on its own rather than only through an error.
	for _, tt := range tests {
		t.Run("SafeSprintf/"+tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantSafeMessage, SafeSprintf(tt.format, tt.args...))
		})
	}

	// And the message half: Errorf must render exactly what fmt.Errorf would from
	// the same values, which is what makes converting a call site invisible.
	for _, tt := range tests {
		t.Run("MatchesFmt/"+tt.name, func(t *testing.T) {
			//nolint:govet // tt.format is a table value, so vet cannot check the verbs
			assert.Equal(t, tt.wantMessage, fmt.Errorf(tt.format, unpackArgs(tt.args)...).Error())
		})
	}

	// The security property of the package, over every row rather than a
	// hand-picked list: whatever the message carries, the safe message does not
	// carry the value that was not marked safe.
	for _, tt := range tests {
		t.Run("NoLeak/"+tt.name, func(t *testing.T) {
			err := Errorf(tt.format, tt.args...)
			safe := SafeError(err)
			for _, marker := range unsafeMarkers {
				if strings.Contains(err.Error(), marker) {
					assert.NotContains(t, safe, marker, "marker %q reached the safe message", marker)
				}
			}
		})
	}
}

func TestSafeErrorChainsEveryLevel(t *testing.T) {
	// The table covers what the outermost error renders; this is the part it
	// cannot express, that every level still reports its own safe message.
	inner := Errorf("cannot convert %s to %s", Safe("string"), Safe("int64"))
	middle := Errorf("%s: cannot set field %q: %w", unsafeValue, Safe("tasks[0].job_id"), inner)
	outer := Errorf("%s: SaveState: %w", unsafeValue, middle)

	assert.Equal(t, "cannot convert string to int64", SafeError(inner))
	assert.Equal(t, `%s: cannot set field "tasks[0].job_id": cannot convert string to int64`, SafeError(middle))
	assert.Equal(t, `%s: SaveState: %s: cannot set field "tasks[0].job_id": cannot convert string to int64`, SafeError(outer))
}

func TestSafeErrorChainsThroughPlainWrap(t *testing.T) {
	// A plain fmt.Errorf in the middle of the chain contributes nothing, so the
	// innermost template that is known reaches the top.
	inner := Errorf("cannot look up %q", Safe("continuous.pause_status"))
	err := fmt.Errorf("%s: %w", unsafeValue, inner)

	assert.Equal(t, `cannot look up "continuous.pause_status"`, SafeError(err))
}

func TestSafeErrorWithoutSafeErr(t *testing.T) {
	assert.Empty(t, SafeError(nil))
	assert.Empty(t, SafeError(errors.New(unsafeValue)))
	assert.Empty(t, SafeError(fmt.Errorf("reading %s", unsafeValue)))
	assert.Empty(t, SafeError(fs.ErrNotExist))
}

func TestNew(t *testing.T) {
	// New takes a literal message, so it is its own safe message and any verbs in
	// it are neither expanded nor lost.
	tests := []struct {
		name string
		text string
	}{
		{
			name: "plain",
			text: "state conversion failed",
		},
		{
			name: "contains verbs",
			text: "100% of %s attempts failed",
		},
		{
			name: "contains an escaped percent",
			text: "100%% done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.text)
			assert.Equal(t, tt.text, err.Error())
			assert.Equal(t, tt.text, SafeError(err))
		})
	}
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

	// Several %w in one call: each branch stays reachable.
	first := Errorf("group %s has no adapter", Safe("quality_monitors"))
	both := Errorf("%s: %w and %w", unsafeValue, first, fs.ErrPermission)
	assert.ErrorIs(t, both, first)
	assert.ErrorIs(t, both, fs.ErrPermission)
}

func TestErrorsAsType(t *testing.T) {
	err := Errorf("%s: %w", unsafeValue, &fs.PathError{Op: "open", Path: "/tmp/x", Err: fs.ErrNotExist})

	pathErr, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pathErr.Op)
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

func TestSafeErrorCapKeepsValidUTF8(t *testing.T) {
	// The cap counts bytes, so it can land inside a rune. The field still has to
	// be valid UTF-8.
	err := New(strings.Repeat("a", maxSafeErrorSize-1) + "é")

	safe := SafeError(err)
	assert.True(t, utf8.ValidString(safe), "safe message must be valid UTF-8")
	assert.Len(t, safe, maxSafeErrorSize-1, "the split rune is dropped, not half-kept")
}

func TestSafeErrorCyclicChainInArgumentTerminates(t *testing.T) {
	// The cycle is closed before Errorf is called, so walking the argument's chain
	// would not terminate. fmt.Errorf copes with such an error; so must Errorf.
	c := &cyclicErr{}
	c.inner = c

	err := Errorf("outer: %w", c)

	assert.Equal(t, "outer: cyclic", err.Error())
	assert.Equal(t, "outer: cyclic", SafeError(err))
}

func TestSafeErrorIsCapped(t *testing.T) {
	// A safe message is a telemetry field, so it cannot grow without bound even
	// though every part of it comes from source literals.
	err := New(strings.Repeat("x", maxSafeErrorSize*2))
	assert.Len(t, SafeError(err), maxSafeErrorSize)

	err = Errorf("%s", Safe(strings.Repeat("y", maxSafeErrorSize*2)))
	assert.Len(t, SafeError(err), maxSafeErrorSize)
}
