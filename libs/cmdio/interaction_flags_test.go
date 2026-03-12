package cmdio

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stringer is a simple fmt.Stringer implementation for tests.
type stringer string

func (s stringer) String() string { return string(s) }

// newTestContext creates a cmdio context with specified capabilities and captured stderr.
func newTestContext(t *testing.T, caps Capabilities) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities:   caps,
		outputFormat:   flags.OutputText,
		headerTemplate: "",
		template:       "",
		in:             io.NopCloser(strings.NewReader("")),
		out:            stdout,
		err:            stderr,
	}
	InContext(ctx, cmdIO)
	return stdout, stderr
}

// --- Quiet mode tests ---

func TestLogStringSuppressedWhenQuiet(t *testing.T) {
	ctx := t.Context()
	stderr := &bytes.Buffer{}
	cmdIO := &cmdIO{
		capabilities: Capabilities{quiet: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          stderr,
	}
	ctx = InContext(ctx, cmdIO)

	LogString(ctx, "should be suppressed")
	assert.Empty(t, stderr.String())
}

func TestLogStringShownWhenNotQuiet(t *testing.T) {
	ctx := t.Context()
	stderr := &bytes.Buffer{}
	cmdIO := &cmdIO{
		capabilities: Capabilities{quiet: false},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          stderr,
	}
	ctx = InContext(ctx, cmdIO)

	LogString(ctx, "hello")
	assert.Equal(t, "hello\n", stderr.String())
}

func TestLogSuppressedWhenQuiet(t *testing.T) {
	ctx := t.Context()
	stderr := &bytes.Buffer{}
	cmdIO := &cmdIO{
		capabilities: Capabilities{quiet: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          stderr,
	}
	ctx = InContext(ctx, cmdIO)

	Log(ctx, stringer("should be suppressed"))
	assert.Empty(t, stderr.String())
}

func TestSpinnerNoOpWhenQuiet(t *testing.T) {
	ctx := t.Context()
	stderr := &bytes.Buffer{}
	cmdIO := &cmdIO{
		capabilities: Capabilities{
			quiet:       true,
			stderrIsTTY: true,
			color:       true,
		},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          stderr,
	}
	ctx = InContext(ctx, cmdIO)

	sp := NewSpinner(ctx)
	assert.Nil(t, sp.p) // Should be no-op (nil program)
	sp.Update("test")
	sp.Close()
}

func TestRenderDiagnosticsQuietFiltersNonErrors(t *testing.T) {
	ctx := t.Context()
	stderr := &bytes.Buffer{}
	cmdIO := &cmdIO{
		capabilities: Capabilities{quiet: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          stderr,
	}
	ctx = InContext(ctx, cmdIO)

	diags := diag.Diagnostics{
		{Severity: diag.Error, Summary: "error message"},
		{Severity: diag.Warning, Summary: "warning message"},
		{Severity: diag.Recommendation, Summary: "recommendation message"},
	}

	err := RenderDiagnostics(ctx, diags)
	require.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "error message")
	assert.NotContains(t, output, "warning message")
	assert.NotContains(t, output, "recommendation message")
}

func TestRenderDiagnosticsNonQuietShowsAll(t *testing.T) {
	ctx := t.Context()
	stderr := &bytes.Buffer{}
	cmdIO := &cmdIO{
		capabilities: Capabilities{quiet: false},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          stderr,
	}
	ctx = InContext(ctx, cmdIO)

	diags := diag.Diagnostics{
		{Severity: diag.Error, Summary: "error message"},
		{Severity: diag.Warning, Summary: "warning message"},
	}

	err := RenderDiagnostics(ctx, diags)
	require.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, "warning message")
}

func TestIsQuietAndSetQuiet(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	assert.False(t, IsQuiet(ctx))
	SetQuiet(ctx)
	assert.True(t, IsQuiet(ctx))
}

// --- No-input mode tests ---

func TestIsNoInputAndSetNoInput(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	assert.False(t, IsNoInput(ctx))
	SetNoInput(ctx)
	assert.True(t, IsNoInput(ctx))
}

func TestSupportsPromptFalseWhenNoInput(t *testing.T) {
	caps := Capabilities{
		stdinIsTTY:  true,
		stdoutIsTTY: true,
		stderrIsTTY: true,
		color:       true,
		isGitBash:   false,
		noInput:     true,
	}
	assert.False(t, caps.SupportsPrompt())
}

func TestAskReturnsDefaultWhenNoInput(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	result, err := Ask(ctx, "Enter value", "default_value")
	require.NoError(t, err)
	assert.Equal(t, "default_value", result)
}

func TestAskReturnsErrNoInputWhenNoDefault(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	_, err := Ask(ctx, "Enter value", "")
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestAskYesOrNoReturnsErrNoInputWhenNoInput(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	_, err := AskYesOrNo(ctx, "Continue?")
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestAskSelectReturnsErrNoInputWhenNoInput(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	_, err := AskSelect(ctx, "Choose", []string{"a", "b"})
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestSelectReturnsErrNoInputWhenNoInput(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	items := map[string]string{"item1": "1", "item2": "2"}
	_, err := Select(ctx, items, "Choose item")
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestSecretReturnsErrNoInputWhenNoInput(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	_, err := Secret(ctx, "Enter secret")
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestRunSelectReturnsErrNoInputWhenNoInput(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	_, _, err := RunSelect(ctx, nil)
	assert.ErrorIs(t, err, ErrNoInput)
}

// --- Yes mode tests ---

func TestIsYesAndSetYes(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	assert.False(t, IsYes(ctx))
	SetYes(ctx)
	assert.True(t, IsYes(ctx))
}

func TestAskYesOrNoReturnsTrueWhenYes(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{yes: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	result, err := AskYesOrNo(ctx, "Continue?")
	require.NoError(t, err)
	assert.True(t, result)
}

// --- Precedence tests ---

func TestAskYesOrNoYesTakesPrecedenceOverNoInput(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{yes: true, noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	// --yes should take precedence over --no-input for yes/no prompts
	result, err := AskYesOrNo(ctx, "Continue?")
	require.NoError(t, err)
	assert.True(t, result)
}

func TestAskStillReturnsErrNoInputEvenWithYes(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{yes: true, noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	// --yes does not affect Ask() (free-form input), --no-input still blocks it
	_, err := Ask(ctx, "Enter value", "")
	assert.ErrorIs(t, err, ErrNoInput)
}

func TestAskSelectStillReturnsErrNoInputEvenWithYes(t *testing.T) {
	ctx := t.Context()
	cmdIO := &cmdIO{
		capabilities: Capabilities{yes: true, noInput: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          io.Discard,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	// --yes does not affect AskSelect(), --no-input still blocks it
	_, err := AskSelect(ctx, "Choose", []string{"a", "b"})
	assert.ErrorIs(t, err, ErrNoInput)
}

// --- Render output is not affected by quiet ---

func TestRenderDataOutputNotAffectedByQuiet(t *testing.T) {
	ctx := t.Context()
	stdout := &bytes.Buffer{}
	cmdIO := &cmdIO{
		capabilities: Capabilities{quiet: true},
		outputFormat: flags.OutputText,
		in:           io.NopCloser(strings.NewReader("")),
		out:          stdout,
		err:          io.Discard,
	}
	ctx = InContext(ctx, cmdIO)

	data := map[string]string{"key": "value"}
	err := Render(ctx, data)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "value")
}
