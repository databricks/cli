package aitools

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
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

// pendingInput returns the number of bytes typed at the terminal and not read
// yet.
func pendingInput(t *testing.T, terminal *os.File) int {
	n, err := unix.IoctlGetInt(int(terminal.Fd()), unix.TIOCINQ)
	require.NoError(t, err)
	return n
}

// An Enter pressed while recording the question waited for the state lock,
// before the question was shown, does not agree: it is discarded, and only
// the answer typed after the question counts.
func TestAskAtLoginIgnoresAnEnterPressedWhileTheStateLockIsHeld(t *testing.T) {
	home := setupAutoInstallHome(t, unaskedState)
	t.Cleanup(autoinstall.ResetDeclined)
	controller, terminal := openPseudoTerminal(t)
	var stderr lockedBuffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, terminal, cmdio.FakeTTY(io.Discard), cmdio.FakeTTY(&stderr), "", ""))

	// Another CLI process holds the state lock.
	lockFile, err := os.OpenFile(filepath.Join(home, ".databricks", "aitools", "auto-install.json.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	require.NoError(t, err)
	t.Cleanup(func() { _ = lockFile.Close() })
	require.NoError(t, unix.Flock(int(lockFile.Fd()), unix.LOCK_EX))

	// A line typed before the question starts tells when its first discard
	// ran.
	_, err = controller.WriteString("git status\n")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return pendingInput(t, terminal) > 0 }, 5*time.Second, time.Millisecond)

	answers := make(chan loginAnswer, 1)
	go func() { answers <- askConsent(ctx, []string{"Cursor"}) }()

	// The question now waits for the state lock, which it waits for up to a
	// second. Enter is pressed before the question is shown.
	require.Eventually(t, func() bool { return pendingInput(t, terminal) == 0 }, 5*time.Second, time.Millisecond)
	_, err = controller.WriteString("\n")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return pendingInput(t, terminal) > 0 }, 5*time.Second, time.Millisecond)
	require.NotContains(t, stderr.String(), cursorQuestion)
	require.NoError(t, unix.Flock(int(lockFile.Fd()), unix.LOCK_UN))

	// Once the question is shown, the Enter is discarded and the user
	// declines.
	require.Eventually(t, func() bool { return strings.Contains(stderr.String(), cursorQuestion) }, 5*time.Second, time.Millisecond,
		"the question was not shown; did its wait for the state lock run out?")
	require.Eventually(t, func() bool { return pendingInput(t, terminal) == 0 }, 5*time.Second, time.Millisecond)
	_, err = controller.WriteString("n\n")
	require.NoError(t, err)

	var got loginAnswer
	select {
	case got = <-answers:
	case <-time.After(10 * time.Second):
		t.Fatal("the question was not answered")
	}
	require.True(t, got.asked)
	assert.Equal(t, autoinstall.ConsentOptedOut, got.answer)
	assert.True(t, got.stored)
	s, err := autoinstall.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{OptedOut: true, ConsentAsked: true, ID: "treatment-machine"}, s)
}
