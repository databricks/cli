package cmdio

import (
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// openPseudoTerminal returns the two ends of a new pseudo-terminal: what is
// written to the controller is typed at the terminal.
func openPseudoTerminal(t *testing.T) (controller, terminal *os.File) {
	controller, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal: %v", err)
	}
	t.Cleanup(func() { _ = controller.Close() })
	require.NoError(t, unix.IoctlSetPointerInt(int(controller.Fd()), unix.TIOCSPTLCK, 0))
	n, err := unix.IoctlGetInt(int(controller.Fd()), unix.TIOCGPTN)
	require.NoError(t, err)
	terminal, err = os.OpenFile("/dev/pts/"+strconv.Itoa(n), os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = terminal.Close() })
	return controller, terminal
}

// waitForInput waits until a line typed at the terminal can be read.
func waitForInput(t *testing.T, terminal *os.File) {
	fds := []unix.PollFd{{Fd: int32(terminal.Fd()), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, 5000)
	require.NoError(t, err)
	require.Equal(t, 1, n, "the typed line did not arrive")
}

func TestDiscardInput(t *testing.T) {
	controller, terminal := openPseudoTerminal(t)
	ctx := InContext(t.Context(), NewIO(t.Context(), flags.OutputText, terminal, io.Discard, io.Discard, "", ""))

	_, err := controller.WriteString("typed ahead\n")
	require.NoError(t, err)
	waitForInput(t, terminal)
	require.NoError(t, DiscardInput(ctx))

	_, err = controller.WriteString("answer\n")
	require.NoError(t, err)
	waitForInput(t, terminal)
	line, err := readLine(terminal)
	require.NoError(t, err)
	assert.Equal(t, "answer", line)
}

// pendingInput returns the number of bytes typed at the terminal and not read
// yet.
func pendingInput(t *testing.T, terminal *os.File) int {
	n, err := unix.IoctlGetInt(int(terminal.Fd()), unix.TIOCINQ)
	require.NoError(t, err)
	return n
}

// What was typed before the question was printed does not answer it, even
// when nothing discarded it before the question.
func TestAskCancellableAfterQuestion(t *testing.T) {
	controller, terminal := openPseudoTerminal(t)
	ctx := InContext(t.Context(), NewIO(t.Context(), flags.OutputText, terminal, io.Discard, io.Discard, "", ""))
	_, err := controller.WriteString("typed ahead\n")
	require.NoError(t, err)
	waitForInput(t, terminal)

	type answer struct {
		line string
		err  error
	}
	answers := make(chan answer, 1)
	go func() {
		line, err := AskCancellableAfterQuestion(ctx, "Question", "")
		answers <- answer{line, err}
	}()
	// The line typed ahead is discarded once the question is printed.
	require.Eventually(t, func() bool { return pendingInput(t, terminal) == 0 }, 5*time.Second, time.Millisecond)
	_, err = controller.WriteString("answer\n")
	require.NoError(t, err)

	select {
	case a := <-answers:
		require.NoError(t, a.err)
		assert.Equal(t, "answer", a.line)
	case <-time.After(5 * time.Second):
		t.Fatal("the question was not answered")
	}
}

// A pipe is not a terminal: nothing is discarded.
func TestDiscardInputOfAPipe(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	ctx := InContext(t.Context(), NewIO(t.Context(), flags.OutputText, r, io.Discard, io.Discard, "", ""))

	_, err = w.WriteString("typed ahead\n")
	require.NoError(t, err)
	require.NoError(t, DiscardInput(ctx))
	line, err := readLine(r)
	require.NoError(t, err)
	assert.Equal(t, "typed ahead", line)
}
