package aircmd

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
)

// An OSC set-title, a CSI clear-screen and a carriage return in one value, and
// how it is shown.
const (
	controlPayload = "x\x1b]0;pwned\x07y\x1b[2Jz\rw"
	controlEscaped = `x\u001b]0;pwned\u0007y\u001b[2Jz\rw`
)

func assertControlEscaped(t *testing.T, out string) {
	t.Helper()
	assert.NotContains(t, out, "\x1b")
	assert.NotContains(t, out, "\x07")
	assert.NotContains(t, out, "\r")
	assert.Contains(t, out, controlEscaped)
}

func TestOSC8(t *testing.T) {
	assert.Equal(t, "\x1b]8;;https://h.test/a\x1b\\label\x1b]8;;\x1b\\", osc8("https://h.test/a", "label"))
	// A control character in the target would end the sequence early.
	assert.Equal(t, "label", osc8("https://h.test/a\x07\x1b[2J", "label"))
	assert.Equal(t, "label", osc8("https://h.test/a\x1b\\", "label"))
	// A raw C1 byte is a control character to a terminal in 8-bit mode.
	assert.Equal(t, "label", osc8("https://h.test/a\x9c", "label"))
}

func TestColorizeConfigEscapesControlCharacters(t *testing.T) {
	// A raw ESC makes the YAML parser reject the config, so it is shown as stored.
	body := reformatYAMLForDisplay([]byte("command: |\n  echo " + controlPayload + "\n\techo done\n"))
	out := colorizeConfig(asciiPalette(t), body)
	assertControlEscaped(t, out)
	assert.Contains(t, out, "echo done")
}

func TestEmitNoLogsEscapesControlCharacters(t *testing.T) {
	status := logRunStatus{lifeCycleState: "TERMINATED", resultState: "FAIL\x1b[2JED", stateMessage: controlPayload}

	var text bytes.Buffer
	emitNoLogs(&text, logRequest{runID: 1, attempt: -1}, status)
	assertControlEscaped(t, text.String())
	assert.Contains(t, text.String(), `in state FAIL\u001b[2JED`)

	// JSON output keeps the message as the server sent it, escaped by the encoder.
	var js bytes.Buffer
	emitNoLogs(&js, logRequest{runID: 1, attempt: -1, jsonOutput: true}, status)
	assert.Contains(t, js.String(), `x\u001b]0;pwned\u0007y\u001b[2Jz\rw`)
}

func TestDisplayCancelPreviewEscapesControlCharacters(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	displayCancelPreview(ctx, []listRow{{RunID: "1", Experiment: controlPayload + "\n2\tfake"}}, map[string][]string{"1": {"train\n3\tfake"}}, "https://h.test")
	assertControlEscaped(t, stderr.String())
	assert.Contains(t, stderr.String(), controlEscaped+`\n2\tfake`)
	assert.Contains(t, stderr.String(), `train\n3\tfake`)
}

func TestRenderPoolsEscapeControlCharacters(t *testing.T) {
	var table bytes.Buffer
	renderPoolTable(&table, []poolRow{{ID: controlPayload, AcceleratorType: controlPayload}})
	assertControlEscaped(t, table.String())

	var detail bytes.Buffer
	renderPoolDetail(&detail, poolDetailData{ID: controlPayload, AcceleratorType: controlPayload})
	assertControlEscaped(t, detail.String())
}
