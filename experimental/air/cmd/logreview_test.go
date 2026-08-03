package aircmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsReviewErrorLine(t *testing.T) {
	for _, line := range []string{
		"RuntimeError: boom",
		"step failed after 3 tries",
		"NCCL timeout waiting for peer",
		"Traceback ... Exception",
		"UPPERCASE ERROR",
	} {
		assert.True(t, isReviewErrorLine(line), line)
	}

	for _, line := range []string{
		"epoch 3 loss 0.5",
		"all reduce complete",
		"",
	} {
		assert.False(t, isReviewErrorLine(line), line)
	}
}

func TestReviewLines(t *testing.T) {
	// Unset (negative) uses the default; an explicit --lines wins.
	assert.Equal(t, defaultReviewLines, logRequest{tailLines: -1}.reviewLines())
	assert.Equal(t, defaultReviewLines, logRequest{tailLines: 0}.reviewLines())
	assert.Equal(t, 25, logRequest{tailLines: 25}.reviewLines())
}

func TestTailFileLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	var b strings.Builder
	for i := range 10 {
		b.WriteString("line " + strconv.Itoa(i) + "\n")
	}
	require.NoError(t, os.WriteFile(path, []byte(b.String()), 0o644))

	// Fewer lines than the file has: the newest ones win.
	tail, err := tailFileLines(path, 3)
	require.NoError(t, err)
	assert.Equal(t, []string{"line 7", "line 8", "line 9"}, tail)

	// More lines than the file has: everything.
	tail, err = tailFileLines(path, 100)
	require.NoError(t, err)
	assert.Len(t, tail, 10)
}

func TestRenderReviewHighlightsErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node_0.log")
	require.NoError(t, os.WriteFile(path, []byte("step 1 ok\nRuntimeError: boom\nstep 2 ok\n"), 0o644))

	var buf bytes.Buffer
	renderReview(cmdio.MockDiscard(t.Context()), &buf, map[int]string{0: path}, 200)

	out := buf.String()
	// The box header names the node and the analyzed line count, and every line
	// is present (error highlighting is a style, stripped in non-color output).
	assert.Contains(t, out, "Node 0")
	assert.Contains(t, out, "RuntimeError: boom")
	assert.Contains(t, out, "step 1 ok")
}

func TestReviewLogsEndToEnd(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, fullDownloadServer(t).URL)

	var buf bytes.Buffer
	success, err := reviewLogs(ctx, w, &buf, logRequest{runID: 123, attempt: -1, tailLines: -1, review: true},
		logRunStatus{lifeCycleState: "TERMINATED", resultState: "SUCCESS"})
	require.NoError(t, err)
	assert.True(t, success)

	// Both nodes of the 2-node run are rendered.
	out := buf.String()
	assert.Contains(t, out, "Node 0")
	assert.Contains(t, out, "Node 1")
	assert.Contains(t, out, "hello")
}

func TestReviewLogsNonAirRun(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	// noMLflowServer's run carries no AI runtime compute, so node-count
	// resolution fails before any download is attempted.
	w := newTestWorkspaceClient(t, noMLflowServer(t).URL)

	_, err := reviewLogs(ctx, w, &bytes.Buffer{}, logRequest{runID: 5, attempt: -1, review: true},
		logRunStatus{lifeCycleState: "TERMINATED", resultState: "SUCCESS"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no AI runtime compute config")
}
